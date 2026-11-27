package jobs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/dbmanager"
	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/observability"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// BatchProgress and BatchStep are defined next to the checkpoint table in dbmanager.
type (
	BatchProgress = dbmanager.BatchProgress
	BatchStep     = dbmanager.BatchStep
)

// BatchExecutor is the optional part of an Executor that supports batch jobs.
// It is a separate interface so an Executor without batch support still compiles;
// the worker fails a batch job with a clear error in that case.
type BatchExecutor interface {
	// Progress returns what is already committed for this job in the target
	// database (the checkpoint table is the source of truth).
	Progress(ctx context.Context, job Job) (BatchProgress, error)
	// ExecuteBatch runs one batch in its own transaction and writes its
	// checkpoint in the same transaction.
	ExecuteBatch(ctx context.Context, job Job, batchNo int, cursor sql.NullString) (BatchStep, error)
	// Complete records the migration as successful in the history. Idempotent.
	Complete(ctx context.Context, job Job, version int64) error
}

func (e *ManagerExecutor) Progress(ctx context.Context, job Job) (BatchProgress, error) {
	m, err := e.Registry.Get(ctx, job.DatabaseID)
	if err != nil {
		return BatchProgress{}, err
	}
	return m.BatchProgress(ctx, job.ID)
}

func (e *ManagerExecutor) ExecuteBatch(ctx context.Context, job Job, batchNo int, cursor sql.NullString) (BatchStep, error) {
	m, err := e.Registry.Get(ctx, job.DatabaseID)
	if err != nil {
		return BatchStep{}, err
	}
	if job.BatchByAttribute() {
		return m.ExecuteAttributeBatch(ctx, job.ID, batchNo, job.SQL, job.BatchTable, job.BatchColumn, cursor, job.BatchSize)
	}
	return m.ExecuteBatch(ctx, job.ID, batchNo, job.SQL, cursor, job.BatchSize)
}

// BatchByAttribute reports whether the platform builds the cursor logic from the
// batch table and column (true), or the user's SQL does it with $1 and $2 (false).
func (j Job) BatchByAttribute() bool {
	return j.BatchTable != "" && j.BatchColumn != ""
}

func (e *ManagerExecutor) Complete(ctx context.Context, job Job, version int64) error {
	m, err := e.Registry.Get(ctx, job.DatabaseID)
	if err != nil {
		return err
	}
	return m.RecordMigrationSuccessDefinitionWithMetadata(ctx, version, job.SQL, job.SQLDown, job.Irreversible, job.IrreversibleReason, job.RequestedBy, migrationMetadata(job))
}

// validateBatchRequest checks the mode and batch settings of an enqueue request.
//
// Two ways to write a batch migration:
//   - by attribute: table + column are given, and the SQL contains {{batch_filter}};
//   - custom cursor: no table/column, and the SQL uses $1 (cursor) and $2 (size).
func validateBatchRequest(mode string, size int, sqlText, table, column string) error {
	switch mode {
	case ModePlain:
		return nil
	case ModeBatch:
		if size < 1 || size > MaxBatchSize {
			return fmt.Errorf("batch size must be between 1 and %d", MaxBatchSize)
		}
		if table != "" || column != "" {
			if table == "" || column == "" {
				return errors.New("batch by attribute needs both a table and a column")
			}
			if strings.Contains(column, ".") {
				return errors.New("batch column must be a plain column name (no dots)")
			}
			if strings.Count(table, ".") > 1 {
				return errors.New(`batch table must be "table" or "schema.table"`)
			}
			if !strings.Contains(sqlText, dbmanager.BatchFilterPlaceholder) {
				return fmt.Errorf("batch SQL must contain %s, for example: UPDATE %s SET ... WHERE %s",
					dbmanager.BatchFilterPlaceholder, table, dbmanager.BatchFilterPlaceholder)
			}
			return nil
		}
		if !strings.Contains(sqlText, "$1") || !strings.Contains(sqlText, "$2") {
			return errors.New("batch SQL must use $1 (cursor after the previous batch) and $2 (batch size), or give a batch table and column and use " + dbmanager.BatchFilterPlaceholder)
		}
		return nil
	default:
		return fmt.Errorf("unknown migration mode %q", mode)
	}
}

