package admin

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
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

func TestDecodeDeleteID(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/delete", strings.NewReader(`{"id":42}`))
	id, err := decodeDeleteID(req)
	if err != nil {
		t.Fatalf("decodeDeleteID returned error: %v", err)
	}
	if id != 42 {
		t.Fatalf("decoded ID = %d, want 42", id)
	}
}

func TestDeleteHandlersRejectInvalidID(t *testing.T) {
	handlers := []struct {
		name    string
		handler func(*sql.DB, http.ResponseWriter, *http.Request)
	}{
		{name: "connection", handler: HandleDeleteConnection},
		{name: "group", handler: HandleDeleteGroup},
		{name: "permission", handler: HandleDeletePermission},
		{name: "user", handler: HandleDeleteUser},
	}

	for _, test := range handlers {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/delete", strings.NewReader(`{"id":"42"}`))
			w := httptest.NewRecorder()
			test.handler(nil, w, req)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
			}
		})
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
	if err := SetGroupLeader(db, foundGroupID, foundUserID); err != nil {
		t.Fatalf("SetGroupLeader failed for group member: %v", err)
	}
	if err := SetGroupLeader(db, foundGroupID, 1); err != sql.ErrNoRows {
		t.Fatalf("SetGroupLeader error for nonmember = %v, want sql.ErrNoRows", err)
	}
	groups, err = GetGroups(db)
	if err != nil {
		t.Fatalf("GetGroups with members failed: %v", err)
	}
	foundGroup := false
	for _, group := range groups {
		if group.ID == foundGroupID {
			foundGroup = true
			if len(group.Members) != 1 || group.Members[0].ID != foundUserID || !group.LeaderIsMember {
				t.Fatalf("group members/leader = %+v, want test user as the only member and leader", group)
			}
		}
	}
	if !foundGroup {
		t.Fatalf("GetGroups did not return group %d", foundGroupID)
	}

	if err := GrantPermission(db, foundGroupID, foundConnID, "view"); err != nil {
		t.Fatalf("GrantPermission failed: %v", err)
	}
	permissions, err := GetPermissions(db)
	if err != nil {
		t.Fatalf("GetPermissions failed: %v", err)
	}
	var foundPermissionID int
	for _, permission := range permissions {
		if permission.GroupID == foundGroupID && permission.DatabaseID == foundConnID {
			foundPermissionID = permission.ID
			break
		}
	}
	if foundPermissionID == 0 {
		t.Fatalf("Expected permission for group %d and connection %d", foundGroupID, foundConnID)
	}

	var requestID int64
	err = db.QueryRow(`
		INSERT INTO migration_change_requests (database_id, requester_user_id, requester_name, action, status)
		VALUES ($1, $2, $3, 'up', 'pending_review')
		RETURNING id`, foundConnID, foundUserID, userName).Scan(&requestID)
	if err != nil {
		t.Fatalf("create migration request fixture: %v", err)
	}
	conflict := runDeleteHandler(t, HandleDeleteConnection, db, foundConnID, http.StatusConflict)
	if !strings.Contains(conflict.Body.String(), "migration history") {
		t.Fatalf("connection delete conflict = %q, want migration history explanation", conflict.Body.String())
	}
	if _, err := db.Exec("DELETE FROM migration_change_requests WHERE id = $1", requestID); err != nil {
		t.Fatalf("delete migration request fixture: %v", err)
	}

	// Cleanup
	runDeleteHandler(t, HandleDeletePermission, db, foundPermissionID, http.StatusOK)
	runDeleteHandler(t, HandleDeleteUser, db, foundUserID, http.StatusOK)
	runDeleteHandler(t, HandleDeleteGroup, db, foundGroupID, http.StatusOK)
	runDeleteHandler(t, HandleDeleteConnection, db, foundConnID, http.StatusOK)
}

func runDeleteHandler(t *testing.T, handler func(*sql.DB, http.ResponseWriter, *http.Request), db *sql.DB, id, wantStatus int) *httptest.ResponseRecorder {
	t.Helper()
	body := `{"id":` + strconv.Itoa(id) + `}`
	req := httptest.NewRequest(http.MethodPost, "/delete", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler(db, w, req)
	if w.Code != wantStatus {
		t.Fatalf("delete handler status = %d, want %d, body = %q", w.Code, wantStatus, w.Body.String())
	}
	return w
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
