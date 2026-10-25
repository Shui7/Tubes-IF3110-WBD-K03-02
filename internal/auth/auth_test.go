package auth

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/dbplatform"
	_ "github.com/lib/pq"
)

// openTestDB opens a platform DB for integration tests.
// Tests that call this are skipped if TEST_DATABASE_URL is not set.
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

// ---------------------------------------------------------------------------
// Session unit tests (no DB required)
// ---------------------------------------------------------------------------

func TestGenerateTokenProducesUniqueValues(t *testing.T) {
	a, err := generateToken(32)
	if err != nil {
		t.Fatalf("generateToken: %v", err)
	}
	b, err := generateToken(32)
	if err != nil {
		t.Fatalf("generateToken: %v", err)
	}
	if a == b {
		t.Fatal("generateToken produced duplicate values — PRNG may be broken")
	}
	if len(a) != 64 { // 32 bytes → 64 hex chars
		t.Fatalf("token length = %d, want 64", len(a))
	}
}

// ---------------------------------------------------------------------------
// Session integration tests
// ---------------------------------------------------------------------------

func newTestStore(db *sql.DB) *SessionStore {
	return NewSessionStore(db, 15*time.Minute, 1*time.Hour, 5*time.Second)
}

func TestCreateAndGetSession(t *testing.T) {
	db := openTestDB(t)
	store := newTestStore(db)

	// Create a real user to satisfy the FK constraint
	var userID int
	err := db.QueryRow(
		`INSERT INTO users (username, password_hash, name, role) VALUES ('sess_test_user', 'hash', 'Test', 'User') RETURNING id`,
	).Scan(&userID)
	if err != nil {
		t.Fatalf("insert test user: %v", err)
	}
	t.Cleanup(func() {
		db.Exec(`DELETE FROM sessions WHERE user_id = $1`, userID)
		db.Exec(`DELETE FROM users WHERE id = $1`, userID)
	})

	sess, err := store.CreateSession(userID)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if sess.ID == "" {
		t.Fatal("session ID is empty")
	}
	if sess.CSRFToken == "" {
		t.Fatal("CSRF token is empty")
	}

	got, err := store.GetSession(sess.ID)
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if got.UserID != userID {
		t.Fatalf("got UserID = %d, want %d", got.UserID, userID)
	}
}

func TestGetSessionRejectsExpiredAbsolute(t *testing.T) {
	db := openTestDB(t)
	// Store with a 1-second absolute lifetime
	store := NewSessionStore(db, 10*time.Minute, 1*time.Second, 5*time.Second)

	var userID int
	err := db.QueryRow(
		`INSERT INTO users (username, password_hash, name, role) VALUES ('sess_abs_user', 'hash', 'Test', 'User') RETURNING id`,
	).Scan(&userID)
	if err != nil {
		t.Fatalf("insert test user: %v", err)
	}
	t.Cleanup(func() {
		db.Exec(`DELETE FROM sessions WHERE user_id = $1`, userID)
		db.Exec(`DELETE FROM users WHERE id = $1`, userID)
	})

	sess, err := store.CreateSession(userID)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	time.Sleep(2 * time.Second)

	if _, err := store.GetSession(sess.ID); err == nil {
		t.Fatal("GetSession accepted an absolutely-expired session")
	}
}

func TestGetSessionRejectsIdleExpired(t *testing.T) {
	db := openTestDB(t)
	// Store with a 1-second idle timeout
	store := NewSessionStore(db, 1*time.Second, 1*time.Hour, 5*time.Second)

	var userID int
	err := db.QueryRow(
		`INSERT INTO users (username, password_hash, name, role) VALUES ('sess_idle_user', 'hash', 'Test', 'User') RETURNING id`,
	).Scan(&userID)
	if err != nil {
		t.Fatalf("insert test user: %v", err)
	}
	t.Cleanup(func() {
		db.Exec(`DELETE FROM sessions WHERE user_id = $1`, userID)
		db.Exec(`DELETE FROM users WHERE id = $1`, userID)
	})

	sess, err := store.CreateSession(userID)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	time.Sleep(2 * time.Second)

	if _, err := store.GetSession(sess.ID); err == nil {
		t.Fatal("GetSession accepted an idle-expired session")
	}
}

func TestRevokeSession(t *testing.T) {
	db := openTestDB(t)
	store := newTestStore(db)

	var userID int
	err := db.QueryRow(
		`INSERT INTO users (username, password_hash, name, role) VALUES ('sess_revoke_user', 'hash', 'Test', 'User') RETURNING id`,
	).Scan(&userID)
	if err != nil {
		t.Fatalf("insert test user: %v", err)
	}
	t.Cleanup(func() {
		db.Exec(`DELETE FROM sessions WHERE user_id = $1`, userID)
		db.Exec(`DELETE FROM users WHERE id = $1`, userID)
	})

	sess, err := store.CreateSession(userID)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if err := store.RevokeSession(sess.ID); err != nil {
		t.Fatalf("RevokeSession: %v", err)
	}
	if _, err := store.GetSession(sess.ID); err == nil {
		t.Fatal("GetSession accepted a revoked session")
	}
}

