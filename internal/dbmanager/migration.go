package dbmanager

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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
	rows, err := m.db.QueryContext(ctx, `
		SELECT id, version, sql_text, executed_sql, sql_down, is_irreversible, irreversible_reason, executed_by, action, target_execution_id, status, error_message, executed_at
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
			&migration.ExecutionID, &migration.Version, &migration.SQL, &migration.ExecutedSQL, &migration.SQLDown, &migration.Irreversible, &migration.IrreversibleReason, &migration.ExecutedBy, &migration.Action, &migration.TargetExecutionID,
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
	var migration Migration
	err := m.db.QueryRowContext(ctx, `
		SELECT id, version, sql_text, executed_sql, sql_down, is_irreversible, irreversible_reason, executed_by, action, target_execution_id, status, error_message, executed_at
		FROM db_platform_migrations WHERE version = $1
		ORDER BY id DESC LIMIT 1`, version).Scan(
		&migration.ExecutionID, &migration.Version, &migration.SQL, &migration.ExecutedSQL, &migration.SQLDown, &migration.Irreversible, &migration.IrreversibleReason, &migration.ExecutedBy, &migration.Action, &migration.TargetExecutionID,
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
	var migration Migration
	err := m.db.QueryRowContext(ctx, `
		SELECT id, version, sql_text, executed_sql, sql_down, is_irreversible, irreversible_reason,
		       executed_by, action, target_execution_id, status, error_message, executed_at
		FROM db_platform_migrations WHERE id=$1`, executionID).Scan(
		&migration.ExecutionID, &migration.Version, &migration.SQL, &migration.ExecutedSQL, &migration.SQLDown,
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
	if version < 1 {
		return errors.New("migration version must be positive")
	}
	if strings.TrimSpace(sqlUp) == "" {
		return errors.New("migration SQL is required")
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
	if strings.TrimSpace(executedBy) == "" {
		return errors.New("migration executor is required")
	}
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
		return fmt.Errorf("migration version must be %d, got %d", latest+1, version)
	}
	var priorUp, priorDown string
	var priorIrreversible bool
	var priorReason sql.NullString
	err = tx.QueryRowContext(ctx, `
		SELECT sql_text, sql_down, is_irreversible, irreversible_reason
		FROM db_platform_migrations WHERE version = $1 ORDER BY id LIMIT 1`, version,
	).Scan(&priorUp, &priorDown, &priorIrreversible, &priorReason)
	if err == nil {
		if priorUp != sqlUp || priorDown != sqlDown || priorIrreversible != irreversible || priorReason.String != irreversibleReason {
			return fmt.Errorf("migration version %d already has an immutable definition; create a new migration", version)
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("check existing migration definition: %w", err)
	}

	if _, err := tx.ExecContext(ctx, sqlUp); err != nil {
		_ = tx.Rollback()
		recordErr := m.recordMigrationFailure(ctx, version, sqlUp, sqlDown, irreversible, irreversibleReason, executedBy, err.Error())
		if recordErr != nil {
			return fmt.Errorf("execute migration %d: %w; record failure: %v", version, err, recordErr)
		}
		return fmt.Errorf("execute migration %d: %w", version, err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO db_platform_migrations
			(version, sql_text, executed_sql, sql_down, is_irreversible, irreversible_reason, executed_by, status)
		VALUES ($1, $2, $2, $3, $4, NULLIF($5, ''), $6, 'success')`,
		version, sqlUp, sqlDown, irreversible, irreversibleReason, executedBy,
	); err != nil {
		return fmt.Errorf("record migration %d: %w", version, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration %d: %w", version, err)
	}
	return nil
}

// RollbackMigration runs SQL down for the latest active migration and records a new execution.
func (m *Manager) RollbackMigration(ctx context.Context, executionID int64, executedBy string) error {
	return m.executeMigrationAction(ctx, executionID, "rollback", executedBy)
}

// ForwardMigration re-runs SQL up for a migration whose latest successful action is rollback.
func (m *Manager) ForwardMigration(ctx context.Context, rollbackExecutionID int64, executedBy string) error {
	return m.executeMigrationAction(ctx, rollbackExecutionID, "forward", executedBy)
}