// ValidateBatchRequest exposes batch input validation to the approval form so
// malformed jobs are rejected before a leader is asked to approve them.
func ValidateBatchRequest(mode string, size int, sqlText, table, column string) error {
	return validateBatchRequest(mode, size, sqlText, table, column)
}

// processBatch runs a batch job: batch after batch, each in its own transaction,
// continuing from the last committed checkpoint. Only the cursor and counters
// are kept in memory, never the dataset.
func (w *Worker) processBatch(ctx, jobCtx context.Context, job Job) {
	tracer := observability.Tracer("worker")

	be, ok := w.exec.(BatchExecutor)
	if !ok {
		w.fail(ctx, job, fmt.Errorf("%w: executor does not support batch mode", ErrValidation))
		return
	}

	// Where did the previous attempt stop? Read it from the target database.
	progress, err := be.Progress(jobCtx, job)
	if err != nil {
		w.finishBatch(ctx, jobCtx, job, BatchProgress{}, err)
		return
	}

	// Keep the version of the first attempt: committed batches already belong to it.
	version := job.Version.Int64
	if !job.Version.Valid {
		version, err = w.exec.NextVersion(jobCtx, job.DatabaseID)
		if err == nil {
			err = w.store.SetVersion(jobCtx, job.ID, version)
		}
		if err != nil {
			w.finishBatch(ctx, jobCtx, job, progress, err)
			return
		}
		job.Version = sql.NullInt64{Int64: version, Valid: true}
	}

	execCtx, execSpan := tracer.Start(jobCtx, "migration.execute", trace.WithAttributes(
		attribute.Int64("migration.version", version),
		attribute.String("migration.mode", ModeBatch),
		attribute.Int("batch.size", job.BatchSize),
		attribute.Int("batch.resume_from", progress.Batches),
	))
	slog.InfoContext(execCtx, "migration.started", "job_id", job.ID, "version", version,
		"mode", ModeBatch, "batch_size", job.BatchSize, "committed_batches", progress.Batches)
	started := time.Now()

	runErr := w.runBatches(execCtx, job, be, &progress, tracer)
	if runErr == nil {
		runErr = be.Complete(execCtx, job, version)
	}
	if runErr != nil {
		observability.RecordError(execSpan, runErr)
	}
	execSpan.SetAttributes(attribute.Int("batch.committed", progress.Batches),
		attribute.Int64("batch.rows", progress.Rows))
	execSpan.End()
	slog.InfoContext(ctx, "migration.finished", "job_id", job.ID, "version", version,
		"mode", ModeBatch, "duration_ms", time.Since(started).Milliseconds(),
		"batches", progress.Batches, "rows", progress.Rows, "ok", runErr == nil)

	w.finishBatch(ctx, jobCtx, job, progress, runErr)
}

// runBatches loops until a batch reports nothing left, or an error occurs.
// progress is updated after every committed batch, also when an error follows.
func (w *Worker) runBatches(ctx context.Context, job Job, be BatchExecutor, progress *BatchProgress, tracer trace.Tracer) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		batchNo := progress.Batches + 1
		bctx, span := tracer.Start(ctx, "batch", trace.WithAttributes(attribute.Int("batch.no", batchNo)))
		started := time.Now()

		step, err := be.ExecuteBatch(bctx, job, batchNo, progress.Cursor)
		if err != nil {
			observability.RecordError(span, err)
			span.End()
			slog.WarnContext(bctx, "batch.failed", "job_id", job.ID, "batch_no", batchNo,
				"sqlstate", SQLState(err), "error", err)
			return err
		}
		if step.Done {
			span.SetAttributes(attribute.Bool("batch.done", true))
			span.End()
			return nil
		}

		progress.Batches++
		progress.Rows += step.Rows
		progress.Cursor = step.Cursor
		span.SetAttributes(attribute.Int64("batch.rows", step.Rows))
		span.End()
		slog.InfoContext(bctx, "batch.committed", "job_id", job.ID, "batch_no", batchNo,
			"rows", step.Rows, "cursor", step.Cursor.String, "total_batches", progress.Batches,
			"total_rows", progress.Rows, "duration_ms", time.Since(started).Milliseconds())

		// The mirror is only for the API; a failed update must not stop the migration.
		if err := w.store.UpdateProgress(ctx, job.ID, progress.Batches, progress.Rows); err != nil {
			slog.WarnContext(ctx, "job.progress_update_failed", "job_id", job.ID, "error", err)
		}
	}
}

