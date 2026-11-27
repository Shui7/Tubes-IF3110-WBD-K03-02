package auth

import (
	"database/sql"
	"log/slog"
	"net/http"
	"time"

	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/password"
	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/web"
)

type AuthHandler struct {
	db           *sql.DB
	sessionStore *SessionStore
}

func NewAuthHandler(db *sql.DB, sessionStore *SessionStore) *AuthHandler {
	return &AuthHandler{
		db:           db,
		sessionStore: sessionStore,
	}
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		loginAsGuest(w, r)
		return
	}

	h.HandleLoginSubmit(w, r)
}

func (h *AuthHandler) HandleLoginSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Cek CSRF
	formToken := r.FormValue("csrf_token")
	cookie, err := r.Cookie("csrf_login_cookie")
	if err != nil || cookie.Value == "" {
		slog.WarnContext(r.Context(), "auth.login.failed", "reason", "missing_csrf_cookie")
		http.Error(w, "Forbidden: missing CSRF cookie", http.StatusForbidden)
		return
	}

	if formToken == "" || formToken != cookie.Value {
		slog.WarnContext(r.Context(), "auth.login.failed", "reason", "invalid_csrf_token")
		http.Error(w, "Forbidden: invalid CSRF token", http.StatusForbidden)
		return
	}

	var req LoginRequest = LoginRequest{
		Username: r.FormValue("username"),
		Password: r.FormValue("password"),
	}

	// Ambil user dari database
	var userID int
	var passwordHash string
	query := "SELECT id, password_hash FROM users WHERE username = $1"
	err = h.db.QueryRowContext(r.Context(), query, req.Username).Scan(&userID, &passwordHash)
	if err != nil {
		if err == sql.ErrNoRows {
			slog.WarnContext(r.Context(), "auth.login.failed", "reason", "invalid_credentials")
			renderLoginWithError(w, formToken, "Invalid username or password")
			return
		}
		slog.ErrorContext(r.Context(), "auth.login.failed", "reason", "user_lookup", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Verifikasi password
	if !password.VerifyPasswordHash(req.Password, passwordHash) {
		slog.WarnContext(r.Context(), "auth.login.failed", "reason", "invalid_credentials")
		renderLoginWithError(w, formToken, "Invalid username or password")
		return
	}

	// Pembuatan session
	sess, err := h.sessionStore.CreateSession(userID)
	if err != nil {
		slog.ErrorContext(r.Context(), "auth.login.failed", "reason", "session_creation", "user_id", userID, "error", err)
		renderLoginWithError(w, formToken, err.Error())
		return
	}
	slog.InfoContext(r.Context(), "auth.login.succeeded", "user_id", userID)

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sess.ID,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Expires:  sess.ExpiresAt,
	})

	http.SetCookie(w, &http.Cookie{
		Name:   "csrf_login_cookie",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func loginAsGuest(w http.ResponseWriter, r *http.Request) {
	token := ""

	cookie, err := r.Cookie("csrf_login_cookie")
	if err == nil && cookie.Value != "" {
		token = cookie.Value
	} else {
		token, err = generateToken(32)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "csrf_login_cookie",
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteStrictMode,
		})
	}

	w.Header().Set("Cache-Control", "no-store")
	web.RenderTemplate(w, "login.html", web.LoginData{
		BaseData: web.BaseData{
			Name: "Guest",
			Role: "Guest",
			CSRFToken: token,
		},
	})
}

func renderLoginWithError(w http.ResponseWriter, csrf_token string, err string) {
	web.RenderTemplate(w, "login.html", web.LoginData{
		BaseData: web.BaseData{
			Name:  "",
			Role:  "Guest",
			Error: err,
			CSRFToken: csrf_token,
		},
	})
}

func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	if err == nil && cookie != nil {
		if err := h.sessionStore.RevokeSession(cookie.Value); err != nil {
			slog.ErrorContext(r.Context(), "auth.logout.failed", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}
	slog.InfoContext(r.Context(), "auth.logout.succeeded")

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	})

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
