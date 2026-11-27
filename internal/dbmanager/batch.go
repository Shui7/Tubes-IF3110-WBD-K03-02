package dbmanager

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/lib/pq"
)

var (
	// A batch SQL did not return what the platform expects. A permanent error.
	ErrInvalidBatchResult = errors.New("invalid batch result")
	// A batch changed rows but returned the same cursor as before
	ErrCursorNotAdvancing = errors.New("batch cursor did not advance")
	// A batch table, column or SQL template is not usable. A permanent error.
	ErrInvalidBatchConfig = errors.New("invalid batch configuration")
)

// BatchFilterPlaceholder is the marker a user puts in the SQL of a "batch by
// attribute" migration. The platform replaces it with the predicate that selects
// the rows of the current batch. Example:
//
//	UPDATE big_items SET price = price * 1.1 WHERE {{batch_filter}}
const BatchFilterPlaceholder = "{{batch_filter}}"

const checkpointTableSQL = `
	CREATE TABLE IF NOT EXISTS db_platform_checkpoints (
		execution_id  TEXT   NOT NULL,
		batch_no      INT    NOT NULL,
		cursor_from   TEXT,
		cursor_to     TEXT,
		rows_affected BIGINT NOT NULL,
		state         TEXT   NOT NULL DEFAULT 'committed' CHECK (state IN ('committed', 'reverted')),
		committed_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (execution_id, batch_no)
	)`

// BatchProgress is what is already committed for one execution.
type BatchProgress struct {
	Batches int            // number of committed batches
	Rows    int64          // total rows changed by committed batches
	Cursor  sql.NullString // cursor after the last committed batch (NULL = not started)
}

// BatchStep is the result of one batch.
type BatchStep struct {
	Cursor sql.NullString // cursor after this batch
	Rows   int64          // rows changed by this batch
	Done   bool           // no more rows to process (nothing was committed)
}

func (m *Manager) EnsureCheckpointTable(ctx context.Context) error {
	if _, err := m.db.ExecContext(ctx, checkpointTableSQL); err != nil {
		return fmt.Errorf("ensure checkpoint table: %w", err)
	}
	return nil
}

// BatchProgress reads the committed checkpoints of an execution.
func (m *Manager) BatchProgress(ctx context.Context, executionID string) (BatchProgress, error) {
	var p BatchProgress
	if strings.TrimSpace(executionID) == "" {
		return p, errors.New("execution id is required")
	}
	if err := m.EnsureCheckpointTable(ctx); err != nil {
		return p, err
	}
	if err := m.db.QueryRowContext(ctx, `
		SELECT COUNT(*), COALESCE(SUM(rows_affected), 0)::bigint
		FROM db_platform_checkpoints
		WHERE execution_id = $1 AND state = 'committed'`, executionID).Scan(&p.Batches, &p.Rows); err != nil {
		return p, fmt.Errorf("read batch progress: %w", err)
	}
	if p.Batches == 0 {
		return p, nil
	}
	if err := m.db.QueryRowContext(ctx, `
		SELECT cursor_to FROM db_platform_checkpoints
		WHERE execution_id = $1 AND state = 'committed'
		ORDER BY batch_no DESC LIMIT 1`, executionID).Scan(&p.Cursor); err != nil {
		return p, fmt.Errorf("read last checkpoint: %w", err)
	}
	return p, nil
}

