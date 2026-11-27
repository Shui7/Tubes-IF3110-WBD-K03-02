package workflow

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/dbplatform"
)

func openWorkflowIntegrationDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	db, err := dbplatform.InitPlatformDB(dsn)
	if err != nil {
		t.Fatalf("InitPlatformDB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestApprovalCreateRespondAndSetStatus(t *testing.T) {
	db := openWorkflowIntegrationDB(t)
	suffix := strings.ReplaceAll(newTestWorkflowID(), "-", "")
	requesterName, leaderName, databaseName := "requester_"+suffix, "leader_"+suffix, "workflow_"+suffix

	var requesterID, leaderID, databaseID, groupID int
	if err := db.QueryRow(`
		INSERT INTO users (username,password_hash,name,role)
		VALUES ($1,'test-hash',$1,'User') RETURNING id`, requesterName).Scan(&requesterID); err != nil {
		t.Fatalf("insert requester: %v", err)
	}
	if err := db.QueryRow(`
		INSERT INTO users (username,password_hash,name,role)
		VALUES ($1,'test-hash',$1,'User') RETURNING id`, leaderName).Scan(&leaderID); err != nil {
		t.Fatalf("insert leader: %v", err)
	}
	if err := db.QueryRow(`
		INSERT INTO database_connections (database_name,name,dbname,host,port,username,password)
		VALUES ($1,$1,$1,'localhost',5432,'test','test') RETURNING id`, databaseName).Scan(&databaseID); err != nil {
		t.Fatalf("insert test database: %v", err)
	}
	if err := db.QueryRow(`INSERT INTO groups (name,leader_user_id) VALUES ($1,$2) RETURNING id`,
		"group_"+suffix, leaderID).Scan(&groupID); err != nil {
		t.Fatalf("insert approval group: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO group_database_access (group_id,database_id,db_id,access_level) VALUES ($1,$2,$2,'edit')`,
		groupID, databaseID); err != nil {
		t.Fatalf("grant database access: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM migration_change_requests WHERE database_id=$1`, databaseID)
		_, _ = db.Exec(`DELETE FROM groups WHERE id=$1`, groupID)
		_, _ = db.Exec(`DELETE FROM users WHERE id IN ($1,$2)`, requesterID, leaderID)
		_, _ = db.Exec(`DELETE FROM database_connections WHERE id=$1`, databaseID)
	})

	createRequest := func() Request {
		t.Helper()
		request, err := CreateRequest(db, databaseID, requesterID, requesterName, "up",
			"CREATE TABLE workflow_test_items (id INT)", "DROP TABLE workflow_test_items",
			false, "", 0, "plain", 0, "", "")
		if err != nil {
			t.Fatalf("CreateRequest: %v", err)
		}
		if request.Status != "pending_review" {
			t.Fatalf("CreateRequest status = %q, want pending_review", request.Status)
		}
		return request
	}

	request := createRequest()
	listed, err := ListRequests(db, databaseID, leaderID)
	if err != nil || len(listed) != 1 || len(listed[0].Approvals) != 1 || !listed[0].Approvals[0].CanRespond {
		t.Fatalf("ListRequests = %+v, %v; want one response assigned to leader", listed, err)
	}
	if _, _, err := Respond(db, request.ID, requesterID, requesterName, "approved", ""); err == nil {
		t.Fatal("Respond accepted a user who is not the assigned leader")
	}
	ready, rejected, err := Respond(db, request.ID, leaderID, leaderName, "approved", "reviewed")
	if err != nil || !ready || rejected {
		t.Fatalf("approval Respond = ready %v rejected %v err %v; want ready approval", ready, rejected, err)
	}
	if err := SetStatus(db, request.ID, "pending_queue", requesterID, requesterName, "approved"); err != nil {
		t.Fatalf("SetStatus pending_queue: %v", err)
	}
	if err := SetStatus(db, request.ID, "running", 0, "test-worker", "started"); err != nil {
		t.Fatalf("SetStatus running: %v", err)
	}
	if err := SetStatus(db, request.ID, "success", 0, "test-worker", "done"); err != nil {
		t.Fatalf("SetStatus success: %v", err)
	}
	if err := SetStatus(db, request.ID, "running", 0, "test-worker", "invalid"); err != sql.ErrNoRows {
		t.Fatalf("invalid SetStatus transition = %v, want sql.ErrNoRows", err)
	}

	rejectedRequest := createRequest()
	ready, rejected, err = Respond(db, rejectedRequest.ID, leaderID, leaderName, "rejected", "not safe")
	if err != nil || ready || !rejected {
		t.Fatalf("rejection Respond = ready %v rejected %v err %v; want rejected", ready, rejected, err)
	}
	got, err := GetRequest(db, rejectedRequest.ID)
	if err != nil || got.Status != "rejected" {
		t.Fatalf("GetRequest = %+v, %v; want rejected", got, err)
	}
}

func TestAdminCanApproveLeaderlessGroupFallback(t *testing.T) {
	db := openWorkflowIntegrationDB(t)
	suffix := strings.ReplaceAll(newTestWorkflowID(), "-", "")
	requesterName, adminName, databaseName := "requester_"+suffix, "admin_"+suffix, "fallback_"+suffix

	var requesterID, adminID, databaseID, groupID int
	if err := db.QueryRow(`
		INSERT INTO users (username,password_hash,name,role)
		VALUES ($1,'test-hash',$1,'User') RETURNING id`, requesterName).Scan(&requesterID); err != nil {
		t.Fatalf("insert requester: %v", err)
	}
	if err := db.QueryRow(`
		INSERT INTO users (username,password_hash,name,role,is_admin)
		VALUES ($1,'test-hash',$1,'Admin',TRUE) RETURNING id`, adminName).Scan(&adminID); err != nil {
		t.Fatalf("insert admin: %v", err)
	}
	if err := db.QueryRow(`
		INSERT INTO database_connections (database_name,name,dbname,host,port,username,password)
		VALUES ($1,$1,$1,'localhost',5432,'test','test') RETURNING id`, databaseName).Scan(&databaseID); err != nil {
		t.Fatalf("insert test database: %v", err)
	}
	if err := db.QueryRow(`INSERT INTO groups (name) VALUES ($1) RETURNING id`, "leaderless_"+suffix).Scan(&groupID); err != nil {
		t.Fatalf("insert leaderless group: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO group_database_access (group_id,database_id,db_id,access_level)
		VALUES ($1,$2,$2,'edit')`, groupID, databaseID); err != nil {
		t.Fatalf("grant database access: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM migration_change_requests WHERE database_id=$1`, databaseID)
		_, _ = db.Exec(`DELETE FROM groups WHERE id=$1`, groupID)
		_, _ = db.Exec(`DELETE FROM users WHERE id IN ($1,$2)`, requesterID, adminID)
		_, _ = db.Exec(`DELETE FROM database_connections WHERE id=$1`, databaseID)
	})

	request, err := CreateRequest(db, databaseID, requesterID, requesterName, "up",
		"CREATE TABLE fallback_test_items (id INT)", "DROP TABLE fallback_test_items",
		false, "", 0, "plain", 0, "", "")
	if err != nil {
		t.Fatalf("CreateRequest without a group leader: %v", err)
	}
	leaderlessView, err := ListRequests(db, databaseID, requesterID)
	if err != nil || len(leaderlessView) != 1 || leaderlessView[0].Approvals[0].CanRespond {
		t.Fatalf("requester approval view = %+v, %v; want no approval permission", leaderlessView, err)
	}
	adminView, err := ListRequests(db, databaseID, adminID)
	if err != nil || len(adminView) != 1 || len(adminView[0].Approvals) != 1 ||
		adminView[0].Approvals[0].LeaderID != 0 ||
		adminView[0].Approvals[0].LeaderName != "Admin fallback (leader unassigned)" ||
		!adminView[0].Approvals[0].CanRespond {
		t.Fatalf("admin approval view = %+v, %v; want an actionable admin fallback", adminView, err)
	}
	if _, _, err := Respond(db, request.ID, requesterID, requesterName, "approved", ""); err == nil {
		t.Fatal("non-admin user approved the administrator fallback")
	}
	ready, rejected, err := Respond(db, request.ID, adminID, adminName, "approved", "leader not assigned")
	if err != nil || !ready || rejected {
		t.Fatalf("admin fallback Respond = ready %v rejected %v err %v; want approved", ready, rejected, err)
	}
	final, err := GetRequest(db, request.ID)
	if err != nil || final.Status != "approved" {
		t.Fatalf("request after admin fallback = %+v, %v; want approved", final, err)
	}
}

func TestAdminCanCreateRequestWithoutAccessGroups(t *testing.T) {
	db := openWorkflowIntegrationDB(t)
	suffix := strings.ReplaceAll(newTestWorkflowID(), "-", "")
	adminName, databaseName := "admin_"+suffix, "no_groups_"+suffix

	var adminID, databaseID int
	if err := db.QueryRow(`
		INSERT INTO users (username,password_hash,name,role,is_admin)
		VALUES ($1,'test-hash',$1,'Admin',TRUE) RETURNING id`, adminName).Scan(&adminID); err != nil {
		t.Fatalf("insert admin: %v", err)
	}
	if err := db.QueryRow(`
		INSERT INTO database_connections (database_name,name,dbname,host,port,username,password)
		VALUES ($1,$1,$1,'localhost',5432,'test','test') RETURNING id`, databaseName).Scan(&databaseID); err != nil {
		t.Fatalf("insert test database: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM migration_change_requests WHERE database_id=$1`, databaseID)
		_, _ = db.Exec(`DELETE FROM database_connections WHERE id=$1`, databaseID)
		_, _ = db.Exec(`DELETE FROM users WHERE id=$1`, adminID)
	})

	request, err := CreateRequest(db, databaseID, adminID, adminName, "up",
		"CREATE TABLE no_groups_test_items (id INT)", "DROP TABLE no_groups_test_items",
		false, "", 0, "plain", 0, "", "")
	if err != nil {
		t.Fatalf("CreateRequest without access groups: %v", err)
	}
	requests, err := ListRequests(db, databaseID, adminID)
	if err != nil || len(requests) != 1 || len(requests[0].Approvals) != 1 ||
		requests[0].Approvals[0].GroupName != "No access groups configured" ||
		requests[0].Approvals[0].LeaderName != "Admin fallback (leader unassigned)" ||
		!requests[0].Approvals[0].CanRespond {
		t.Fatalf("request approvals = %+v, %v; want one actionable Admin fallback", requests, err)
	}
	ready, rejected, err := Respond(db, request.ID, adminID, adminName, "approved", "no access groups configured")
	if err != nil || !ready || rejected {
		t.Fatalf("Admin fallback Respond = ready %v rejected %v err %v; want approved", ready, rejected, err)
	}
}

func newTestWorkflowID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
