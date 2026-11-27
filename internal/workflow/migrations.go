package workflow

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Approval struct {
	GroupID    int
	GroupName  string
	LeaderID   int
	LeaderName string
	Decision   string
	Comment    string
	CanRespond bool
}

type Request struct {
	ID                 int64
	DatabaseID         int
	RequesterID        int
	RequesterName      string
	Action             string
	SQLUp              string
	SQLDown            string
	Irreversible       bool
	IrreversibleReason string
	Mode               string
	BatchSize          int
	BatchTable         string
	BatchColumn        string
	TargetExecutionID  sql.NullInt64
	ResumeJobID        sql.NullString
	Status             string
	Error              sql.NullString
	CreatedAt          time.Time
	Approvals          []Approval
	Events             []Event
}

type Event struct {
	Status     string
	ActorName  string
	Note       string
	OccurredAt time.Time
}

func CreateRequest(db *sql.DB, databaseID, requesterID int, requesterName, action, sqlUp, sqlDown string, irreversible bool, reason string, targetID int64, mode string, batchSize int, batchTable, batchColumn string) (Request, error) {
	if databaseID < 1 || requesterID < 1 || strings.TrimSpace(requesterName) == "" {
		return Request{}, errors.New("database and requester are required")
	}
	if action != "up" && action != "rollback" && action != "forward" {
		return Request{}, errors.New("invalid migration action")
	}
	if mode == "" {
		mode = "plain"
	}
	if mode != "plain" && mode != "batch" {
		return Request{}, errors.New("migration mode must be plain or batch")
	}
	if action != "up" && mode != "plain" {
		return Request{}, errors.New("rollback and forward requests must use plain mode")
	}
	if mode == "batch" {
		if batchSize < 1 || batchSize > 100000 {
			return Request{}, errors.New("batch size must be between 1 and 100000")
		}
		if (batchTable == "") != (batchColumn == "") {
			return Request{}, errors.New("batch by attribute needs both a table and a column")
		}
		if batchTable != "" {
			if strings.Contains(batchColumn, ".") || strings.Count(batchTable, ".") > 1 || !strings.Contains(sqlUp, "{{batch_filter}}") {
				return Request{}, errors.New("batch-by-attribute SQL needs a table, a plain unique column, and {{batch_filter}}")
			}
		} else if !strings.Contains(sqlUp, "$1") || !strings.Contains(sqlUp, "$2") {
			return Request{}, errors.New("custom batch SQL must use $1 for the cursor and $2 for batch size")
		}
	} else {
		batchSize, batchTable, batchColumn = 0, "", ""
	}
	if action == "up" {
		if strings.TrimSpace(sqlUp) == "" {
			return Request{}, errors.New("SQL up is required")
		}
		if irreversible && strings.TrimSpace(reason) == "" {
			return Request{}, errors.New("irreversible reason is required")
		}
		if !irreversible && strings.TrimSpace(sqlDown) == "" {
			return Request{}, errors.New("SQL down is required unless marked irreversible")
		}
		if irreversible {
			sqlDown = ""
		} else {
			reason = ""
		}
	} else if targetID < 1 {
		return Request{}, errors.New("target execution ID must be positive")
	}

	tx, err := db.Begin()
	if err != nil {
		return Request{}, fmt.Errorf("begin migration approval request: %w", err)
	}
	defer tx.Rollback()

	required, err := loadRequiredApprovals(tx, databaseID)
	if err != nil {
		return Request{}, err
	}

	var request Request
	err = tx.QueryRow(`
		INSERT INTO migration_change_requests
			(database_id, requester_user_id, requester_name, action, sql_up, sql_down, is_irreversible, irreversible_reason, target_execution_id, mode, batch_size, batch_table, batch_column, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,NULLIF($8,''),NULLIF($9,0),$10,$11,NULLIF($12,''),NULLIF($13,''),'draft')
		RETURNING id, database_id, requester_user_id, requester_name, action, sql_up, sql_down, is_irreversible, COALESCE(irreversible_reason,''), target_execution_id, mode, batch_size, COALESCE(batch_table,''), COALESCE(batch_column,''), status, error_message, created_at`,
		databaseID, requesterID, requesterName, action, sqlUp, sqlDown, irreversible, reason, targetID, mode, batchSize, batchTable, batchColumn,
	).Scan(&request.ID, &request.DatabaseID, &request.RequesterID, &request.RequesterName, &request.Action, &request.SQLUp, &request.SQLDown, &request.Irreversible, &request.IrreversibleReason, &request.TargetExecutionID, &request.Mode, &request.BatchSize, &request.BatchTable, &request.BatchColumn, &request.Status, &request.Error, &request.CreatedAt)
	if err != nil {
		return Request{}, fmt.Errorf("create migration approval request: %w", err)
	}
	if _, err := tx.Exec(`UPDATE migration_change_requests SET status='pending_review', updated_at=NOW() WHERE id=$1`, request.ID); err != nil {
		return Request{}, err
	}
	if err := insertEvent(tx, request.ID, "draft", requesterID, requesterName, "Request created"); err != nil {
		return Request{}, err
	}
	if err := insertEvent(tx, request.ID, "pending_review", requesterID, requesterName, "Submitted for required approval"); err != nil {
		return Request{}, err
	}
	for _, approval := range required {
		if _, err := tx.Exec(`
			INSERT INTO migration_change_approvals (request_id, group_id, group_name, leader_user_id, leader_name)
			VALUES ($1,$2,$3,$4,$5)`, request.ID, approval.groupID, approval.group, approval.leader, approval.name); err != nil {
			return Request{}, fmt.Errorf("create group approval: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return Request{}, fmt.Errorf("commit migration approval request: %w", err)
	}
	request.Status = "pending_review"
	return request, nil
}

func ListRequests(db *sql.DB, databaseID, userID int) ([]Request, error) {
	var isAdmin bool
	if err := db.QueryRow(`SELECT is_admin OR role='Admin' FROM users WHERE id=$1`, userID).Scan(&isAdmin); err != nil {
		return nil, fmt.Errorf("check migration approval role: %w", err)
	}
	rows, err := db.Query(`
		SELECT id, database_id, requester_user_id, requester_name, action, sql_up, sql_down,
		       is_irreversible, COALESCE(irreversible_reason,''), target_execution_id, mode, COALESCE(batch_size,0),
	       COALESCE(batch_table,''), COALESCE(batch_column,''), status, error_message, created_at
		FROM migration_change_requests WHERE database_id=$1 ORDER BY id DESC`, databaseID)
	if err != nil {
		return nil, fmt.Errorf("list migration requests: %w", err)
	}
	defer rows.Close()
	var requests []Request
	for rows.Next() {
		var request Request
		if err := rows.Scan(&request.ID, &request.DatabaseID, &request.RequesterID, &request.RequesterName,
			&request.Action, &request.SQLUp, &request.SQLDown, &request.Irreversible, &request.IrreversibleReason,
			&request.TargetExecutionID, &request.Mode, &request.BatchSize, &request.BatchTable, &request.BatchColumn,
			&request.Status, &request.Error, &request.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan migration request: %w", err)
		}
		approvals, err := db.Query(`
			SELECT group_id, group_name, leader_user_id, leader_name, decision, comment
			FROM migration_change_approvals WHERE request_id=$1 ORDER BY group_id`, request.ID)
		if err != nil {
			return nil, fmt.Errorf("list request approvals: %w", err)
		}
		for approvals.Next() {
			var approval Approval
			var leaderID sql.NullInt64
			if err := approvals.Scan(&approval.GroupID, &approval.GroupName, &leaderID, &approval.LeaderName, &approval.Decision, &approval.Comment); err != nil {
				approvals.Close()
				return nil, fmt.Errorf("scan request approval: %w", err)
			}
			if leaderID.Valid {
				approval.LeaderID = int(leaderID.Int64)
			}
			approval.CanRespond = request.Status == "pending_review" && approval.Decision == "pending" &&
				((leaderID.Valid && approval.LeaderID == userID) || (!leaderID.Valid && isAdmin))
			request.Approvals = append(request.Approvals, approval)
		}
		if err := approvals.Err(); err != nil {
			approvals.Close()
			return nil, fmt.Errorf("iterate request approvals: %w", err)
		}
		approvals.Close()
		events, err := db.Query(`SELECT status, actor_name, note, occurred_at FROM migration_change_events WHERE request_id=$1 ORDER BY id`, request.ID)
		if err != nil {
			return nil, fmt.Errorf("list request workflow events: %w", err)
		}
		for events.Next() {
			var event Event
			if err := events.Scan(&event.Status, &event.ActorName, &event.Note, &event.OccurredAt); err != nil {
				events.Close()
				return nil, fmt.Errorf("scan request workflow event: %w", err)
			}
			request.Events = append(request.Events, event)
		}
		if err := events.Err(); err != nil {
			events.Close()
			return nil, fmt.Errorf("iterate request workflow events: %w", err)
		}
		events.Close()
		requests = append(requests, request)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate migration requests: %w", err)
	}
	return requests, nil
}

func GetRequest(db *sql.DB, requestID int64) (Request, error) {
	var request Request
	err := db.QueryRow(`
		SELECT id, database_id, requester_user_id, requester_name, action, sql_up, sql_down,
		       is_irreversible, COALESCE(irreversible_reason,''), target_execution_id, mode, COALESCE(batch_size,0),
	       COALESCE(batch_table,''), COALESCE(batch_column,''), status, error_message, created_at, resume_job_id
		FROM migration_change_requests WHERE id=$1`, requestID).Scan(
		&request.ID, &request.DatabaseID, &request.RequesterID, &request.RequesterName, &request.Action,
		&request.SQLUp, &request.SQLDown, &request.Irreversible, &request.IrreversibleReason,
		&request.TargetExecutionID, &request.Mode, &request.BatchSize, &request.BatchTable, &request.BatchColumn,
		&request.Status, &request.Error, &request.CreatedAt, &request.ResumeJobID)
	if err != nil {
		return Request{}, err
	}
	return request, nil
}

// Respond records an assigned leader's vote or an Admin fallback vote.
// ready is true when all required approvals have been recorded.
func Respond(db *sql.DB, requestID int64, userID int, userName, decision, comment string) (ready, rejected bool, err error) {
	if decision != "approved" && decision != "rejected" {
		return false, false, errors.New("decision must be approved or rejected")
	}
	tx, err := db.Begin()
	if err != nil {
		return false, false, err
	}
	defer tx.Rollback()
	var status string
	if err := tx.QueryRow(`SELECT status FROM migration_change_requests WHERE id=$1 FOR UPDATE`, requestID).Scan(&status); err != nil {
		return false, false, err
	}
	if status != "pending_review" {
		return false, false, errors.New("request is not awaiting approval")
	}
	var adminFallback bool
	if err := tx.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM migration_change_approvals
			WHERE request_id=$1 AND leader_user_id IS NULL AND decision='pending'
		) AND EXISTS (
			SELECT 1 FROM users WHERE id=$2 AND (is_admin OR role='Admin')
		)`, requestID, userID).Scan(&adminFallback); err != nil {
		return false, false, err
	}
	result, err := tx.Exec(`
		UPDATE migration_change_approvals SET decision=$1, comment=$2, responded_at=NOW()
		WHERE request_id=$3 AND decision='pending'
		  AND (leader_user_id=$4 OR (
			leader_user_id IS NULL AND EXISTS (
				SELECT 1 FROM users WHERE id=$4 AND (is_admin OR role='Admin')
			)
		  ))`, decision, strings.TrimSpace(comment), requestID, userID)
	if err != nil {
		return false, false, err
	}
	updated, err := result.RowsAffected()
	if err != nil || updated < 1 {
		if err != nil {
			return false, false, err
		}
		return false, false, errors.New("only an assigned group leader or an Admin fallback can approve this request")
	}
	eventStatus, eventNote := "leader_"+decision, "Group leader decision"
	if adminFallback {
		eventStatus, eventNote = "admin_fallback_"+decision, "Administrator fallback decision"
	}
	if err := insertEvent(tx, requestID, eventStatus, userID, userName, eventNote); err != nil {
		return false, false, err
	}
	var pending int
	var hasRejection bool
	if err := tx.QueryRow(`
		SELECT COUNT(*) FILTER (WHERE decision='pending'), BOOL_OR(decision='rejected')
		FROM migration_change_approvals WHERE request_id=$1`, requestID).Scan(&pending, &hasRejection); err != nil {
		return false, false, err
	}
	if hasRejection {
		if _, err := tx.Exec(`UPDATE migration_change_requests SET status='rejected', updated_at=NOW() WHERE id=$1`, requestID); err != nil {
			return false, false, err
		}
		if err := insertEvent(tx, requestID, "rejected", userID, userName, "A required approver rejected the request"); err != nil {
			return false, false, err
		}
		rejected = true
	} else if pending == 0 {
		if _, err := tx.Exec(`UPDATE migration_change_requests SET status='approved', updated_at=NOW() WHERE id=$1`, requestID); err != nil {
			return false, false, err
		}
		if err := insertEvent(tx, requestID, "approved", userID, userName, "All required approvers approved"); err != nil {
			return false, false, err
		}
		ready = true
	}
	if err := tx.Commit(); err != nil {
		return false, false, err
	}
	return ready, rejected, nil
}

func SetStatus(db *sql.DB, requestID int64, status string, actorID int, actorName, note string) error {
	var allowed string
	switch status {
	case "pending_queue":
		allowed = "status IN ('approved', 'partial')"
	case "running":
		allowed = "status IN ('pending_queue', 'running', 'partial')"
	case "success", "partial":
		allowed = "status='running'"
	case "failed":
		allowed = "status IN ('pending_queue', 'running')"
	default:
		return fmt.Errorf("invalid workflow transition to %q", status)
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.Exec(`
		UPDATE migration_change_requests
		SET status=$1, error_message=CASE WHEN $1 IN ('failed','partial') THEN NULLIF($2,'') ELSE NULL END, updated_at=NOW()
		WHERE id=$3 AND `+allowed, status, note, requestID)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return sql.ErrNoRows
	}
	if err := insertEvent(tx, requestID, status, actorID, actorName, note); err != nil {
		return err
	}
	return tx.Commit()
}

func insertEvent(tx *sql.Tx, requestID int64, status string, actorID int, actorName, note string) error {
	_, err := tx.Exec(`INSERT INTO migration_change_events (request_id,status,actor_user_id,actor_name,note) VALUES ($1,$2,$3,$4,$5)`, requestID, status, actorID, actorName, note)
	return err
}
