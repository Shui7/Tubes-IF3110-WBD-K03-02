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
			version BIGINT PRIMARY KEY,
			sql_text TEXT NOT NULL,
			executed_by TEXT NOT NULL,
			status TEXT NOT NULL CHECK (status IN ('success', 'failed')),
			error_message TEXT,
			executed_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return fmt.Errorf("ensure migration table: %w", err)
	}
	if _, err := m.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS db_platform_migration_drafts (
			id BIGSERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			sql_text TEXT NOT NULL,
			created_by TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`); err != nil {
		return fmt.Errorf("ensure migration drafts table: %w", err)
	}
	return nil
}

func (m *Manager) ListMigrations(ctx context.Context) ([]Migration, error) {
	rows, err := m.db.QueryContext(ctx, `
		SELECT version, sql_text, executed_by, status, error_message, executed_at
		FROM db_platform_migrations
		ORDER BY version`)
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
			&migration.Version, &migration.SQL, &migration.ExecutedBy,
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
		SELECT version, sql_text, executed_by, status, error_message, executed_at
		FROM db_platform_migrations WHERE version = $1`, version).Scan(
		&migration.Version, &migration.SQL, &migration.ExecutedBy,
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
	if version < 1 {
		return errors.New("migration version must be positive")
	}
	if strings.TrimSpace(migrationSQL) == "" {
		return errors.New("migration SQL is required")
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

	if _, err := tx.ExecContext(ctx, migrationSQL); err != nil {
		_ = tx.Rollback()
		recordErr := m.recordMigrationFailure(ctx, version, migrationSQL, executedBy, err.Error())
		if recordErr != nil {
			return fmt.Errorf("execute migration %d: %w; record failure: %v", version, err, recordErr)
		}
		return fmt.Errorf("execute migration %d: %w", version, err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO db_platform_migrations
			(version, sql_text, executed_by, status)
		VALUES ($1, $2, $3, 'success')`,
		version, migrationSQL, executedBy,
	); err != nil {
		return fmt.Errorf("record migration %d: %w", version, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration %d: %w", version, err)
	}
	return nil
}

func (m *Manager) recordMigrationFailure(ctx context.Context, version int64, migrationSQL, executedBy, message string) error {
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
			(version, sql_text, executed_by, status, error_message)
		VALUES ($1, $2, $3, 'failed', $4)
		ON CONFLICT (version) DO UPDATE
		SET sql_text = EXCLUDED.sql_text,
		    executed_by = EXCLUDED.executed_by,
		    status = EXCLUDED.status,
		    error_message = EXCLUDED.error_message,
		    executed_at = CURRENT_TIMESTAMP`,
		version, migrationSQL, executedBy, message); err != nil {
		return fmt.Errorf("record failed migration: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit failed migration record: %w", err)
	}
	return nil
}

func (m *Manager) CreateMigrationDraft(ctx context.Context, name, migrationSQL, createdBy string) (MigrationDraft, error) {
	if strings.TrimSpace(name) == "" {
		return MigrationDraft{}, errors.New("migration draft name is required")
	}
	if strings.TrimSpace(migrationSQL) == "" {
		return MigrationDraft{}, errors.New("migration draft SQL is required")
	}
	if strings.TrimSpace(createdBy) == "" {
		return MigrationDraft{}, errors.New("migration draft creator is required")
	}
	if err := m.ensureMigrationDraftsTable(ctx); err != nil {
		return MigrationDraft{}, err
	}

	var draft MigrationDraft
	err := m.db.QueryRowContext(ctx, `
		INSERT INTO db_platform_migration_drafts (name, sql_text, created_by)
		VALUES ($1, $2, $3)
		RETURNING id, name, sql_text, created_by, created_at, updated_at`,
		strings.TrimSpace(name), migrationSQL, strings.TrimSpace(createdBy),
	).Scan(&draft.ID, &draft.Name, &draft.SQL, &draft.CreatedBy, &draft.CreatedAt, &draft.UpdatedAt)
	if err != nil {
		return MigrationDraft{}, fmt.Errorf("create migration draft: %w", err)
	}
	return draft, nil
}

func (m *Manager) UpdateMigrationDraft(ctx context.Context, id int64, name, migrationSQL string) (MigrationDraft, error) {
	if id < 1 {
		return MigrationDraft{}, errors.New("migration draft id must be positive")
	}
	if strings.TrimSpace(name) == "" {
		return MigrationDraft{}, errors.New("migration draft name is required")
	}
	if strings.TrimSpace(migrationSQL) == "" {
		return MigrationDraft{}, errors.New("migration draft SQL is required")
	}
	if err := m.ensureMigrationDraftsTable(ctx); err != nil {
		return MigrationDraft{}, err
	}

	var draft MigrationDraft
	err := m.db.QueryRowContext(ctx, `
		UPDATE db_platform_migration_drafts
		SET name = $1, sql_text = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3
		RETURNING id, name, sql_text, created_by, created_at, updated_at`,
		strings.TrimSpace(name), migrationSQL, id,
	).Scan(&draft.ID, &draft.Name, &draft.SQL, &draft.CreatedBy, &draft.CreatedAt, &draft.UpdatedAt)
	if err != nil {
		return MigrationDraft{}, fmt.Errorf("update migration draft %d: %w", id, err)
	}
	return draft, nil
}

func (m *Manager) GetMigrationDraft(ctx context.Context, id int64) (MigrationDraft, error) {
	if id < 1 {
		return MigrationDraft{}, errors.New("migration draft id must be positive")
	}
	var draft MigrationDraft
	err := m.db.QueryRowContext(ctx, `
		SELECT id, name, sql_text, created_by, created_at, updated_at
		FROM db_platform_migration_drafts
		WHERE id = $1`, id,
	).Scan(&draft.ID, &draft.Name, &draft.SQL, &draft.CreatedBy, &draft.CreatedAt, &draft.UpdatedAt)
	if isUndefinedTableError(err) {
		return MigrationDraft{}, sql.ErrNoRows
	}
	if err != nil {
		return MigrationDraft{}, fmt.Errorf("get migration draft %d: %w", id, err)
	}
	return draft, nil
}

func (m *Manager) ListMigrationDrafts(ctx context.Context) ([]MigrationDraft, error) {
	rows, err := m.db.QueryContext(ctx, `
		SELECT id, name, sql_text, created_by, created_at, updated_at
		FROM db_platform_migration_drafts
		ORDER BY updated_at DESC, id DESC`)
	if err != nil && isUndefinedTableError(err) {
		return []MigrationDraft{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("list migration drafts: %w", err)
	}
	defer rows.Close()

	var drafts []MigrationDraft
	for rows.Next() {
		var draft MigrationDraft
		if err := rows.Scan(&draft.ID, &draft.Name, &draft.SQL, &draft.CreatedBy, &draft.CreatedAt, &draft.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan migration draft: %w", err)
		}
		drafts = append(drafts, draft)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate migration drafts: %w", err)
	}
	return drafts, nil
}

func (m *Manager) DeleteMigrationDraft(ctx context.Context, id int64) error {
	if id < 1 {
		return errors.New("migration draft id must be positive")
	}
	result, err := m.db.ExecContext(ctx, `DELETE FROM db_platform_migration_drafts WHERE id = $1`, id)
	if isUndefinedTableError(err) {
		return sql.ErrNoRows
	}
	if err != nil {
		return fmt.Errorf("delete migration draft %d: %w", id, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check deleted migration draft %d: %w", id, err)
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (m *Manager) ensureMigrationDraftsTable(ctx context.Context) error {
	_, err := m.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS db_platform_migration_drafts (
			id BIGSERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			sql_text TEXT NOT NULL,
			created_by TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return fmt.Errorf("ensure migration drafts table: %w", err)
	}
	return nil
}
