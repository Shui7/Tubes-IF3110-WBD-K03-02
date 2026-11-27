package jobs

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/dbmanager"
	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/managed"
	"github.com/lib/pq"
)

// Executor runs a job against the target database.
// TODO: Implement Executor
type Executor interface {
	NextVersion(ctx context.Context, databaseID int) (int64, error)
	// Execute runs the SQL in one transaction and records history.
	// Returns an error wrapping *dbmanager.CommitError when COMMIT fails.
	Execute(ctx context.Context, job Job, version int64) error
	// WasApplied answers "did this job's transaction commit?" after an uncertain outcome.
	WasApplied(ctx context.Context, job Job, version int64) (bool, error)
}

// Validator re-checks a job right before execution (approval still valid,
// version still next, user still has access, SQL hash unchanged, etc.).
// Return an error wrapping ErrValidation to fail the job permanently.
// TODO: Implement Validator
type Validator interface {
	Validate(ctx context.Context, job Job) error
}

// NoopValidator accepts everything.
type NoopValidator struct{}

func (NoopValidator) Validate(context.Context, Job) error { return nil }

// ManagerExecutor uses the existing dbmanager history (db_platform_migrations in the target database).
type ManagerExecutor struct {
	Registry *managed.Registry
}

func (e *ManagerExecutor) NextVersion(ctx context.Context, databaseID int) (int64, error) {
	m, err := e.Registry.Get(ctx, databaseID)
	if err != nil {
		return 0, err
	}
	return m.NextMigrationVersion(ctx)
}

func (e *ManagerExecutor) Execute(ctx context.Context, job Job, version int64) error {
	m, err := e.Registry.Get(ctx, job.DatabaseID)
	if err != nil {
		return err
	}
	switch job.Kind {
	case "", "up":
		return m.ExecuteMigrationDefinition(ctx, version, job.SQL, job.SQLDown, job.Irreversible, job.IrreversibleReason, job.RequestedBy)
	case "rollback":
		if !job.TargetExecutionID.Valid {
			return errors.New("queued rollback has no target execution ID")
		}
		target, err := m.GetMigrationExecution(ctx, job.TargetExecutionID.Int64)
		if err != nil {
			return err
		}
		if target.SQLDown != job.SQL {
			return errors.New("queued rollback SQL differs from the immutable migration definition")
		}
		return m.RollbackMigration(ctx, job.TargetExecutionID.Int64, job.RequestedBy)
	case "forward":
		if !job.TargetExecutionID.Valid {
			return errors.New("queued forward has no target execution ID")
		}
		target, err := m.GetMigrationExecution(ctx, job.TargetExecutionID.Int64)
		if err != nil {
			return err
		}
		if target.SQL != job.SQL {
			return errors.New("queued forward SQL differs from the immutable migration definition")
		}
		return m.ForwardMigration(ctx, job.TargetExecutionID.Int64, job.RequestedBy)
	default:
		return fmt.Errorf("unsupported migration job kind %q", job.Kind)
	}
}

// WasApplied checks whether a successful migration with the same version
// and SQL has already been recorded.
// TODO: check execution_id once history has it.
func (e *ManagerExecutor) WasApplied(ctx context.Context, job Job, version int64) (bool, error) {
	m, err := e.Registry.Get(ctx, job.DatabaseID)
	if err != nil {
		return false, err
	}
	if job.Kind == "rollback" || job.Kind == "forward" {
		migrations, err := m.ListMigrations(ctx)
		if err != nil {
			return false, err
		}
		for i := len(migrations) - 1; i >= 0; i-- {
			migration := migrations[i]
			if migration.Version == version && migration.Action == job.Kind && migration.TargetExecutionID.Valid && job.TargetExecutionID.Valid && migration.TargetExecutionID.Int64 == job.TargetExecutionID.Int64 {
				return migration.Status == "success", nil
			}
		}
		return false, nil
	}
	migration, err := m.GetMigration(ctx, version)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return migration.Status == "success" && migration.SQL == job.SQL, nil
}

// ErrValidation marks a job that must fail permanently because revalidation
// before execution rejected it (approval revoked, version conflict, etc.).
var ErrValidation = errors.New("job validation failed")

// SQLState returns the PostgreSQL SQLSTATE code inside err, or "".
func SQLState(err error) string {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return string(pqErr.Code)
	}
	return ""
}

// IsTransient reports whether err is worth retrying: connection problems,
// serialization failures, deadlocks, server shutdown, too many connections.
// Ordinary SQL errors (syntax, constraint, permission, etc.) are permanent.
func IsTransient(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, driver.ErrBadConn) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	code := SQLState(err)
	if code == "" {
		return false
	}
	if strings.HasPrefix(code, "08") { // connection exception
		return true
	}
	switch code { // source: https://www.postgresql.org/docs/current/errcodes-appendix.html
	case "40001", // serialization_failure
		"40P01", // deadlock_detected
		"55P03", // lock_not_available
		"53300", // too_many_connections
		"57P01", // admin_shutdown
		"57P02", // crash_shutdown
		"57P03": // cannot_connect_now
		return true
	}
	return false
}

// IsCommitUncertain reports whether COMMIT failed without a server verdict
// (connection lost, timeout, cancellation): the transaction may or may not have
// been committed. If the server itself answered with an error, the transaction
// is definitely not committed and this returns false.
func IsCommitUncertain(err error) bool {
	var commitErr *dbmanager.CommitError
	if !errors.As(err, &commitErr) {
		return false
	}
	return SQLState(commitErr.Err) == ""
}
