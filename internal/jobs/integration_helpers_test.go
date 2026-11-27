package jobs

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"

	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/dbplatform"
)

func openJobsIntegrationDB(t *testing.T) *sql.DB {
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
	if err := EnsureSchema(context.Background(), db); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	return db
}

func createJobsTestDatabase(t *testing.T, db *sql.DB) int {
	t.Helper()
	var id int
	name := "jobs_test_" + strings.ReplaceAll(newID(), "-", "")
	if err := db.QueryRow(`
		INSERT INTO database_connections (database_name,name,dbname,host,port,username,password)
		VALUES ($1,$1,$1,'localhost',5432,'test','test') RETURNING id`, name).Scan(&id); err != nil {
		t.Fatalf("insert test database connection: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM migration_change_requests WHERE database_id=$1`, id)
		_, _ = db.Exec(`DELETE FROM database_connections WHERE id=$1`, id)
	})
	return id
}

func createJobsTestUser(t *testing.T, db *sql.DB, role string) (int, string) {
	t.Helper()
	name := "jobs_user_" + strings.ReplaceAll(newID(), "-", "")
	var id int
	if err := db.QueryRow(`
		INSERT INTO users (username,password_hash,name,role)
		VALUES ($1,'test-hash',$1,$2) RETURNING id`, name, role).Scan(&id); err != nil {
		t.Fatalf("insert test user: %v", err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM users WHERE id=$1`, id) })
	return id, name
}

func grantJobsTestAccess(t *testing.T, db *sql.DB, databaseID, userID int, level string) {
	t.Helper()
	groupName := "jobs_group_" + strings.ReplaceAll(newID(), "-", "")
	var groupID int
	if err := db.QueryRow(`INSERT INTO groups (name,leader_user_id) VALUES ($1,$2) RETURNING id`,
		groupName, userID).Scan(&groupID); err != nil {
		t.Fatalf("insert test group: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO user_groups (user_id,group_id) VALUES ($1,$2)`, userID, groupID); err != nil {
		t.Fatalf("add test user to group: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO group_database_access (group_id,database_id,db_id,access_level)
		VALUES ($1,$2,$2,$3)`, groupID, databaseID, level); err != nil {
		t.Fatalf("grant test access: %v", err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM groups WHERE id=$1`, groupID) })
}

func enqueueJobsTestJob(t *testing.T, ctx context.Context, store *Store, databaseID int, request EnqueueRequest) Job {
	t.Helper()
	request.DatabaseID = databaseID
	if request.SQL == "" {
		request.SQL = "SELECT 1"
	}
	if request.RequestedBy == "" {
		request.RequestedBy = "integration-test"
	}
	job, err := store.Enqueue(ctx, request)
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	return job
}