// ExecuteBatch runs one batch of batchSQL in its own transaction.
//
// Contract for batchSQL: a single statement that
//   - takes $1 = cursor after the previous batch (NULL for the first batch) and
//     $2 = batch size,
//   - processes only the rows after $1, at most $2 of them, in a stable order
//     (for example WHERE id > COALESCE($1::bigint, 0) ORDER BY id LIMIT $2::int),
//   - returns exactly one row with two columns: the new cursor (text) and the
//     number of rows it changed. Zero rows changed means there is nothing left.
//
// When rows were changed, the checkpoint is written in the same transaction
// before COMMIT. If COMMIT fails without a server verdict the error wraps
// *CommitError; callers must then re-read BatchProgress to learn the outcome.
func (m *Manager) ExecuteBatch(ctx context.Context, executionID string, batchNo int,
	batchSQL string, cursor sql.NullString, size int) (BatchStep, error) {

	if strings.TrimSpace(executionID) == "" {
		return BatchStep{}, errors.New("execution id is required")
	}
	if batchNo < 1 {
		return BatchStep{}, errors.New("batch number must be positive")
	}
	if strings.TrimSpace(batchSQL) == "" {
		return BatchStep{}, errors.New("batch SQL is required")
	}
	if size < 1 {
		return BatchStep{}, errors.New("batch size must be positive")
	}
	if err := m.EnsureCheckpointTable(ctx); err != nil {
		return BatchStep{}, err
	}

	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return BatchStep{}, fmt.Errorf("begin batch %d: %w", batchNo, err)
	}
	defer tx.Rollback()

	var next sql.NullString
	var rows int64
	if err := tx.QueryRowContext(ctx, batchSQL, cursor, size).Scan(&next, &rows); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return BatchStep{}, fmt.Errorf("batch %d: %w: the SQL returned no row (it must return next cursor and row count)", batchNo, ErrInvalidBatchResult)
		}
		return BatchStep{}, fmt.Errorf("execute batch %d: %w", batchNo, err)
	}

	if rows < 0 {
		return BatchStep{}, fmt.Errorf("batch %d: %w: negative row count", batchNo, ErrInvalidBatchResult)
	}
	if rows == 0 {
		return BatchStep{Cursor: cursor, Done: true}, nil // nothing changed, nothing to commit
	}
	if !next.Valid {
		return BatchStep{}, fmt.Errorf("batch %d: %w: rows were changed but the next cursor is NULL", batchNo, ErrInvalidBatchResult)
	}
	if cursor.Valid && next.String == cursor.String {
		return BatchStep{}, fmt.Errorf("batch %d: %w (cursor stayed at %q)", batchNo, ErrCursorNotAdvancing, cursor.String)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO db_platform_checkpoints (execution_id, batch_no, cursor_from, cursor_to, rows_affected)
		VALUES ($1, $2, $3, $4, $5)`, executionID, batchNo, cursor, next, rows); err != nil {
		return BatchStep{}, fmt.Errorf("write checkpoint for batch %d: %w", batchNo, err)
	}
	if err := tx.Commit(); err != nil {
		return BatchStep{}, &CommitError{Batch: batchNo, Err: err}
	}
	return BatchStep{Cursor: next, Rows: rows}, nil
}

// RecordMigrationSuccess preserves the legacy API for callers which do not
// provide a reversible definition. New migration jobs should call
// RecordMigrationSuccessDefinition with the approved SQL up/down pair.
func (m *Manager) RecordMigrationSuccess(ctx context.Context, version int64, migrationSQL, executedBy string) error {
	return m.RecordMigrationSuccessDefinition(ctx, version, migrationSQL, "", true,
		"Legacy batch migration call has no SQL down definition.", executedBy)
}

// RecordMigrationSuccessDefinition writes the history row for a migration whose
// data was changed by batches. It is idempotent for the same immutable definition.
func (m *Manager) RecordMigrationSuccessDefinition(ctx context.Context, version int64, migrationSQL, sqlDown string, irreversible bool, irreversibleReason, executedBy string) error {
	return m.RecordMigrationSuccessDefinitionWithMetadata(ctx, version, migrationSQL, sqlDown, irreversible, irreversibleReason, executedBy, MigrationMetadata{})
}

func (m *Manager) RecordMigrationSuccessDefinitionWithMetadata(ctx context.Context, version int64, migrationSQL, sqlDown string, irreversible bool, irreversibleReason, executedBy string, metadata MigrationMetadata) error {
	if version < 1 {
		return errors.New("migration version must be positive")
	}
	if strings.TrimSpace(migrationSQL) == "" {
		return errors.New("migration SQL is required")
	}
	if strings.TrimSpace(executedBy) == "" {
		return errors.New("migration executor is required")
	}
	if irreversible {
		if strings.TrimSpace(irreversibleReason) == "" {
			return errors.New("irreversible migration reason is required")
		}
		sqlDown = ""
	} else if strings.TrimSpace(sqlDown) == "" {
		return errors.New("SQL down is required unless the migration is marked irreversible")
	} else {
		irreversibleReason = ""
	}
	if err := m.EnsureMigrationTable(ctx); err != nil {
		return err
	}

	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration record: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtext('db_platform_migrations'))`); err != nil {
		return fmt.Errorf("lock migration history: %w", err)
	}

	var oldUp, oldDown string
	var oldIrreversible bool
	var oldReason sql.NullString
	err = tx.QueryRowContext(ctx, `SELECT sql_text, sql_down, is_irreversible, irreversible_reason
		FROM db_platform_migrations WHERE version=$1 AND status='success' ORDER BY id LIMIT 1`, version).
		Scan(&oldUp, &oldDown, &oldIrreversible, &oldReason)
	if err == nil {
		if oldUp != migrationSQL || oldDown != sqlDown || oldIrreversible != irreversible || oldReason.String != irreversibleReason {
			return fmt.Errorf("migration version %d already has an immutable definition", version)
		}
		var recorded bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM db_platform_migrations WHERE version=$1 AND status='success')`, version).Scan(&recorded); err != nil {
			return fmt.Errorf("check migration record: %w", err)
		}
		if recorded {
			return nil
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("check migration definition: %w", err)
	}

	var latest int64
	if err := tx.QueryRowContext(ctx, `
		SELECT COALESCE(MAX(version), 0) FROM db_platform_migrations WHERE status = 'success'`).Scan(&latest); err != nil {
		return fmt.Errorf("read latest migration version: %w", err)
	}
	if version != latest+1 {
		return fmt.Errorf("migration version must be %d, got %d", latest+1, version)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO db_platform_migrations
			(version, sql_text, executed_sql, sql_down, is_irreversible, irreversible_reason, migration_id, job_id, request_id, revision, executed_by, status)
		VALUES ($1, $2, $2, $3, $4, NULLIF($5,''), $6, NULLIF($7,''), NULLIF($8,''), $9, $10, 'success')`,
		version, migrationSQL, sqlDown, irreversible, irreversibleReason, metadata.MigrationID, metadata.JobID.String,
		metadata.RequestID, migrationRevision(metadata), executedBy); err != nil {
		return fmt.Errorf("record migration %d: %w", version, err)
	}
	if err := tx.Commit(); err != nil {
		return &CommitError{Version: version, Err: err}
	}
	return nil
}

