package authz_test

import (
	"database/sql"
	"os"
	"strings"
	"testing"

	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/authz"
	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/dbplatform"
	_ "github.com/lib/pq"
)

func openTestDB(t *testing.T) *sql.DB {
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

// setupAuthzFixture creates a minimal set of users, groups, connections, and
// access grants for testing. Returns IDs and a cleanup function.
type authzFixture struct {
	adminID   int
	userID    int
	groupID   int
	connID    int
	db        *sql.DB
}

func setupFixture(t *testing.T, db *sql.DB) authzFixture {
	t.Helper()
	var f authzFixture
	f.db = db

	// Admin user
	err := db.QueryRow(
		`INSERT INTO users (username, password_hash, name, role, is_admin) VALUES ('az_admin', 'h', 'Admin', 'Admin', true) RETURNING id`,
	).Scan(&f.adminID)
	if err != nil {
		t.Fatalf("insert admin: %v", err)
	}

	// Regular user
	err = db.QueryRow(
		`INSERT INTO users (username, password_hash, name, role) VALUES ('az_user', 'h', 'User', 'User') RETURNING id`,
	).Scan(&f.userID)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	// Group
	err = db.QueryRow(
		`INSERT INTO groups (name) VALUES ('az_group') RETURNING id`,
	).Scan(&f.groupID)
	if err != nil {
		t.Fatalf("insert group: %v", err)
	}

	// Database connection
	err = db.QueryRow(
		`INSERT INTO database_connections (database_name, name, dbname, host, port, username, password) VALUES ('az_db', 'az_db', 'az_db', 'localhost', 5432, 'postgres', 'pass') RETURNING id`,
	).Scan(&f.connID)
	if err != nil {
		t.Fatalf("insert connection: %v", err)
	}

	t.Cleanup(func() {
		db.Exec(`DELETE FROM group_database_access WHERE group_id = $1`, f.groupID)
		db.Exec(`DELETE FROM user_groups WHERE group_id = $1`, f.groupID)
		db.Exec(`DELETE FROM database_connections WHERE id = $1`, f.connID)
		db.Exec(`DELETE FROM groups WHERE id = $1`, f.groupID)
		db.Exec(`DELETE FROM users WHERE id IN ($1, $2)`, f.adminID, f.userID)
	})

	return f
}

// ---------------------------------------------------------------------------
// IsAdmin unit test (no DB needed)
// ---------------------------------------------------------------------------

func TestIsAdmin(t *testing.T) {
	if !authz.IsAdmin("Admin") {
		t.Error("IsAdmin('Admin') = false, want true")
	}
	if authz.IsAdmin("User") {
		t.Error("IsAdmin('User') = true, want false")
	}
	if authz.IsAdmin("") {
		t.Error("IsAdmin('') = true, want false")
	}
}

// ---------------------------------------------------------------------------
// GetUserDatabaseAccess integration tests
// ---------------------------------------------------------------------------

func TestGetUserDatabaseAccess_NoAccess(t *testing.T) {
	db := openTestDB(t)
	f := setupFixture(t, db)

	// User has no group membership → should get AccessNone
	level, err := authz.GetUserDatabaseAccess(db, f.userID, f.connID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if level != authz.AccessNone {
		t.Fatalf("expected AccessNone for user with no group, got %q", level)
	}
}

func TestGetUserDatabaseAccess_ViewAccess(t *testing.T) {
	db := openTestDB(t)
	f := setupFixture(t, db)

	// Add user to group, grant view access to the connection
	db.Exec(`INSERT INTO user_groups (user_id, group_id) VALUES ($1, $2)`, f.userID, f.groupID)
	db.Exec(`INSERT INTO group_database_access (group_id, database_id, db_id, access_level) VALUES ($1, $2, $2, 'view')`, f.groupID, f.connID)

	level, err := authz.GetUserDatabaseAccess(db, f.userID, f.connID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if level != authz.AccessView {
		t.Fatalf("expected AccessView, got %q", level)
	}
}

func TestGetUserDatabaseAccess_EditAccess(t *testing.T) {
	db := openTestDB(t)
	f := setupFixture(t, db)

	db.Exec(`INSERT INTO user_groups (user_id, group_id) VALUES ($1, $2)`, f.userID, f.groupID)
	db.Exec(`INSERT INTO group_database_access (group_id, database_id, db_id, access_level) VALUES ($1, $2, $2, 'edit')`, f.groupID, f.connID)

	level, err := authz.GetUserDatabaseAccess(db, f.userID, f.connID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if level != authz.AccessEdit {
		t.Fatalf("expected AccessEdit, got %q", level)
	}
}

func TestGetUserDatabaseAccess_AdminBypass(t *testing.T) {
	db := openTestDB(t)
	f := setupFixture(t, db)

	// Admin has no group membership, yet should get edit access to everything
	level, err := authz.GetUserDatabaseAccess(db, f.adminID, f.connID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if level != authz.AccessEdit {
		t.Fatalf("admin should always get AccessEdit, got %q", level)
	}
}

func TestGetUserDatabaseAccess_MultiGroupHighestWins(t *testing.T) {
	db := openTestDB(t)
	f := setupFixture(t, db)

	// Create a second group that has edit access
	var group2ID int
	err := db.QueryRow(`INSERT INTO groups (name) VALUES ('az_group2') RETURNING id`).Scan(&group2ID)
	if err != nil {
		t.Fatalf("insert group2: %v", err)
	}
	t.Cleanup(func() {
		db.Exec(`DELETE FROM group_database_access WHERE group_id = $1`, group2ID)
		db.Exec(`DELETE FROM user_groups WHERE group_id = $1`, group2ID)
		db.Exec(`DELETE FROM groups WHERE id = $1`, group2ID)
	})

	// User in group1 (view) AND group2 (edit)
	db.Exec(`INSERT INTO user_groups (user_id, group_id) VALUES ($1, $2)`, f.userID, f.groupID)
	db.Exec(`INSERT INTO user_groups (user_id, group_id) VALUES ($1, $2)`, f.userID, group2ID)
	db.Exec(`INSERT INTO group_database_access (group_id, database_id, db_id, access_level) VALUES ($1, $2, $2, 'view')`, f.groupID, f.connID)
	db.Exec(`INSERT INTO group_database_access (group_id, database_id, db_id, access_level) VALUES ($1, $2, $2, 'edit')`, group2ID, f.connID)

	// Documented rule: highest privilege wins → expect edit
	level, err := authz.GetUserDatabaseAccess(db, f.userID, f.connID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if level != authz.AccessEdit {
		t.Fatalf("multi-group: highest privilege should win (edit), got %q", level)
	}
}

// ---------------------------------------------------------------------------
// GetUserDatabaseList integration tests
// ---------------------------------------------------------------------------

func TestGetUserDatabaseList_EmptyForUserWithNoGroups(t *testing.T) {
	db := openTestDB(t)
	f := setupFixture(t, db)

	list, err := authz.GetUserDatabaseList(db, f.userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, d := range list {
		if d.ConnectionID == f.connID {
			t.Fatal("user with no groups should not see any database, but saw one")
		}
	}
}

func TestGetUserDatabaseList_AdminSeesAll(t *testing.T) {
	db := openTestDB(t)
	f := setupFixture(t, db)

	list, err := authz.GetUserDatabaseList(db, f.adminID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var found bool
	for _, d := range list {
		if d.ConnectionID == f.connID {
			found = true
			if d.AccessLevel != authz.AccessEdit {
				t.Fatalf("admin should see edit access, got %q", d.AccessLevel)
			}
		}
	}
	if !found {
		t.Fatal("admin did not see the test connection in their database list")
	}
}

func TestGetUserDatabaseList_UserSeesGrantedDBsOnly(t *testing.T) {
	db := openTestDB(t)
	f := setupFixture(t, db)

	// Create a second connection the user is NOT granted access to
	var conn2ID int
	err := db.QueryRow(
		`INSERT INTO database_connections (database_name, name, dbname, host, port, username, password) VALUES ('az_db2', 'az_db2', 'az_db2', 'localhost', 5432, 'postgres', 'pass') RETURNING id`,
	).Scan(&conn2ID)
	if err != nil {
		t.Fatalf("insert conn2: %v", err)
	}
	t.Cleanup(func() { db.Exec(`DELETE FROM database_connections WHERE id = $1`, conn2ID) })

	// Grant the user view access to only f.connID
	db.Exec(`INSERT INTO user_groups (user_id, group_id) VALUES ($1, $2)`, f.userID, f.groupID)
	db.Exec(`INSERT INTO group_database_access (group_id, database_id, db_id, access_level) VALUES ($1, $2, $2, 'view')`, f.groupID, f.connID)

	list, err := authz.GetUserDatabaseList(db, f.userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var foundGranted, foundUngranted bool
	for _, d := range list {
		if d.ConnectionID == f.connID {
			foundGranted = true
		}
		if d.ConnectionID == conn2ID {
			foundUngranted = true
		}
	}
	if !foundGranted {
		t.Fatal("user did not see the database they have access to")
	}
	if foundUngranted {
		t.Fatal("user saw a database they do not have access to")
	}
}
