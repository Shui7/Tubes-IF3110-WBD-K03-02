package auth

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/password"
	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/web"
)

type AuthHandlerAPI struct {
	db           *sql.DB
	sessionStore *SessionStore
}

type apiStringResponse struct {
	Message string `json:"message"`
}

type apiValidateSessionResponse struct {
	Valid bool `json:"valid"`
}

type apiNameRoleResponse struct {
	Name string `json:"name"`
	Role string `json:"role"`
}

func NewAuthHandlerAPI(db *sql.DB, sessionStore *SessionStore) *AuthHandlerAPI {
	return &AuthHandlerAPI{
		db:           db,
		sessionStore: sessionStore,
	}
}

// "api/auth/login"
func (h *AuthHandlerAPI) HandleLoginSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Cek CSRF
	formToken := r.FormValue("csrf_token")
	cookie, err := r.Cookie("csrf_login_cookie")
	if err != nil || cookie.Value == "" {
		http.Error(w, "Forbidden: missing CSRF cookie", http.StatusForbidden)
		return
	}

	if formToken == "" || formToken != cookie.Value {
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
	err = h.db.QueryRow(query, req.Username).Scan(&userID, &passwordHash)
	if err != nil {
		if err == sql.ErrNoRows {
			renderLoginWithError(w, formToken, "Invalid username or password")
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Verifikasi password
	if !password.VerifyPasswordHash(req.Password, passwordHash) {
		renderLoginWithError(w, formToken, "Invalid username or password")
		return
	}

	// Pembuatan session
	sess, err := h.sessionStore.CreateSession(userID)
	if err != nil {
		renderLoginWithError(w, formToken, err.Error())
		return
	}

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

func (h *AuthHandlerAPI) RefreshLoginCsrfToken(w http.ResponseWriter, r *http.Request) {
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
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(web.LoginData{
		BaseData: web.BaseData{
			Name: "Guest",
			Role: "Guest",
			CSRFToken: token,
		},
	})
}

func (h *AuthHandlerAPI) renderLoginWithError(w http.ResponseWriter, csrf_token string, err string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(web.LoginData{
		BaseData: web.BaseData{
			Name:  "",
			Role:  "Guest",
			Error: err,
			CSRFToken: csrf_token,
		},
	})
}

func (h *AuthHandlerAPI) GetNameAndRole(w http.ResponseWriter, r *http.Request) {	
	name := "Guest"
	role := "Guest"
	cookie, err := r.Cookie("session_id")
	if err == nil && cookie != nil {
		session, err := h.sessionStore.GetSession(cookie.Value)
		if err == nil {
			_ = h.sessionStore.TouchSession(session.ID)
			name = session.Name
			role = session.Role
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(apiNameRoleResponse{
		Name: name,
		Role: role,
	})
}

func (h *AuthHandlerAPI) ValidateSession(w http.ResponseWriter, r *http.Request) {
	sessionValid := false
	
	cookie, err := r.Cookie("session_id")
	if err == nil && cookie != nil {
		session, err := h.sessionStore.GetSession(cookie.Value)
		if err == nil {
			_ = h.sessionStore.TouchSession(session.ID)
			sessionValid = true
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(apiValidateSessionResponse{
		Valid: sessionValid,
	})
}

func (h *AuthHandlerAPI) HandleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	if err == nil && cookie != nil {
		_ = h.sessionStore.RevokeSession(cookie.Value)
	}

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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(apiStringResponse{
		Message: "Logout successful",
	})
}
