package dbmanager

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/lib/pq"
)

/* migration shi */
func (m *Manager) EnsureMigrationTable(ctx context.Context) error {
	_, err := m.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS db_platform_migrations (
			id BIGSERIAL PRIMARY KEY,
			version BIGINT NOT NULL,
			sql_text TEXT NOT NULL,
			executed_sql TEXT NOT NULL DEFAULT '',
			sql_down TEXT NOT NULL DEFAULT '',
			is_irreversible BOOLEAN NOT NULL DEFAULT FALSE,
			irreversible_reason TEXT,
			action TEXT NOT NULL DEFAULT 'up' CHECK (action IN ('up', 'rollback', 'forward')),
			target_execution_id BIGINT REFERENCES db_platform_migrations(id),
			migration_id BIGINT,
			job_id TEXT,
			request_id TEXT,
			revision INT NOT NULL DEFAULT 1,
			executed_by TEXT NOT NULL,
			status TEXT NOT NULL CHECK (status IN ('success', 'failed')),
			error_message TEXT,
			executed_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return fmt.Errorf("ensure migration table: %w", err)
	}
	for _, stmt := range []string{
		`ALTER TABLE db_platform_migrations ADD COLUMN IF NOT EXISTS sql_down TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE db_platform_migrations ADD COLUMN IF NOT EXISTS executed_sql TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE db_platform_migrations ADD COLUMN IF NOT EXISTS is_irreversible BOOLEAN NOT NULL DEFAULT TRUE`,
		`ALTER TABLE db_platform_migrations ADD COLUMN IF NOT EXISTS irreversible_reason TEXT`,
		`ALTER TABLE db_platform_migrations ADD COLUMN IF NOT EXISTS action TEXT NOT NULL DEFAULT 'up'`,
		`ALTER TABLE db_platform_migrations ADD COLUMN IF NOT EXISTS target_execution_id BIGINT`,
		`ALTER TABLE db_platform_migrations ADD COLUMN IF NOT EXISTS migration_id BIGINT`,
		`ALTER TABLE db_platform_migrations ADD COLUMN IF NOT EXISTS job_id TEXT`,
		`ALTER TABLE db_platform_migrations ADD COLUMN IF NOT EXISTS request_id TEXT`,
		`ALTER TABLE db_platform_migrations ADD COLUMN IF NOT EXISTS revision INT NOT NULL DEFAULT 1`,
	} {
		if _, err := m.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("upgrade migration definitions: %w", err)
		}
	}
	if _, err := m.db.ExecContext(ctx, `UPDATE db_platform_migrations SET executed_sql = sql_text WHERE executed_sql = ''`); err != nil {
		return fmt.Errorf("backfill executed migration SQL: %w", err)
	}
	if _, err := m.db.ExecContext(ctx, `
		UPDATE db_platform_migrations
		SET irreversible_reason = 'Legacy migration predates stored SQL down definitions.'
		WHERE is_irreversible = TRUE AND (irreversible_reason IS NULL OR irreversible_reason = '')`); err != nil {
		return fmt.Errorf("backfill legacy irreversible migration reasons: %w", err)
	}
	return nil
}

