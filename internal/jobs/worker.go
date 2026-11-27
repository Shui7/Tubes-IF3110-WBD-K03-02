package jobs

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/observability"
	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/workflow"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type WorkerConfig struct {
	ID                string
	Concurrency       int           // jobs processed at the same time (default 4)
	PollInterval      time.Duration // default 1s
	HeartbeatInterval time.Duration // default 5s
	StaleAfter        time.Duration // heartbeat age after which a job is considered orphaned (default 30s)
	BaseBackoff       time.Duration // first retry delay (default 2s), doubled per attempt
}

type Worker struct {
	cfg       WorkerConfig
	store     *Store
	exec      Executor
	validator Validator
}

func NewWorker(store *Store, exec Executor, validator Validator, cfg WorkerConfig) *Worker {
	if cfg.ID == "" {
		cfg.ID = "worker-" + newID()[:8]
	}
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 4
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = time.Second
	}
	if cfg.HeartbeatInterval <= 0 {
		cfg.HeartbeatInterval = 5 * time.Second
	}
	if cfg.StaleAfter <= 0 {
		cfg.StaleAfter = 30 * time.Second
	}
	if cfg.BaseBackoff <= 0 {
		cfg.BaseBackoff = 2 * time.Second
	}
	if validator == nil {
		validator = NoopValidator{}
	}
	return &Worker{cfg: cfg, store: store, exec: exec, validator: validator}
}