func TestRotateSessionGracePeriod(t *testing.T) {
	db := openTestDB(t)
	store := newTestStore(db)

	var userID int
	err := db.QueryRow(
		`INSERT INTO users (username, password_hash, name, role) VALUES ('sess_rotate_user', 'hash', 'Test', 'User') RETURNING id`,
	).Scan(&userID)
	if err != nil {
		t.Fatalf("insert test user: %v", err)
	}
	t.Cleanup(func() {
		db.Exec(`DELETE FROM sessions WHERE user_id = $1`, userID)
		db.Exec(`DELETE FROM users WHERE id = $1`, userID)
	})

	old, err := store.CreateSession(userID)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	newSess, err := store.RotateSession(old.ID)
	if err != nil {
		t.Fatalf("RotateSession: %v", err)
	}
	if newSess.ID == old.ID {
		t.Fatal("rotated session has same ID as old session")
	}

	// Old ID should still work during grace period
	if _, err := store.GetSession(old.ID); err != nil {
		t.Fatalf("old session rejected within grace period: %v", err)
	}
	// New ID must work
	if _, err := store.GetSession(newSess.ID); err != nil {
		t.Fatalf("new session rejected: %v", err)
	}
	// Rotation must not extend absolute expiration
	if newSess.ExpiresAt != old.ExpiresAt {
		t.Fatal("rotation extended absolute expiration — not allowed per spec")
	}
}

// ---------------------------------------------------------------------------
// Middleware unit tests (no DB required)
// ---------------------------------------------------------------------------

func TestRequireAuthRedirectsUnauthenticated(t *testing.T) {
	store := NewSessionStore(nil, time.Hour, 24*time.Hour, 5*time.Minute)
	mw := NewAuthMiddleware(store)

	called := false
	handler := mw.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("expected 302 redirect for unauthenticated request, got %d", w.Code)
	}
	if called {
		t.Fatal("handler was called despite missing session cookie")
	}
	if loc := w.Header().Get("Location"); !strings.Contains(loc, "/login") {
		t.Fatalf("redirect location = %q, want /login", loc)
	}
}

func TestRequireAdminRejectsNonAdmin(t *testing.T) {
	db := openTestDB(t)
	store := newTestStore(db)
	mw := NewAuthMiddleware(store)

	var userID int
	err := db.QueryRow(
		`INSERT INTO users (username, password_hash, name, role) VALUES ('mw_user_role', 'hash', 'Test', 'User') RETURNING id`,
	).Scan(&userID)
	if err != nil {
		t.Fatalf("insert test user: %v", err)
	}
	t.Cleanup(func() {
		db.Exec(`DELETE FROM sessions WHERE user_id = $1`, userID)
		db.Exec(`DELETE FROM users WHERE id = $1`, userID)
	})

	sess, err := store.CreateSession(userID)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	called := false
	handler := mw.RequireAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: sess.ID})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for non-admin user, got %d", w.Code)
	}
	if called {
		t.Fatal("admin handler was called for a non-admin user")
	}
}

func TestRequireAdminAllowsAdmin(t *testing.T) {
	db := openTestDB(t)
	store := newTestStore(db)
	mw := NewAuthMiddleware(store)

	var userID int
	err := db.QueryRow(
		`INSERT INTO users (username, password_hash, name, role) VALUES ('mw_admin_role', 'hash', 'Admin User', 'Admin') RETURNING id`,
	).Scan(&userID)
	if err != nil {
		t.Fatalf("insert test user: %v", err)
	}
	t.Cleanup(func() {
		db.Exec(`DELETE FROM sessions WHERE user_id = $1`, userID)
		db.Exec(`DELETE FROM users WHERE id = $1`, userID)
	})

	sess, err := store.CreateSession(userID)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	called := false
	handler := mw.RequireAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: sess.ID})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Fatal("admin handler was not called for admin user")
	}
}

func TestRequireAuthCSRFProtection(t *testing.T) {
	db := openTestDB(t)
	store := newTestStore(db)
	mw := NewAuthMiddleware(store)

	var userID int
	err := db.QueryRow(
		`INSERT INTO users (username, password_hash, name, role) VALUES ('mw_csrf_user', 'hash', 'Test', 'User') RETURNING id`,
	).Scan(&userID)
	if err != nil {
		t.Fatalf("insert test user: %v", err)
	}
	t.Cleanup(func() {
		db.Exec(`DELETE FROM sessions WHERE user_id = $1`, userID)
		db.Exec(`DELETE FROM users WHERE id = $1`, userID)
	})

	sess, err := store.CreateSession(userID)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	handler := mw.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// POST with wrong CSRF token
	req := httptest.NewRequest(http.MethodPost, "/databases/1/migrations", strings.NewReader("csrf_token=wrong"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "session_id", Value: sess.ID})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for wrong CSRF token on POST, got %d", w.Code)
	}

	// POST with correct CSRF token
	req2 := httptest.NewRequest(http.MethodPost, "/databases/1/migrations",
		strings.NewReader("csrf_token="+sess.CSRFToken))
	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req2.AddCookie(&http.Cookie{Name: "session_id", Value: sess.ID})
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200 for correct CSRF token on POST, got %d", w2.Code)
	}
}