func (m *Manager) ListMigrations(ctx context.Context) ([]Migration, error) {
	if err := m.EnsureMigrationTable(ctx); err != nil {
		return nil, err
	}
	rows, err := m.db.QueryContext(ctx, `
		SELECT id, version, migration_id, job_id, request_id, revision, sql_text, executed_sql, sql_down, is_irreversible, irreversible_reason, executed_by, action, target_execution_id, status, error_message, executed_at
		FROM db_platform_migrations
		ORDER BY version, id`)
	if err != nil && isUndefinedTableError(err) {
		return []Migration{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("list migrations: %w", err)
	}
	defer rows.Close()

	var migrations []Migration
	for rows.Next() {
		var migration Migration
		if err := rows.Scan(
			&migration.ExecutionID, &migration.Version, &migration.MigrationID, &migration.JobID, &migration.RequestID, &migration.Revision,
			&migration.SQL, &migration.ExecutedSQL, &migration.SQLDown, &migration.Irreversible, &migration.IrreversibleReason, &migration.ExecutedBy, &migration.Action, &migration.TargetExecutionID,
			&migration.Status, &migration.Error, &migration.ExecutedAt,
		); err != nil {
			return nil, fmt.Errorf("scan migration: %w", err)
		}
		migrations = append(migrations, migration)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate migrations: %w", err)
	}
	return migrations, nil
}

func (m *Manager) GetMigration(ctx context.Context, version int64) (Migration, error) {
	if err := m.EnsureMigrationTable(ctx); err != nil {
		return Migration{}, err
	}
	var migration Migration
	err := m.db.QueryRowContext(ctx, `
		SELECT id, version, migration_id, job_id, request_id, revision, sql_text, executed_sql, sql_down, is_irreversible, irreversible_reason, executed_by, action, target_execution_id, status, error_message, executed_at
		FROM db_platform_migrations WHERE version = $1
		ORDER BY id DESC LIMIT 1`, version).Scan(
		&migration.ExecutionID, &migration.Version, &migration.MigrationID, &migration.JobID, &migration.RequestID, &migration.Revision,
		&migration.SQL, &migration.ExecutedSQL, &migration.SQLDown, &migration.Irreversible, &migration.IrreversibleReason, &migration.ExecutedBy, &migration.Action, &migration.TargetExecutionID,
		&migration.Status, &migration.Error, &migration.ExecutedAt,
	)
	if isUndefinedTableError(err) {
		return Migration{}, sql.ErrNoRows
	}
	if err != nil {
		return Migration{}, fmt.Errorf("get migration %d: %w", version, err)
	}
	return migration, nil
}

func (m *Manager) GetMigrationExecution(ctx context.Context, executionID int64) (Migration, error) {
	if err := m.EnsureMigrationTable(ctx); err != nil {
		return Migration{}, err
	}
	var migration Migration
	err := m.db.QueryRowContext(ctx, `
		SELECT id, version, migration_id, job_id, request_id, revision, sql_text, executed_sql, sql_down, is_irreversible, irreversible_reason,
		       executed_by, action, target_execution_id, status, error_message, executed_at
		FROM db_platform_migrations WHERE id=$1`, executionID).Scan(
		&migration.ExecutionID, &migration.Version, &migration.MigrationID, &migration.JobID, &migration.RequestID, &migration.Revision,
		&migration.SQL, &migration.ExecutedSQL, &migration.SQLDown,
		&migration.Irreversible, &migration.IrreversibleReason, &migration.ExecutedBy, &migration.Action,
		&migration.TargetExecutionID, &migration.Status, &migration.Error, &migration.ExecutedAt)
	if isUndefinedTableError(err) {
		return Migration{}, sql.ErrNoRows
	}
	if err != nil {
		return Migration{}, fmt.Errorf("get migration execution %d: %w", executionID, err)
	}
	return migration, nil
}

func (m *Manager) NextMigrationVersion(ctx context.Context) (int64, error) {
	var version int64
	err := m.db.QueryRowContext(ctx, `
		SELECT COALESCE(MAX(version), 0) + 1
		FROM db_platform_migrations
		WHERE status = 'success'`).Scan(&version)
	if err != nil && isUndefinedTableError(err) {
		return 1, nil
	}
	if err != nil {
		return 0, fmt.Errorf("read next migration version: %w", err)
	}
	return version, nil
}

func isUndefinedTableError(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "42P01"
}

func (m *Manager) ExecuteMigration(ctx context.Context, version int64, migrationSQL, executedBy string) error {
	return m.ExecuteMigrationDefinition(ctx, version, migrationSQL, "", true, "Legacy migration call has no SQL down definition.", executedBy)
}

func (m *Manager) ExecuteMigrationDefinition(ctx context.Context, version int64, sqlUp, sqlDown string, irreversible bool, irreversibleReason, executedBy string) error {
	return m.ExecuteMigrationDefinitionWithMetadata(ctx, version, sqlUp, sqlDown, irreversible, irreversibleReason, executedBy, MigrationMetadata{})
}

func (m *Manager) ExecuteMigrationDefinitionWithMetadata(ctx context.Context, version int64, sqlUp, sqlDown string, irreversible bool, irreversibleReason, executedBy string, metadata MigrationMetadata) error {
	if version < 1 {
		return logMigrationValidation(ctx, "definition", "up", version, 0, metadata, errors.New("migration version must be positive"))
	}
	if strings.TrimSpace(sqlUp) == "" {
		return logMigrationValidation(ctx, "definition", "up", version, 0, metadata, errors.New("migration SQL is required"))
	}
	if irreversible {
		if strings.TrimSpace(irreversibleReason) == "" {
			return logMigrationValidation(ctx, "definition", "up", version, 0, metadata, errors.New("irreversible migration reason is required"))
		}
		sqlDown = ""
	} else if strings.TrimSpace(sqlDown) == "" {
		return logMigrationValidation(ctx, "definition", "up", version, 0, metadata, errors.New("SQL down is required unless the migration is marked irreversible"))
	} else {
		irreversibleReason = ""
	}
	if strings.TrimSpace(executedBy) == "" {
		return logMigrationValidation(ctx, "definition", "up", version, 0, metadata, errors.New("migration executor is required"))
	}
	logMigrationValidation(ctx, "definition", "up", version, 0, metadata, nil, "sql_bytes", len(sqlUp), "irreversible", irreversible)
	if err := m.EnsureMigrationTable(ctx); err != nil {
		return err
	}

	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtext('db_platform_migrations'))`); err != nil {
		return fmt.Errorf("lock migration history: %w", err)
	}

	var latest int64
	if err := tx.QueryRowContext(ctx, `
		SELECT COALESCE(MAX(version), 0)
		FROM db_platform_migrations
		WHERE status = 'success'`).Scan(&latest); err != nil {
		return fmt.Errorf("read latest migration version: %w", err)
	}
	if version != latest+1 {
		err := fmt.Errorf("migration version must be %d, got %d", latest+1, version)
		return logMigrationValidation(ctx, "version", "up", version, 0, metadata, err, "expected_version", latest+1)
	}
	var priorUp, priorDown string
	var priorIrreversible bool
	var priorReason sql.NullString
	err = tx.QueryRowContext(ctx, `
		SELECT sql_text, sql_down, is_irreversible, irreversible_reason
		FROM db_platform_migrations WHERE version = $1 AND status = 'success' ORDER BY id LIMIT 1`, version,
	).Scan(&priorUp, &priorDown, &priorIrreversible, &priorReason)
	if err == nil {
		if priorUp != sqlUp || priorDown != sqlDown || priorIrreversible != irreversible || priorReason.String != irreversibleReason {
			err := fmt.Errorf("migration version %d already has an immutable definition; create a new migration", version)
			return logMigrationValidation(ctx, "immutable_definition", "up", version, 0, metadata, err)
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("check existing migration definition: %w", err)
	}
	logMigrationValidation(ctx, "version_and_definition", "up", version, 0, metadata, nil)

	if _, err := tx.ExecContext(ctx, sqlUp); err != nil {
		slog.ErrorContext(ctx, "migration.sql_execution.failed", "migration_version", version,
			"job_id", nullStringText(metadata.JobID), "sql_bytes", len(sqlUp), "error", err)
		_ = tx.Rollback()
		recordErr := m.recordMigrationFailure(ctx, version, sqlUp, sqlDown, irreversible, irreversibleReason, executedBy, err.Error(), metadata)
		if recordErr != nil {
			return fmt.Errorf("execute migration %d: %w; record failure: %v", version, err, recordErr)
		}
		return fmt.Errorf("execute migration %d: %w", version, err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO db_platform_migrations
			(version, sql_text, executed_sql, sql_down, is_irreversible, irreversible_reason, migration_id, job_id, request_id, revision, executed_by, status)
		VALUES ($1, $2, $2, $3, $4, NULLIF($5, ''), $6, NULLIF($7,''), NULLIF($8,''), $9, $10, 'success')`,
		version, sqlUp, sqlDown, irreversible, irreversibleReason, metadata.MigrationID, metadata.JobID.String,
		metadata.RequestID, migrationRevision(metadata), executedBy,
	); err != nil {
		return fmt.Errorf("record migration %d: %w", version, err)
	}
	if err := tx.Commit(); err != nil {
		return &CommitError{Version: version, Err: err}
	}
	return nil
}