// Run polls the queue until ctx is cancelled, then waits for running jobs.
func (w *Worker) Run(ctx context.Context) error {
	slog.InfoContext(ctx, "worker.started", "worker_id", w.cfg.ID, "concurrency", w.cfg.Concurrency)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() { defer wg.Done(); w.reaperLoop(ctx) }()

	sem := make(chan struct{}, w.cfg.Concurrency)
	ticker := time.NewTicker(w.cfg.PollInterval)
	defer ticker.Stop()

	for {
		for w.tryStart(ctx, sem, &wg) {
		}
		select {
		case <-ctx.Done():
			wg.Wait()
			slog.Info("worker.stopped", "worker_id", w.cfg.ID)
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

// tryStart claims one job if a slot is free. It reports whether a job was started.
func (w *Worker) tryStart(ctx context.Context, sem chan struct{}, wg *sync.WaitGroup) bool {
	if ctx.Err() != nil {
		return false
	}
	select {
	case sem <- struct{}{}:
	default:
		return false // all slots busy
	}

	claimStart := time.Now()
	job, err := w.store.Claim(ctx, w.cfg.ID)
	claimEnd := time.Now()
	if err != nil {
		<-sem
		if ctx.Err() == nil {
			slog.ErrorContext(ctx, "job.claim_failed", "worker_id", w.cfg.ID, "error", err)
		}
		return false
	}
	if job == nil {
		<-sem
		return false
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() { <-sem }()
		w.process(ctx, *job, claimStart, claimEnd)
	}()
	return true
}

func (w *Worker) process(parent context.Context, job Job, claimStart, claimEnd time.Time) {
	// Continue the trace started by the API request that enqueued the job.
	ctx := observability.ContextWithTraceparent(parent, job.Traceparent)
	ctx = observability.WithRequestID(ctx, job.RequestID)
	tracer := observability.Tracer("worker")
	ctx, span := tracer.Start(ctx, "job.process", trace.WithAttributes(
		attribute.String("job.id", job.ID),
		attribute.Int("database.id", job.DatabaseID),
		attribute.Int("job.attempt", job.Attempt),
		attribute.String("job.kind", job.Kind),
	))
	defer span.End()

	// The claim already happened; record it as a span with its real timing.
	_, lockSpan := tracer.Start(ctx, "lock.acquire", trace.WithTimestamp(claimStart))
	lockSpan.End(trace.WithTimestamp(claimEnd))
	slog.InfoContext(ctx, "lock.acquired", "job_id", job.ID, "database_id", job.DatabaseID,
		"worker_id", w.cfg.ID, "attempt", job.Attempt)

	jobCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go w.heartbeat(jobCtx, job.ID, cancel)
	if job.WorkflowRequestID.Valid {
		if err := workflow.SetStatus(w.store.db, job.WorkflowRequestID.Int64, "running", 0, w.cfg.ID, "Execution started by queue worker"); err != nil {
			slog.ErrorContext(ctx, "migration.workflow_status_update_failed", "job_id", job.ID, "status", "running", "error", err)
			w.fail(ctx, job, err)
			return
		}
	}

	// Revalidate right before running.
	_, valSpan := tracer.Start(jobCtx, "revalidation")
	if err := w.validator.Validate(jobCtx, job); err != nil {
		observability.RecordError(valSpan, err)
		valSpan.End()
		slog.WarnContext(ctx, "revalidation.result", "job_id", job.ID, "ok", false, "error", err)
		w.fail(ctx, job, err)
		return
	}
	valSpan.End()
	slog.InfoContext(ctx, "revalidation.result", "job_id", job.ID, "ok", true)

	if job.Mode == ModeBatch {
		w.processBatch(ctx, jobCtx, job)
		return
	}

	var version int64
	var err error
	if job.Kind == "up" || job.Kind == "" {
		version, err = w.exec.NextVersion(jobCtx, job.DatabaseID)
		if err == nil {
			// Persist the version first: crash recovery needs it to find evidence.
			err = w.store.SetVersion(jobCtx, job.ID, version)
		}
	} else if job.Version.Valid {
		version = job.Version.Int64
	} else {
		err = errors.New("queued rollback/forward job has no migration version")
	}
	if err != nil {
		w.finish(ctx, jobCtx, job, err)
		return
	}
	job.Version = sql.NullInt64{Int64: version, Valid: true}

	execCtx, execSpan := tracer.Start(jobCtx, "migration.execute",
		trace.WithAttributes(attribute.Int64("migration.version", version)))
	slog.InfoContext(execCtx, "migration.started", "job_id", job.ID, "version", version)
	started := time.Now()
	execErr := w.exec.Execute(execCtx, job, version)
	if execErr != nil {
		observability.RecordError(execSpan, execErr)
	}
	execSpan.End()
	slog.InfoContext(ctx, "migration.finished", "job_id", job.ID, "version", version,
		"duration_ms", time.Since(started).Milliseconds(), "ok", execErr == nil)

	w.finish(ctx, jobCtx, job, execErr)
}

// finish records the outcome. State updates use a context that survives
// cancellation so that a shutdown cannot leave the job in 'running'.
func (w *Worker) finish(ctx, jobCtx context.Context, job Job, execErr error) {
	sctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
	defer cancel()

	switch {
	case execErr == nil:
		if err := w.store.MarkSuccess(sctx, job.ID); err != nil {
			slog.ErrorContext(ctx, "job.state_update_failed", "job_id", job.ID, "error", err)
			return
		}
		slog.InfoContext(ctx, "job.finished", "job_id", job.ID, "status", StatusSuccess)
		w.updateWorkflowStatus(ctx, job, "success", "Execution completed")

	case IsCommitUncertain(execErr):
		slog.WarnContext(ctx, "job.uncertain", "job_id", job.ID, "error", execErr)
		_ = w.store.MarkUncertain(sctx, job.ID, observability.Redact(execErr.Error()))
		w.resolveUncertain(sctx, ctx, job)

	case jobCtx.Err() != nil:
		if parentDone(ctx) {
			// Graceful shutdown: give the job back without using up an attempt.
			_ = w.store.Requeue(sctx, job.ID)
			slog.InfoContext(ctx, "job.requeued", "job_id", job.ID)
		} else {
			// Heartbeat lost: the reaper owns this job now. Do not touch its state.
			slog.WarnContext(ctx, "job.abandoned", "job_id", job.ID)
		}

	case IsTransient(execErr) && job.Attempt < job.MaxAttempts:
		w.retry(sctx, ctx, job, execErr)

	default:
		w.fail(sctx, job, execErr)
	}
}

func parentDone(ctx context.Context) bool { return ctx.Err() != nil }

func (w *Worker) retry(sctx, logCtx context.Context, job Job, cause error) {
	delay := w.backoff(job.Attempt)
	msg := observability.Redact(cause.Error())
	if err := w.store.ScheduleRetry(sctx, job.ID, delay, msg, SQLState(cause)); err != nil {
		slog.ErrorContext(logCtx, "job.state_update_failed", "job_id", job.ID, "error", err)
		return
	}
	slog.WarnContext(logCtx, "job.retry_scheduled", "job_id", job.ID, "attempt", job.Attempt,
		"max_attempts", job.MaxAttempts, "delay_ms", delay.Milliseconds(),
		"sqlstate", SQLState(cause), "error", cause)
}

func (w *Worker) fail(ctx context.Context, job Job, cause error) {
	sctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
	defer cancel()
	msg := observability.Redact(cause.Error())
	if err := w.store.MarkFailed(sctx, job.ID, msg, SQLState(cause)); err != nil {
		slog.ErrorContext(ctx, "job.state_update_failed", "job_id", job.ID, "error", err)
		return
	}
	slog.ErrorContext(ctx, "job.failed", "job_id", job.ID, "sqlstate", SQLState(cause), "error", cause)
	w.updateWorkflowStatus(ctx, job, "failed", msg)
}

func (w *Worker) updateWorkflowStatus(ctx context.Context, job Job, status, note string) {
	if !job.WorkflowRequestID.Valid {
		return
	}
	if err := workflow.SetStatus(w.store.db, job.WorkflowRequestID.Int64, status, 0, w.cfg.ID, note); err != nil {
		slog.ErrorContext(ctx, "migration.workflow_status_update_failed", "job_id", job.ID, "request_id", job.WorkflowRequestID.Int64, "status", status, "error", err)
	}
}

// resolveUncertain decides what happened to a job whose COMMIT result is unknown.
func (w *Worker) resolveUncertain(sctx, logCtx context.Context, job Job) {
	if job.Mode == ModeBatch {
		// The checkpoints in the target database are the evidence, and the worker
		// re-reads them when it runs the job again, so resuming is always safe.
		if job.Attempt < job.MaxAttempts {
			w.retry(sctx, logCtx, job, errors.New("previous attempt ended without a known outcome"))
			return
		}
		w.stopBatch(sctx, logCtx, job, errors.New("attempts exhausted after an unknown outcome"))
		return
	}

	applied := false
	if job.Version.Valid {
		ok, err := w.exec.WasApplied(sctx, job, job.Version.Int64)
		if err != nil {
			// Cannot tell. Stay 'uncertain' (lock held); the reaper tries again.
			slog.WarnContext(logCtx, "job.uncertain_unresolved", "job_id", job.ID, "error", err)
			return
		}
		applied = ok
	}
	if applied {
		if err := w.store.MarkSuccess(sctx, job.ID); err == nil {
			slog.InfoContext(logCtx, "job.finished", "job_id", job.ID, "status", StatusSuccess, "recovered", true)
			w.updateWorkflowStatus(logCtx, job, "success", "Execution completed after uncertain commit was verified")
		}
		return
	}
	// Not applied (or never reached execution): safe to try again.
	if job.Attempt < job.MaxAttempts {
		w.retry(sctx, logCtx, job, errors.New("previous attempt did not commit"))
		return
	}
	w.fail(sctx, job, errors.New("attempts exhausted after uncertain commit"))
}

func (w *Worker) backoff(attempt int) time.Duration {
	d := w.cfg.BaseBackoff << min(max(attempt-1, 0), 6)
	return d + time.Duration(rand.Int64N(int64(d)/5+1))
}

func (w *Worker) heartbeat(ctx context.Context, jobID string, cancelJob context.CancelFunc) {
	t := time.NewTicker(w.cfg.HeartbeatInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			owned, err := w.store.Heartbeat(ctx, jobID, w.cfg.ID)
			if err != nil {
				slog.WarnContext(ctx, "job.heartbeat_failed", "job_id", jobID, "error", err)
				continue
			}
			if !owned {
				slog.ErrorContext(ctx, "job.lost", "job_id", jobID, "worker_id", w.cfg.ID)
				cancelJob()
				return
			}
		}
	}
}

// reaperLoop recovers jobs whose worker died: stale 'running' jobs become
// 'uncertain', then every uncertain job is resolved from evidence in the target.
func (w *Worker) reaperLoop(ctx context.Context) {
	t := time.NewTicker(w.cfg.StaleAfter / 3)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			w.reapOnce(ctx)
		}
	}
}

func (w *Worker) reapOnce(ctx context.Context) {
	n, err := w.store.ReapStale(ctx, w.cfg.StaleAfter)
	if err != nil {
		slog.WarnContext(ctx, "worker.reap_failed", "error", err)
		return
	}
	if n > 0 {
		slog.WarnContext(ctx, "worker.reaped", "worker_id", w.cfg.ID, "jobs", n)
	}
	jobs, err := w.store.ListUncertain(ctx)
	if err != nil {
		slog.WarnContext(ctx, "worker.reap_failed", "error", err)
		return
	}
	for _, job := range jobs {
		sctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
		w.resolveUncertain(sctx, ctx, job)
		cancel()
	}
}