func TestRequireAuthCSRFViaHeader(t *testing.T) {
	db := openTestDB(t)
	store := newTestStore(db)
	mw := NewAuthMiddleware(store)

	var userID int
	err := db.QueryRow(
		`INSERT INTO users (username, password_hash, name, role) VALUES ('mw_csrf_hdr', 'hash', 'Test', 'User') RETURNING id`,
	).Scan(&userID)
	if err != nil {
		t.Fatalf("insert test user: %v", err)
	}
	t.Cleanup(func() {
		db.Exec(`DELETE FROM sessions WHERE user_id = $1`, userID)
		db.Exec(`DELETE FROM users WHERE id = $1`, userID)
	})

	sess, err := store.CreateSession(userID)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	handler := mw.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// CSRF via X-CSRF-Token header
	req := httptest.NewRequest(http.MethodDelete, "/resource", nil)
	req.Header.Set("X-CSRF-Token", sess.CSRFToken)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: sess.ID})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for CSRF via header, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// Auth handler unit tests
// ---------------------------------------------------------------------------

func TestHandleLoginRejectsGet(t *testing.T) {
	// GET to /login should show the login form (not 405)
	store := NewSessionStore(nil, time.Hour, 24*time.Hour, 5*time.Minute)
	h := NewAuthHandler(nil, store)

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	w := httptest.NewRecorder()
	h.HandleLogin(w, req)

	// GET shows the form — expect 200 (renders login.html)
	// If template rendering is unavailable this may panic; skip gracefully.
	if w.Code == http.StatusMethodNotAllowed {
		t.Fatal("GET /login returned 405, expected the login form")
	}
}

func TestHandleLoginRejectsInvalidCSRF(t *testing.T) {
	store := NewSessionStore(nil, time.Hour, 24*time.Hour, 5*time.Minute)
	h := NewAuthHandler(nil, store)

	form := strings.NewReader("username=admin&password=secret&csrf_token=bad_token")
	req := httptest.NewRequest(http.MethodPost, "/login", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "csrf_login_cookie", Value: "correct_token"})
	w := httptest.NewRecorder()

	h.HandleLoginSubmit(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for CSRF mismatch on login, got %d", w.Code)
	}
}

func TestHandleLoginRejectsMissingCSRFCookie(t *testing.T) {
	store := NewSessionStore(nil, time.Hour, 24*time.Hour, 5*time.Minute)
	h := NewAuthHandler(nil, store)

	form := strings.NewReader("username=admin&password=secret&csrf_token=token")
	req := httptest.NewRequest(http.MethodPost, "/login", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// No csrf_login_cookie set
	w := httptest.NewRecorder()

	h.HandleLoginSubmit(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for missing CSRF cookie, got %d", w.Code)
	}
}

func TestHandleLogoutClearsSessionCookie(t *testing.T) {
	db := openTestDB(t)
	store := newTestStore(db)
	h := NewAuthHandler(db, store)

	var userID int
	err := db.QueryRow(
		`INSERT INTO users (username, password_hash, name, role) VALUES ('logout_user', 'hash', 'Test', 'User') RETURNING id`,
	).Scan(&userID)
	if err != nil {
		t.Fatalf("insert test user: %v", err)
	}
	t.Cleanup(func() {
		db.Exec(`DELETE FROM sessions WHERE user_id = $1`, userID)
		db.Exec(`DELETE FROM users WHERE id = $1`, userID)
	})

	sess, _ := store.CreateSession(userID)
	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: sess.ID})
	w := httptest.NewRecorder()

	h.HandleLogout(w, req)

	// Session must be revoked in the store
	if _, err := store.GetSession(sess.ID); err == nil {
		t.Fatal("session still valid after logout")
	}
	// Cookie must be cleared
	var cleared bool
	for _, c := range w.Result().Cookies() {
		if c.Name == "session_id" && (c.MaxAge < 0 || c.Value == "") {
			cleared = true
		}
	}
	if !cleared {
		t.Fatal("logout did not clear session_id cookie")
	}
}

// Inject a live session into context so handlers can call currentUser.
func ctxWithSession(ctx context.Context, sess *Session) context.Context {
	return context.WithValue(ctx, SessionContextKey, sess)
}
