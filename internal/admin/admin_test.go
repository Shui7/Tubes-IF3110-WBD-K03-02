package admin

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

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

func TestHandleAddConnectionValidation(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/admin/connections/add", nil)
	w := httptest.NewRecorder()

	HandleAddConnection(nil, w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected status 303 for GET request, got %d", w.Code)
	}
}

func TestHandleAddUserValidation(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/admin/users/add", nil)
	w := httptest.NewRecorder()

	HandleAddUser(nil, w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected status 303 for GET request, got %d", w.Code)
	}
}

func TestRejectPlatformDBConnection(t *testing.T) {
	if err := AddConnection(nil, "dbplatform", "localhost", 5432, "postgres", "postgres"); err == nil {
		t.Fatal("Expected AddConnection to reject 'dbplatform', got nil error")
	}

	form := url.Values{}
	form.Set("database", "dbplatform")
	form.Set("host", "localhost")
	form.Set("user", "postgres")
	req := httptest.NewRequest(http.MethodPost, "/admin/connections/add", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	HandleAddConnection(nil, w, req)

	if !strings.Contains(w.Header().Get("Location"), "error=") {
		t.Fatalf("Expected redirect location to contain error, got %s", w.Header().Get("Location"))
	}
}

func TestAdminFlowIntegration(t *testing.T) {
	db := openTestDB(t)

	// Test Connections
	testDBName := "test_flow_db"
	if err := AddConnection(db, testDBName, "localhost", 5432, "postgres", "postgres"); err != nil {
		t.Fatalf("AddConnection failed: %v", err)
	}
	conns, err := GetConnections(db)
	if err != nil {
		t.Fatalf("GetConnections failed: %v", err)
	}
	var foundConnID int
	for _, c := range conns {
		if c.Database == testDBName {
			foundConnID = c.ID
			break
		}
	}
	if foundConnID == 0 {
		t.Fatalf("Expected connection %s to be listed", testDBName)
	}
	if err := DeleteConnection(db, foundConnID); err != nil {
		t.Fatalf("DeleteConnection failed: %v", err)
	}

	// Test Groups
	groupName := "test_flow_group"
	if err := AddGroup(db, groupName); err != nil {
		t.Fatalf("AddGroup failed: %v", err)
	}
	groups, err := GetGroups(db)
	if err != nil {
		t.Fatalf("GetGroups failed: %v", err)
	}
	var foundGroupID int
	for _, g := range groups {
		if g.Name == groupName {
			foundGroupID = g.ID
			break
		}
	}
	if foundGroupID == 0 {
		t.Fatalf("Expected group %s to be listed", groupName)
	}

	// Test Users
	userName := "test_flow_user"
	if err := AddUser(db, userName, "Test User", "secret123", "User", []string{groupName}); err != nil {
		t.Fatalf("AddUser failed: %v", err)
	}
	users, err := GetUsers(db)
	if err != nil {
		t.Fatalf("GetUsers failed: %v", err)
	}
	var foundUserID int
	for _, u := range users {
		if u.Username == userName {
			foundUserID = u.ID
			break
		}
	}
	if foundUserID == 0 {
		t.Fatalf("Expected user %s to be listed", userName)
	}

	// Cleanup
	_ = DeleteUser(db, foundUserID)
	_ = DeleteGroup(db, foundGroupID)
}

func TestHandleAddUserForm(t *testing.T) {
	form := url.Values{}
	form.Set("username", "")
	req := httptest.NewRequest(http.MethodPost, "/admin/users/add", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	HandleAddUser(nil, w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect, got %d", w.Code)
	}
}
