package workflow

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrResumeRequestPending = errors.New("batch continuation is already awaiting approval")
	ErrResumeNotEligible    = errors.New("job is not eligible for batch continuation")
)

type requiredApproval struct {
	groupID int
	group   string
	leader  sql.NullInt64
	name    string
}

func loadRequiredApprovals(tx *sql.Tx, databaseID int) ([]requiredApproval, error) {
	rows, err := tx.Query(`
		SELECT DISTINCT g.id, g.name, g.leader_user_id, COALESCE(NULLIF(u.name, ''), u.username, '')
		FROM group_database_access gda
		JOIN groups g ON g.id = gda.group_id
		LEFT JOIN users u ON u.id = g.leader_user_id
		WHERE COALESCE(gda.database_id, gda.db_id) = $1
		ORDER BY g.id`, databaseID)
	if err != nil {
		return nil, fmt.Errorf("load database access groups: %w", err)
	}
	defer rows.Close()

	var required []requiredApproval
	for rows.Next() {
		var approval requiredApproval
		if err := rows.Scan(&approval.groupID, &approval.group, &approval.leader, &approval.name); err != nil {
			return nil, fmt.Errorf("scan approval group: %w", err)
		}
		required = append(required, approval)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate approval groups: %w", err)
	}
	if len(required) == 0 {
		return nil, errors.New("database has no access groups; configure at least one access group before requesting a migration")
	}
	for i := range required {
		approval := &required[i]
		if !approval.leader.Valid {
			approval.name = "Admin fallback (leader unassigned)"
			continue
		}
		if approval.name == "" {
			return nil, fmt.Errorf("leader for group %q could not be found", approval.group)
		}
	}
	return required, nil
}