func (m *Manager) RecordMigrationFailureDefinition(ctx context.Context, version int64, sqlUp, sqlDown string, irreversible bool, irreversibleReason, executedBy, message string, metadata MigrationMetadata) error {
	if version < 1 {
		return errors.New("migration version must be positive")
	}
	if strings.TrimSpace(sqlUp) == "" {
		return errors.New("migration SQL is required")
	}
	if strings.TrimSpace(executedBy) == "" {
		return errors.New("migration executor is required")
	}
	if irreversible {
		sqlDown = ""
	} else if strings.TrimSpace(sqlDown) == "" {
		return errors.New("SQL down is required unless the migration is marked irreversible")
	}
	if err := m.EnsureMigrationTable(ctx); err != nil {
		return err
	}
	return m.recordMigrationFailure(ctx, version, sqlUp, sqlDown, irreversible, irreversibleReason, executedBy, message, metadata)
}

// RollbackMigration runs SQL down for the latest active migration and records a new execution.
func (m *Manager) RollbackMigration(ctx context.Context, executionID int64, executedBy string) error {
	return m.RollbackMigrationWithMetadata(ctx, executionID, executedBy, MigrationMetadata{})
}

// ForwardMigration re-runs SQL up for a migration whose latest successful action is rollback.
func (m *Manager) ForwardMigration(ctx context.Context, rollbackExecutionID int64, executedBy string) error {
	return m.ForwardMigrationWithMetadata(ctx, rollbackExecutionID, executedBy, MigrationMetadata{})
}