// quoteTableName quotes "table" or "schema.table" so it is safe inside SQL.
func quoteTableName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("%w: batch table is required", ErrInvalidBatchConfig)
	}
	parts := strings.Split(name, ".")
	if len(parts) > 2 {
		return "", fmt.Errorf("%w: table must be \"table\" or \"schema.table\"", ErrInvalidBatchConfig)
	}
	quoted := make([]string, len(parts))
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return "", fmt.Errorf("%w: table name has an empty part", ErrInvalidBatchConfig)
		}
		quoted[i] = pq.QuoteIdentifier(part)
	}
	return strings.Join(quoted, "."), nil
}

// quoteColumnName quotes a single column name so it is safe inside SQL.
func quoteColumnName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("%w: batch column is required", ErrInvalidBatchConfig)
	}
	if strings.Contains(name, ".") {
		return "", fmt.Errorf("%w: batch column must be a plain column name", ErrInvalidBatchConfig)
	}
	return pq.QuoteIdentifier(name), nil
}

// ExecuteAttributeBatch runs one batch of a "batch by attribute" migration.
func (m *Manager) ExecuteAttributeBatch(ctx context.Context, executionID string, batchNo int,
	batchSQL, table, column string, cursor sql.NullString, size int) (BatchStep, error) {

	if strings.TrimSpace(executionID) == "" {
		return BatchStep{}, errors.New("execution id is required")
	}
	if batchNo < 1 {
		return BatchStep{}, errors.New("batch number must be positive")
	}
	if size < 1 {
		return BatchStep{}, errors.New("batch size must be positive")
	}
	if !strings.Contains(batchSQL, BatchFilterPlaceholder) {
		return BatchStep{}, fmt.Errorf("%w: the SQL must contain %s", ErrInvalidBatchConfig, BatchFilterPlaceholder)
	}
	tbl, err := quoteTableName(table)
	if err != nil {
		return BatchStep{}, err
	}
	col, err := quoteColumnName(column)
	if err != nil {
		return BatchStep{}, err
	}
	if err := m.EnsureCheckpointTable(ctx); err != nil {
		return BatchStep{}, err
	}

	// $1 = cursor (NULL for the first batch), $2 = upper bound / batch size.
	after := "(" + col + " > $1 OR $1 IS NULL)"
	boundSQL := "SELECT b.c::text FROM (SELECT " + col + " AS c FROM " + tbl +
		" WHERE " + after + " ORDER BY " + col + " LIMIT $2::int) b ORDER BY b.c DESC LIMIT 1"
	filter := "(" + after + " AND " + col + " <= $2)"
	statement := strings.ReplaceAll(batchSQL, BatchFilterPlaceholder, filter)

	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return BatchStep{}, fmt.Errorf("begin batch %d: %w", batchNo, err)
	}
	defer tx.Rollback()

	var upper sql.NullString
	err = tx.QueryRowContext(ctx, boundSQL, cursor, size).Scan(&upper)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && !upper.Valid) {
		return BatchStep{Cursor: cursor, Done: true}, nil // no rows after the cursor
	}
	if err != nil {
		return BatchStep{}, fmt.Errorf("find bound of batch %d: %w", batchNo, err)
	}

	result, err := tx.ExecContext(ctx, statement, cursor, upper.String)
	if err != nil {
		return BatchStep{}, fmt.Errorf("execute batch %d: %w", batchNo, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return BatchStep{}, fmt.Errorf("batch %d: read affected rows: %w", batchNo, err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO db_platform_checkpoints (execution_id, batch_no, cursor_from, cursor_to, rows_affected)
		VALUES ($1, $2, $3, $4, $5)`, executionID, batchNo, cursor, upper, rows); err != nil {
		return BatchStep{}, fmt.Errorf("write checkpoint for batch %d: %w", batchNo, err)
	}
	if err := tx.Commit(); err != nil {
		return BatchStep{}, &CommitError{Batch: batchNo, Err: err}
	}
	return BatchStep{Cursor: upper, Rows: rows}, nil
}