// CreateResumeRequest opens a fresh approval round before a partial batch job
// can continue. The job remains partial and keeps its database lock meanwhile.
func CreateResumeRequest(db *sql.DB, jobID string, requesterID int, requesterName string) (Request, error) {
	if strings.TrimSpace(jobID) == "" || requesterID < 1 || strings.TrimSpace(requesterName) == "" {
		return Request{}, errors.New("job and requester are required")
	}

	tx, err := db.Begin()
	if err != nil {
		return Request{}, fmt.Errorf("begin batch resume approval: %w", err)
	}
	defer tx.Rollback()

	var databaseID, batchSize int
	var kind, mode, jobSQL, jobSQLDown, jobStatus string
	var jobIrreversible, holdsLock bool
	var jobIrreversibleReason string
	var targetID, originalRequestID sql.NullInt64
	err = tx.QueryRow(`
		SELECT database_id, kind, mode, sql_text, COALESCE(sql_down,''), is_irreversible,
		       COALESCE(irreversible_reason,''), target_execution_id, workflow_request_id,
		       COALESCE(batch_size,0), status, holds_lock
		FROM migration_jobs WHERE id=$1 FOR UPDATE`, jobID).Scan(
		&databaseID, &kind, &mode, &jobSQL, &jobSQLDown, &jobIrreversible,
		&jobIrreversibleReason, &targetID, &originalRequestID, &batchSize, &jobStatus, &holdsLock)
	if err != nil {
		return Request{}, fmt.Errorf("load partial batch job: %w", err)
	}
	if jobStatus != "partial" || !holdsLock || kind != "up" || mode != "batch" || !originalRequestID.Valid {
		return Request{}, ErrResumeNotEligible
	}
	var activeResume bool
	if err := tx.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM migration_change_requests
			WHERE resume_job_id=$1 AND status IN ('pending_review','approved','pending_queue','running')
		)`, jobID).Scan(&activeResume); err != nil {
		return Request{}, fmt.Errorf("check pending batch continuation approval: %w", err)
	}
	if activeResume {
		return Request{}, ErrResumeRequestPending
	}

	var source Request
	err = tx.QueryRow(`
		SELECT id, database_id, requester_user_id, requester_name, action, sql_up, sql_down,
		       is_irreversible, COALESCE(irreversible_reason,''), target_execution_id, mode,
		       COALESCE(batch_size,0), COALESCE(batch_table,''), COALESCE(batch_column,''), status
		FROM migration_change_requests WHERE id=$1 FOR UPDATE`, originalRequestID.Int64).Scan(
		&source.ID, &source.DatabaseID, &source.RequesterID, &source.RequesterName, &source.Action,
		&source.SQLUp, &source.SQLDown, &source.Irreversible, &source.IrreversibleReason,
		&source.TargetExecutionID, &source.Mode, &source.BatchSize, &source.BatchTable, &source.BatchColumn, &source.Status)
	if err != nil {
		return Request{}, fmt.Errorf("load original batch request: %w", err)
	}
	if source.DatabaseID != databaseID || source.Status != "partial" ||
		source.Action != "up" || source.Mode != "batch" ||
		source.SQLUp != jobSQL || source.SQLDown != jobSQLDown ||
		source.Irreversible != jobIrreversible || source.IrreversibleReason != jobIrreversibleReason ||
		source.BatchSize != batchSize || !sameNullableInt(source.TargetExecutionID, targetID) {
		return Request{}, ErrResumeNotEligible
	}

	required, err := loadRequiredApprovals(tx, databaseID)
	if err != nil {
		return Request{}, err
	}

	var request Request
	err = tx.QueryRow(`
		INSERT INTO migration_change_requests
			(database_id, requester_user_id, requester_name, action, sql_up, sql_down,
			 is_irreversible, irreversible_reason, target_execution_id, mode, batch_size,
			 batch_table, batch_column, status, resume_job_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7,NULLIF($8,''),NULLIF($9,0),$10,$11,NULLIF($12,''),NULLIF($13,''),'pending_review',$14)
		RETURNING id, database_id, requester_user_id, requester_name, action, sql_up, sql_down,
		          is_irreversible, COALESCE(irreversible_reason,''), target_execution_id, mode,
		          batch_size, COALESCE(batch_table,''), COALESCE(batch_column,''), status,
		          error_message, created_at, resume_job_id`,
		databaseID, requesterID, requesterName, source.Action, source.SQLUp, source.SQLDown,
		source.Irreversible, source.IrreversibleReason, nullableIntValue(source.TargetExecutionID),
		source.Mode, source.BatchSize, source.BatchTable, source.BatchColumn, jobID,
	).Scan(&request.ID, &request.DatabaseID, &request.RequesterID, &request.RequesterName,
		&request.Action, &request.SQLUp, &request.SQLDown, &request.Irreversible,
		&request.IrreversibleReason, &request.TargetExecutionID, &request.Mode, &request.BatchSize,
		&request.BatchTable, &request.BatchColumn, &request.Status, &request.Error,
		&request.CreatedAt, &request.ResumeJobID)
	if err != nil {
		return Request{}, fmt.Errorf("create batch resume approval request: %w", err)
	}
	if err := insertEvent(tx, request.ID, "pending_review", requesterID, requesterName, "Batch continuation submitted for required approval"); err != nil {
		return Request{}, err
	}
	for _, approval := range required {
		if _, err := tx.Exec(`
			INSERT INTO migration_change_approvals (request_id, group_id, group_name, leader_user_id, leader_name)
			VALUES ($1,$2,$3,$4,$5)`, request.ID, approval.groupID, approval.group, approval.leader, approval.name); err != nil {
			return Request{}, fmt.Errorf("create batch resume approval: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return Request{}, fmt.Errorf("commit batch resume approval: %w", err)
	}
	return request, nil
}

func sameNullableInt(a, b sql.NullInt64) bool {
	return a.Valid == b.Valid && (!a.Valid || a.Int64 == b.Int64)
}

func nullableIntValue(value sql.NullInt64) int64 {
	if value.Valid {
		return value.Int64
	}
	return 0
}
