package jobs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"
	"testing"

	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/dbplatform"
	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/workflow"
)

func TestBatchResumeRequiresApprovalAndRevalidatesAccess(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	db, err := dbplatform.InitPlatformDB(dsn)
	if err != nil {
		t.Fatalf("InitPlatformDB: %v", err)
	}
	defer db.Close()
	if err := EnsureSchema(context.Background(), db); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}

	suffix := strings.ReplaceAll(newID(), "-", "")
	username := "resume_" + suffix
	groupName := "resume_" + suffix
	databaseName := "resume_" + suffix
	jobID := newID()

	var userID, groupID, databaseID int
	var sourceRequestID int64
	t.Cleanup(func() {
		if databaseID != 0 {
			_, _ = db.Exec(`DELETE FROM migration_change_requests WHERE database_id=$1`, databaseID)
			_, _ = db.Exec(`DELETE FROM migration_jobs WHERE database_id=$1`, databaseID)
		}
		if groupID != 0 {
			_, _ = db.Exec(`DELETE FROM groups WHERE id=$1`, groupID)
		}
		if userID != 0 {
			_, _ = db.Exec(`DELETE FROM users WHERE id=$1`, userID)
		}
		if databaseID != 0 {
			_, _ = db.Exec(`DELETE FROM database_connections WHERE id=$1`, databaseID)
		}
	})

	if err := db.QueryRow(`
		INSERT INTO users (username,password_hash,name,role)
		VALUES ($1,'test-hash',$1,'User') RETURNING id`, username).Scan(&userID); err != nil {
		t.Fatalf("insert requester: %v", err)
	}
	if err := db.QueryRow(`INSERT INTO groups (name,leader_user_id) VALUES ($1,$2) RETURNING id`,
		groupName, userID).Scan(&groupID); err != nil {
		t.Fatalf("insert approval group: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO user_groups (user_id,group_id) VALUES ($1,$2)`, userID, groupID); err != nil {
		t.Fatalf("add requester to group: %v", err)
	}
	if err := db.QueryRow(`
		INSERT INTO database_connections (database_name,name,dbname,host,port,username,password)
		VALUES ($1,$1,$1,'localhost',5432,'test','test') RETURNING id`, databaseName).Scan(&databaseID); err != nil {
		t.Fatalf("insert database connection: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO group_database_access (group_id,database_id,db_id,access_level)
		VALUES ($1,$2,$2,'edit')`, groupID, databaseID); err != nil {
		t.Fatalf("grant requester edit access: %v", err)
	}

	const sqlText = `UPDATE batch_items SET value = value + 1 WHERE id > COALESCE($1::bigint,0) AND id <= COALESCE($1::bigint,0) + $2::int`
	sum := sha256.Sum256([]byte(sqlText))
	sqlHash := hex.EncodeToString(sum[:])
	requesterName := username
	if err := db.QueryRow(`
		INSERT INTO migration_change_requests
			(database_id,requester_user_id,requester_name,action,sql_up,sql_down,
			 is_irreversible,irreversible_reason,mode,batch_size,status)
		VALUES ($1,$2,$3,'up',$4,'',TRUE,'Integration test','batch',10,'partial')
		RETURNING id`,
		databaseID, userID, requesterName, sqlText).Scan(&sourceRequestID); err != nil {
		t.Fatalf("insert original migration request: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO migration_jobs
			(id,database_id,kind,mode,name,sql_text,sql_down,is_irreversible,irreversible_reason,
			 workflow_request_id,sql_sha256,requested_by,status,holds_lock,batch_size,
			 batches_committed,rows_committed)
		VALUES ($1,$2,'up','batch','resume test',$3,'',TRUE,'Integration test',
		        $4,$5,$6,'partial',TRUE,10,2,20)`,
		jobID, databaseID, sqlText, sourceRequestID, sqlHash, requesterName); err != nil {
		t.Fatalf("insert partial batch job: %v", err)
	}

	resumeRequest, err := workflow.CreateResumeRequest(db, jobID, userID, requesterName)
	if err != nil {
		t.Fatalf("CreateResumeRequest: %v", err)
	}
	if resumeRequest.Status != "pending_review" || !resumeRequest.ResumeJobID.Valid ||
		resumeRequest.ResumeJobID.String != jobID {
		t.Fatalf("resume request = %+v, want pending approval for job %s", resumeRequest, jobID)
	}
	if job, err := NewStore(db).Get(context.Background(), jobID); err != nil || job.Status != StatusPartial {
		t.Fatalf("job changed before approval: job=%+v err=%v", job, err)
	}
	if _, err := NewStore(db).ResumeApproved(context.Background(), jobID, resumeRequest.ID, requesterName); err != ErrResumeNotApproved {
		t.Fatalf("ResumeApproved before all votes = %v, want ErrResumeNotApproved", err)
	}

	ready, rejected, err := workflow.Respond(db, resumeRequest.ID, userID, requesterName, "approved", "")
	if err != nil || !ready || rejected {
		t.Fatalf("Respond = ready %v, rejected %v, err %v; want approved", ready, rejected, err)
	}
	if err := workflow.SetStatus(db, resumeRequest.ID, "pending_queue", userID, requesterName, "approved"); err != nil {
		t.Fatalf("mark continuation queued: %v", err)
	}
	resumed, err := NewStore(db).ResumeApproved(context.Background(), jobID, resumeRequest.ID, requesterName)
	if err != nil {
		t.Fatalf("ResumeApproved: %v", err)
	}
	if resumed.Status != StatusPending || !resumed.WorkflowRequestID.Valid ||
		resumed.WorkflowRequestID.Int64 != resumeRequest.ID || resumed.BatchesCommitted != 2 {
		t.Fatalf("resumed job = %+v, want queued with its checkpoints and new approval request", resumed)
	}
	if err := workflow.SetStatus(db, resumeRequest.ID, "running", 0, "test-worker", "validating"); err != nil {
		t.Fatalf("mark continuation running: %v", err)
	}
	validator := workflow.ExecutionValidator{PlatformDB: db}
	if err := validator.Validate(context.Background(), workflowExecution(resumed)); err != nil {
		t.Fatalf("Validate with current edit access: %v", err)
	}

	if _, err := db.Exec(`DELETE FROM user_groups WHERE user_id=$1 AND group_id=$2`, userID, groupID); err != nil {
		t.Fatalf("revoke requester membership: %v", err)
	}
	if err := validator.Validate(context.Background(), workflowExecution(resumed)); err == nil {
		t.Fatal("Validate succeeded after requester edit access was revoked")
	}
}