// finishBatch records the outcome of a batch job.
func (w *Worker) finishBatch(ctx, jobCtx context.Context, job Job, p BatchProgress, runErr error) {
	sctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
	defer cancel()

	if p.Batches > 0 {
		if err := w.store.UpdateProgress(sctx, job.ID, p.Batches, p.Rows); err != nil {
			slog.WarnContext(ctx, "job.progress_update_failed", "job_id", job.ID, "error", err)
		}
		job.BatchesCommitted, job.RowsCommitted = p.Batches, p.Rows
	}

	switch {
	case runErr == nil:
		if err := w.store.MarkSuccess(sctx, job.ID); err != nil {
			slog.ErrorContext(ctx, "job.state_update_failed", "job_id", job.ID, "error", err)
			return
		}
		slog.InfoContext(ctx, "job.finished", "job_id", job.ID, "status", StatusSuccess,
			"batches", p.Batches, "rows", p.Rows)
		w.updateWorkflowStatus(ctx, job, "success", "Batch migration completed")

	case jobCtx.Err() != nil:
		if parentDone(ctx) {
			// Graceful shutdown: committed batches stay committed; give the job
			// back without using up an attempt.
			_ = w.store.Requeue(sctx, job.ID)
			slog.InfoContext(ctx, "job.requeued", "job_id", job.ID, "batches", p.Batches)
		} else {
			slog.WarnContext(ctx, "job.abandoned", "job_id", job.ID)
		}

	case (IsTransient(runErr) || IsCommitUncertain(runErr)) && job.Attempt < job.MaxAttempts:
		// Resuming is always safe: the checkpoints in the target tell what committed.
		w.retry(sctx, ctx, job, runErr)

	default:
		w.stopBatch(sctx, ctx, job, runErr)
	}
}

// stopBatch ends a batch job that cannot continue on its own: 'partial' when at
// least one batch is committed (the user can resume), 'failed' otherwise.
func (w *Worker) stopBatch(sctx, logCtx context.Context, job Job, cause error) {
	w.recordFailure(sctx, job, cause)
	if job.BatchesCommitted == 0 {
		w.fail(sctx, job, cause)
		return
	}
	msg := observability.Redact(cause.Error())
	if err := w.store.MarkPartial(sctx, job.ID, msg, SQLState(cause)); err != nil {
		slog.ErrorContext(logCtx, "job.state_update_failed", "job_id", job.ID, "error", err)
		return
	}
	slog.ErrorContext(logCtx, "job.partial", "job_id", job.ID, "batches", job.BatchesCommitted,
		"rows", job.RowsCommitted, "sqlstate", SQLState(cause), "error", cause)
	w.updateWorkflowStatus(logCtx, job, "partial", msg)
}

func (w *Worker) recordFailure(ctx context.Context, job Job, cause error) {
	recorder, ok := w.exec.(FailureRecorder)
	if !ok {
		return
	}
	version := int64(0)
	if job.Version.Valid {
		version = job.Version.Int64
	}
	if err := recorder.RecordFailure(ctx, job, version, cause); err != nil {
		slog.ErrorContext(ctx, "migration.failure_history_record_failed", "job_id", job.ID, "error", err)
	}
}
