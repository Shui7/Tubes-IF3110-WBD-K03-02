package workflow

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/authz"
	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/managed"
)

var (
	ErrInvalidExecution = errors.New("job is not authorized by its migration request")
	ErrNotFullyApproved = errors.New("migration request is not fully approved")
)

// ExecutionJob contains only the queued values workflow must compare with the
// approved request. It deliberately avoids depending on the jobs package.
type ExecutionJob struct {
	ID                 string
	DatabaseID         int
	Kind               string
	Mode               string
	BatchSize          int
	BatchTable         string
	BatchColumn        string
	SQL                string
	SQLDown            string
	SQLSHA256          string
	Irreversible       bool
	IrreversibleReason string
	TargetExecutionID  sql.NullInt64
	WorkflowRequestID  sql.NullInt64
	Version            sql.NullInt64
	RequestedBy        string
}

type ExecutionValidator struct {
	PlatformDB *sql.DB
	Registry   *managed.Registry
}

// Validate checks the approved workflow request, requester access, queued SQL,
// and rollback/forward eligibility immediately before execution.
func (v ExecutionValidator) Validate(ctx context.Context, job ExecutionJob) error {
	if v.PlatformDB == nil {
		return errors.New("workflow validation database is unavailable")
	}
	if err := validateSQLHash(job.SQL, job.SQLSHA256); err != nil {
		return err
	}
	request, err := ValidateExecutionRequest(ctx, v.PlatformDB, job)
	if err != nil {
		return err
	}
	if request.Action != "rollback" && request.Action != "forward" {
		return nil
	}
	if v.Registry == nil || !job.TargetExecutionID.Valid {
		return invalidExecution("rollback/forward job has no target execution")
	}
	manager, err := v.Registry.Get(ctx, job.DatabaseID)
	if err != nil {
		return fmt.Errorf("open migration target database: %w", err)
	}
	defer manager.Close()

	target, err := manager.ValidateMigrationAction(ctx, job.TargetExecutionID.Int64, request.Action)
	if err != nil {
		return fmt.Errorf("%w: rollback/forward target is no longer eligible: %w", ErrInvalidExecution, err)
	}
	if target.SQL != request.SQLUp || target.SQLDown != request.SQLDown {
		return invalidExecution("approved SQL differs from the current immutable migration definition")
	}
	if (request.Action == "rollback" && target.SQLDown != job.SQL) ||
		(request.Action == "forward" && target.SQL != job.SQL) {
		return invalidExecution("queued SQL differs from the approved migration definition")
	}
	if !job.Version.Valid || target.Version != job.Version.Int64 {
		return invalidExecution("rollback/forward target version changed")
	}
	return nil
}

// ValidateExecutionRequest confirms the request is fully approved, still
// matches the queued job, and its requester retains edit access.
func ValidateExecutionRequest(ctx context.Context, db *sql.DB, job ExecutionJob) (Request, error) {
	if db == nil {
		return Request{}, errors.New("workflow database is unavailable")
	}
	if !job.WorkflowRequestID.Valid {
		return Request{}, invalidExecution("job has no workflow request")
	}
	request, err := GetFullyApprovedRequest(ctx, db, job.WorkflowRequestID.Int64)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, ErrNotFullyApproved) {
			return Request{}, fmt.Errorf("%w: %v", ErrInvalidExecution, err)
		}
		return Request{}, fmt.Errorf("load approved migration request: %w", err)
	}
	if err := validateRequestMatchesExecution(request, job); err != nil {
		return Request{}, err
	}

	level, err := authz.GetUserDatabaseAccess(ctx, db, request.RequesterID, request.DatabaseID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Request{}, invalidExecution("requester no longer exists")
		}
		return Request{}, fmt.Errorf("recheck requester database access: %w", err)
	}
	if level != authz.AccessEdit {
		return Request{}, invalidExecution("requester no longer has edit access to the target database")
	}
	return request, nil
}

// GetFullyApprovedRequest returns a request only while execution is in progress
// and every approval recorded for it remains approved.
func GetFullyApprovedRequest(ctx context.Context, db *sql.DB, requestID int64) (Request, error) {
	var request Request
	err := db.QueryRowContext(ctx, `
		SELECT id, database_id, requester_user_id, requester_name, action, sql_up, sql_down,
		       is_irreversible, COALESCE(irreversible_reason,''), target_execution_id, mode,
		       COALESCE(batch_size,0), COALESCE(batch_table,''), COALESCE(batch_column,''),
		       status, error_message, created_at, resume_job_id
		FROM migration_change_requests WHERE id=$1`, requestID).Scan(
		&request.ID, &request.DatabaseID, &request.RequesterID, &request.RequesterName,
		&request.Action, &request.SQLUp, &request.SQLDown, &request.Irreversible,
		&request.IrreversibleReason, &request.TargetExecutionID, &request.Mode,
		&request.BatchSize, &request.BatchTable, &request.BatchColumn, &request.Status,
		&request.Error, &request.CreatedAt, &request.ResumeJobID)
	if err != nil {
		return Request{}, err
	}
	if request.Status != "running" {
		return Request{}, fmt.Errorf("%w: status is %q", ErrNotFullyApproved, request.Status)
	}
	var total, approved int
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(*), COUNT(*) FILTER (WHERE decision='approved')
		FROM migration_change_approvals WHERE request_id=$1`, requestID).Scan(&total, &approved); err != nil {
		return Request{}, fmt.Errorf("verify migration approvals: %w", err)
	}
	if total == 0 || approved != total {
		return Request{}, ErrNotFullyApproved
	}
	return request, nil
}

func validateSQLHash(sqlText, storedHash string) error {
	if storedHash == "" {
		return invalidExecution("queued SQL hash is missing")
	}
	sum := sha256.Sum256([]byte(sqlText))
	if !strings.EqualFold(storedHash, hex.EncodeToString(sum[:])) {
		return invalidExecution("queued SQL hash does not match the stored migration SQL")
	}
	return nil
}

func validateRequestMatchesExecution(request Request, job ExecutionJob) error {
	kind := job.Kind
	if kind == "" {
		kind = "up"
	}
	sqlToRun := request.SQLUp
	if kind == "rollback" {
		sqlToRun = request.SQLDown
	}
	if !job.WorkflowRequestID.Valid || request.ID != job.WorkflowRequestID.Int64 ||
		request.DatabaseID != job.DatabaseID ||
		request.Action != kind ||
		request.Mode != job.Mode ||
		request.BatchSize != job.BatchSize ||
		request.BatchTable != job.BatchTable ||
		request.BatchColumn != job.BatchColumn ||
		(request.SQLUp != job.SQL && kind == "up") ||
		request.SQLDown != job.SQLDown ||
		request.Irreversible != job.Irreversible ||
		request.IrreversibleReason != job.IrreversibleReason ||
		request.RequesterName != job.RequestedBy ||
		sqlToRun != job.SQL ||
		!sameExecutionTarget(request.TargetExecutionID, job.TargetExecutionID) {
		return invalidExecution("queued job no longer matches its approved migration request")
	}
	if request.ResumeJobID.Valid && request.ResumeJobID.String != job.ID {
		return invalidExecution("batch continuation request references a different job")
	}
	return nil
}

func sameExecutionTarget(requestTarget, jobTarget sql.NullInt64) bool {
	return requestTarget.Valid == jobTarget.Valid &&
		(!requestTarget.Valid || requestTarget.Int64 == jobTarget.Int64)
}

func invalidExecution(message string) error {
	return fmt.Errorf("%w: %s", ErrInvalidExecution, message)
}