func (m *Manager) executeMigrationAction(ctx context.Context, targetID int64, action, executedBy string) error {
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

	var target Migration
	err = tx.QueryRowContext(ctx, `
		SELECT id, version, sql_text, sql_down, is_irreversible, irreversible_reason, action, status
		FROM db_platform_migrations WHERE id = $1`, targetID,
	).Scan(&target.ExecutionID, &target.Version, &target.SQL, &target.SQLDown, &target.Irreversible, &target.IrreversibleReason, &target.Action, &target.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("migration execution %d was not found", targetID)
	}
	if err != nil {
		return fmt.Errorf("load migration execution %d: %w", targetID, err)
	}
	if target.Status != "success" {
		return errors.New("only successful migration executions can be rolled back or forwarded")
	}

	var sqlToRun string
	var targetExecutionID int64
	if action == "rollback" {
		if target.Action != "up" && target.Action != "forward" {
			return errors.New("rollback target must be an active up or forward execution")
		}
		if target.Irreversible || strings.TrimSpace(target.SQLDown) == "" {
			return errors.New("migration is irreversible and cannot be rolled back")
		}
		var latestID int64
		var latestAction string
		if err := tx.QueryRowContext(ctx, `SELECT id, action FROM db_platform_migrations WHERE version=$1 AND status='success' ORDER BY id DESC LIMIT 1`, target.Version).Scan(&latestID, &latestAction); err != nil {
			return fmt.Errorf("read latest action for migration %d: %w", target.Version, err)
		}
		if latestID != targetID || (latestAction != "up" && latestAction != "forward") {
			return errors.New("only the latest active execution of a migration can be rolled back")
		}
		var higherActive bool
		if err := tx.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM db_platform_migrations current
				WHERE current.version > $1 AND current.status='success'
				  AND current.action IN ('up', 'forward')
				  AND NOT EXISTS (SELECT 1 FROM db_platform_migrations later WHERE later.version=current.version AND later.status='success' AND later.id > current.id)
			)`, target.Version).Scan(&higherActive); err != nil {
			return fmt.Errorf("check migration order: %w", err)
		}
		if higherActive {
			return errors.New("roll back later active migrations first")
		}
		sqlToRun, targetExecutionID = target.SQLDown, targetID
	} else {
		if target.Action != "rollback" {
			return errors.New("forward target must be a successful rollback execution")
		}
		var latestID int64
		if err := tx.QueryRowContext(ctx, `SELECT id FROM db_platform_migrations WHERE version=$1 AND status='success' ORDER BY id DESC LIMIT 1`, target.Version).Scan(&latestID); err != nil {
			return fmt.Errorf("read latest action for migration %d: %w", target.Version, err)
		}
		if latestID != targetID {
			return errors.New("only the latest rollback of a migration can be forwarded")
		}
		var higherActive bool
		if err := tx.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM db_platform_migrations current
				WHERE current.version > $1 AND current.status='success'
				  AND current.action IN ('up', 'forward')
				  AND NOT EXISTS (SELECT 1 FROM db_platform_migrations later WHERE later.version=current.version AND later.status='success' AND later.id > current.id)
			)`, target.Version).Scan(&higherActive); err != nil {
			return fmt.Errorf("check migration order: %w", err)
		}
		if higherActive {
			return errors.New("forward later migrations in order after this migration")
		}
		var lowerInactive bool
		if err := tx.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM db_platform_migrations current
				WHERE current.version < $1 AND current.status='success'
				  AND current.action='rollback'
				  AND NOT EXISTS (SELECT 1 FROM db_platform_migrations later WHERE later.version=current.version AND later.status='success' AND later.id > current.id)
			)`, target.Version).Scan(&lowerInactive); err != nil {
			return fmt.Errorf("check forward order: %w", err)
		}
		if lowerInactive {
			return errors.New("forward earlier rolled-back migrations first")
		}
		sqlToRun, targetExecutionID = target.SQL, targetID
	}
	if strings.TrimSpace(sqlToRun) == "" {
		return errors.New("migration action SQL is empty")
	}

	if _, err := tx.ExecContext(ctx, sqlToRun); err != nil {
		_ = tx.Rollback()
		recordErr := m.recordActionAttempt(ctx, target, targetExecutionID, action, executedBy, "failed", err.Error())
		if recordErr != nil {
			return fmt.Errorf("execute %s for migration %d: %w; record failure: %v", action, target.Version, err, recordErr)
		}
		return fmt.Errorf("execute %s for migration %d: %w", action, target.Version, err)
	}
	if err := insertActionAttempt(ctx, tx, target, targetExecutionID, action, executedBy, "success", ""); err != nil {
		return fmt.Errorf("record %s for migration %d: %w", action, target.Version, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit %s for migration %d: %w", action, target.Version, err)
	}
	return nil
}

func (m *Manager) recordActionAttempt(ctx context.Context, target Migration, targetExecutionID int64, action, executedBy, status, message string) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin action history transaction: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtext('db_platform_migrations'))`); err != nil {
		return fmt.Errorf("lock migration history: %w", err)
	}
	if err := insertActionAttempt(ctx, tx, target, targetExecutionID, action, executedBy, status, message); err != nil {
		return err
	}
	return tx.Commit()
}

func insertActionAttempt(ctx context.Context, tx *sql.Tx, target Migration, targetExecutionID int64, action, executedBy, status, message string) error {
	var sqlToRun string
	if action == "rollback" {
		sqlToRun = target.SQLDown
	} else {
		sqlToRun = target.SQL
	}
	_, err := tx.ExecContext(ctx, `
		INSERT INTO db_platform_migrations
			(version, sql_text, executed_sql, sql_down, is_irreversible, irreversible_reason, executed_by, action, target_execution_id, status, error_message)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), $7, $8, $9, $10, NULLIF($11, ''))`,
		target.Version, target.SQL, sqlToRun, target.SQLDown, target.Irreversible, nullStringText(target.IrreversibleReason), executedBy, action, targetExecutionID, status, message)
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

func (m *Manager) recordMigrationFailure(ctx context.Context, version int64, sqlUp, sqlDown string, irreversible bool, reason, executedBy, message string) error {
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
		(version, sql_text, executed_sql, sql_down, is_irreversible, irreversible_reason, executed_by, status, error_message)
		VALUES ($1, $2, $2, $3, $4, NULLIF($5, ''), $6, 'failed', $7)`,
		version, sqlUp, sqlDown, irreversible, reason, executedBy, message); err != nil {
		return fmt.Errorf("record failed migration: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit failed migration record: %w", err)
	}
	return nil
}