func (m *Manager) RollbackMigrationWithMetadata(ctx context.Context, executionID int64, executedBy string, metadata MigrationMetadata) error {
	return m.executeMigrationAction(ctx, executionID, "rollback", executedBy, metadata)
}

func (m *Manager) ForwardMigrationWithMetadata(ctx context.Context, rollbackExecutionID int64, executedBy string, metadata MigrationMetadata) error {
	return m.executeMigrationAction(ctx, rollbackExecutionID, "forward", executedBy, metadata)
}

func (m *Manager) ValidateMigrationAction(ctx context.Context, targetID int64, action string) (Migration, error) {
	if targetID < 1 {
		err := errors.New("target execution ID must be positive")
		return Migration{}, logMigrationValidation(ctx, "action_preflight", action, 0, targetID, MigrationMetadata{}, err)
	}
	if action != "rollback" && action != "forward" {
		err := errors.New("unsupported migration action")
		return Migration{}, logMigrationValidation(ctx, "action_preflight", action, 0, targetID, MigrationMetadata{}, err)
	}
	if err := m.EnsureMigrationTable(ctx); err != nil {
		return Migration{}, logMigrationValidation(ctx, "action_preflight_schema", action, 0, targetID, MigrationMetadata{}, err)
	}
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		err = fmt.Errorf("begin %s validation: %w", action, err)
		return Migration{}, logMigrationValidation(ctx, "action_preflight_transaction", action, 0, targetID, MigrationMetadata{}, err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtext('db_platform_migrations'))`); err != nil {
		return Migration{}, fmt.Errorf("lock migration history: %w", err)
	}
	target, _, err := validateMigrationAction(ctx, tx, targetID, action)
	logMigrationValidation(ctx, "action_preflight", action, target.Version, targetID, MigrationMetadata{}, err)
	return target, err
}

func validateMigrationAction(ctx context.Context, tx *sql.Tx, targetID int64, action string) (Migration, string, error) {
	var target Migration
	err := tx.QueryRowContext(ctx, `
		SELECT id, version, sql_text, sql_down, is_irreversible, irreversible_reason, action, status
		FROM db_platform_migrations WHERE id = $1`, targetID,
	).Scan(&target.ExecutionID, &target.Version, &target.SQL, &target.SQLDown, &target.Irreversible, &target.IrreversibleReason, &target.Action, &target.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return Migration{}, "", fmt.Errorf("migration execution %d was not found", targetID)
	}
	if err != nil {
		return Migration{}, "", fmt.Errorf("load migration execution %d: %w", targetID, err)
	}
	if target.Status != "success" {
		return target, "", errors.New("only successful migration executions can be rolled back or forwarded")
	}

	if action == "rollback" {
		if target.Action != "up" && target.Action != "forward" {
			return target, "", errors.New("rollback target must be an active up or forward execution")
		}
		if target.Irreversible || strings.TrimSpace(target.SQLDown) == "" {
			return target, "", errors.New("migration is irreversible and cannot be rolled back")
		}
		var latestID int64
		var latestAction string
		if err := tx.QueryRowContext(ctx, `SELECT id, action FROM db_platform_migrations WHERE version=$1 AND status='success' ORDER BY id DESC LIMIT 1`, target.Version).Scan(&latestID, &latestAction); err != nil {
			return target, "", fmt.Errorf("read latest action for migration %d: %w", target.Version, err)
		}
		if latestID != targetID || (latestAction != "up" && latestAction != "forward") {
			return target, "", errors.New("only the latest active execution of a migration can be rolled back")
		}
		var higherActive bool
		if err := tx.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM db_platform_migrations current
				WHERE current.version > $1 AND current.status='success'
				  AND current.action IN ('up', 'forward')
				  AND NOT EXISTS (SELECT 1 FROM db_platform_migrations later WHERE later.version=current.version AND later.status='success' AND later.id > current.id)
			)`, target.Version).Scan(&higherActive); err != nil {
			return target, "", fmt.Errorf("check migration order: %w", err)
		}
		if higherActive {
			return target, "", errors.New("roll back later active migrations first")
		}
		return target, target.SQLDown, nil
	}

	if target.Action != "rollback" {
		return target, "", errors.New("forward target must be a successful rollback execution")
	}
	var latestID int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM db_platform_migrations WHERE version=$1 AND status='success' ORDER BY id DESC LIMIT 1`, target.Version).Scan(&latestID); err != nil {
		return target, "", fmt.Errorf("read latest action for migration %d: %w", target.Version, err)
	}
	if latestID != targetID {
		return target, "", errors.New("only the latest rollback of a migration can be forwarded")
	}
	var higherActive bool
	if err := tx.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM db_platform_migrations current
			WHERE current.version > $1 AND current.status='success'
			  AND current.action IN ('up', 'forward')
			  AND NOT EXISTS (SELECT 1 FROM db_platform_migrations later WHERE later.version=current.version AND later.status='success' AND later.id > current.id)
		)`, target.Version).Scan(&higherActive); err != nil {
		return target, "", fmt.Errorf("check migration order: %w", err)
	}
	if higherActive {
		return target, "", errors.New("forward later migrations in order after this migration")
	}
	var lowerInactive bool
	if err := tx.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM db_platform_migrations current
			WHERE current.version < $1 AND current.status='success'
			  AND current.action='rollback'
			  AND NOT EXISTS (SELECT 1 FROM db_platform_migrations later WHERE later.version=current.version AND later.status='success' AND later.id > current.id)
		)`, target.Version).Scan(&lowerInactive); err != nil {
		return target, "", fmt.Errorf("check forward order: %w", err)
	}
	if lowerInactive {
		return target, "", errors.New("forward earlier rolled-back migrations first")
	}
	return target, target.SQL, nil
}

func (m *Manager) RecordMigrationActionFailure(ctx context.Context, targetID int64, action, executedBy, message string, metadata MigrationMetadata) error {
	if err := m.EnsureMigrationTable(ctx); err != nil {
		return err
	}
	if action != "rollback" && action != "forward" {
		return errors.New("unsupported migration action")
	}
	target, err := m.GetMigrationExecution(ctx, targetID)
	if err != nil {
		return err
	}
	return m.recordActionAttempt(ctx, target, targetID, action, executedBy, "failed", message, metadata)
}

func (m *Manager) executeMigrationAction(ctx context.Context, targetID int64, action, executedBy string, metadata MigrationMetadata) error {
	if targetID < 1 {
		return errors.New("target execution ID must be positive")
	}
	if strings.TrimSpace(executedBy) == "" {
		return errors.New("migration executor is required")
	}
	if action != "rollback" && action != "forward" {
		return errors.New("unsupported migration action")
	}
	if err := m.EnsureMigrationTable(ctx); err != nil {
		return err
	}
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin %s transaction: %w", action, err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtext('db_platform_migrations'))`); err != nil {
		return fmt.Errorf("lock migration history: %w", err)
	}

	target, sqlToRun, err := validateMigrationAction(ctx, tx, targetID, action)
	if err != nil {
		logMigrationValidation(ctx, "action_execution", action, target.Version, targetID, metadata, err)
		if target.ExecutionID == 0 {
			return err
		}
		if recordErr := insertActionAttempt(ctx, tx, target, targetID, action, executedBy, "failed", err.Error(), metadata); recordErr != nil {
			return fmt.Errorf("%w; record rejected %s attempt: %v", err, action, recordErr)
		}
		if commitErr := tx.Commit(); commitErr != nil {
			return fmt.Errorf("%w; commit rejected %s attempt: %v", err, action, commitErr)
		}
		return err
	}
	logMigrationValidation(ctx, "action_execution", action, target.Version, targetID, metadata, nil, "sql_bytes", len(sqlToRun))
	if strings.TrimSpace(sqlToRun) == "" {
		err := errors.New("migration action SQL is empty")
		logMigrationValidation(ctx, "action_sql", action, target.Version, targetID, metadata, err)
		if recordErr := insertActionAttempt(ctx, tx, target, targetID, action, executedBy, "failed", err.Error(), metadata); recordErr != nil {
			return fmt.Errorf("%w; record rejected %s attempt: %v", err, action, recordErr)
		}
		if commitErr := tx.Commit(); commitErr != nil {
			return fmt.Errorf("%w; commit rejected %s attempt: %v", err, action, commitErr)
		}
		return err
	}

	if _, err := tx.ExecContext(ctx, sqlToRun); err != nil {
		slog.ErrorContext(ctx, "migration.sql_execution.failed", "migration_version", target.Version,
			"action", action, "target_execution_id", targetID, "job_id", nullStringText(metadata.JobID),
			"sql_bytes", len(sqlToRun), "error", err)
		_ = tx.Rollback()
		recordErr := m.recordActionAttempt(ctx, target, targetID, action, executedBy, "failed", err.Error(), metadata)
		if recordErr != nil {
			return fmt.Errorf("execute %s for migration %d: %w; record failure: %v", action, target.Version, err, recordErr)
		}
		return fmt.Errorf("execute %s for migration %d: %w", action, target.Version, err)
	}
	if err := insertActionAttempt(ctx, tx, target, targetID, action, executedBy, "success", "", metadata); err != nil {
		return fmt.Errorf("record %s for migration %d: %w", action, target.Version, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit %s for migration %d: %w", action, target.Version, err)
	}
	return nil
}

func (m *Manager) recordActionAttempt(ctx context.Context, target Migration, targetExecutionID int64, action, executedBy, status, message string, metadata MigrationMetadata) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin action history transaction: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtext('db_platform_migrations'))`); err != nil {
		return fmt.Errorf("lock migration history: %w", err)
	}
	if err := insertActionAttempt(ctx, tx, target, targetExecutionID, action, executedBy, status, message, metadata); err != nil {
		return err
	}
	return tx.Commit()
}

func insertActionAttempt(ctx context.Context, tx *sql.Tx, target Migration, targetExecutionID int64, action, executedBy, status, message string, metadata MigrationMetadata) error {
	var sqlToRun string
	if action == "rollback" {
		sqlToRun = target.SQLDown
	} else {
		sqlToRun = target.SQL
	}
	_, err := tx.ExecContext(ctx, `
		INSERT INTO db_platform_migrations
			(version, sql_text, executed_sql, sql_down, is_irreversible, irreversible_reason, migration_id, job_id, request_id, revision, executed_by, action, target_execution_id, status, error_message)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), $7, NULLIF($8,''), NULLIF($9,''), $10, $11, $12, $13, $14, NULLIF($15, ''))`,
		target.Version, target.SQL, sqlToRun, target.SQLDown, target.Irreversible, nullStringText(target.IrreversibleReason),
		metadata.MigrationID, metadata.JobID.String, metadata.RequestID, migrationRevision(metadata), executedBy,
		action, targetExecutionID, status, message)
	if err != nil {
		return fmt.Errorf("insert migration action attempt: %w", err)
	}
	return nil
}

func nullStringText(value sql.NullString) string {
	if value.Valid {
		return value.String
	}
	return ""
}

func HiddenBySuccessfulRollback(migrations []Migration) []int64 {
	hidden := make(map[int64]struct{})
	for _, migration := range migrations {
		if migration.Status != "success" || migration.Action != "rollback" {
			continue
		}
		hidden[migration.ExecutionID] = struct{}{}
		if migration.TargetExecutionID.Valid {
			hidden[migration.TargetExecutionID.Int64] = struct{}{}
		}
	}
	ids := make([]int64, 0, len(hidden))
	for id := range hidden {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

// CommitError is returned when COMMIT of a migration (or of one batch, when
// Batch > 0) fails. If Err is not a PostgreSQL error (connection lost, timeout),
// the transaction may or may not have been committed.
type CommitError struct {
	Version int64
	Batch   int
	Err     error
}

func (e *CommitError) Error() string {
	if e.Batch > 0 {
		return fmt.Sprintf("commit batch %d: %v", e.Batch, e.Err)
	}
	return fmt.Sprintf("commit migration %d: %v", e.Version, e.Err)
}

func (e *CommitError) Unwrap() error { return e.Err }

func (m *Manager) recordMigrationFailure(ctx context.Context, version int64, sqlUp, sqlDown string, irreversible bool, reason, executedBy, message string, metadata MigrationMetadata) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin failed migration record: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtext('db_platform_migrations'))`); err != nil {
		return fmt.Errorf("lock failed migration record: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO db_platform_migrations
		(version, sql_text, executed_sql, sql_down, is_irreversible, irreversible_reason, migration_id, job_id, request_id, revision, executed_by, status, error_message)
		VALUES ($1, $2, $2, $3, $4, NULLIF($5, ''), $6, NULLIF($7,''), NULLIF($8,''), $9, $10, 'failed', $11)`,
		version, sqlUp, sqlDown, irreversible, reason, metadata.MigrationID, metadata.JobID.String, metadata.RequestID,
		migrationRevision(metadata), executedBy, message); err != nil {
		return fmt.Errorf("record failed migration: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit failed migration record: %w", err)
	}
	return nil
}

func migrationRevision(metadata MigrationMetadata) int {
	if metadata.Revision > 0 {
		return metadata.Revision
	}
	return 1
}

func logMigrationValidation(ctx context.Context, stage, action string, version, targetID int64, metadata MigrationMetadata, err error, extra ...any) error {
	args := []any{
		"validation_stage", stage,
		"action", action,
		"migration_version", version,
		"target_execution_id", targetID,
	}
	if metadata.MigrationID.Valid {
		args = append(args, "migration_id", metadata.MigrationID.Int64)
	}
	if metadata.JobID.Valid {
		args = append(args, "job_id", metadata.JobID.String)
	}
	if metadata.RequestID != "" {
		args = append(args, "request_id", metadata.RequestID)
	}
	if metadata.Revision > 0 {
		args = append(args, "revision", metadata.Revision)
	}
	args = append(args, extra...)
	if err != nil {
		args = append(args, "ok", false, "error", err)
		slog.WarnContext(ctx, "migration.validation.failed", args...)
		return err
	}
	args = append(args, "ok", true)
	slog.InfoContext(ctx, "migration.validation.passed", args...)
	return nil
}
