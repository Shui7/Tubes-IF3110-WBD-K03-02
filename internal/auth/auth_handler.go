package auth

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/web"
	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/password"
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
		renderLoginAsGuest(w)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var userID int
	var passwordHash string
	query := "SELECT id, password_hash FROM users WHERE username = $1"
	err := h.db.QueryRow(query, req.Username).Scan(&userID, &passwordHash)
	if err != nil {
		if err == sql.ErrNoRows {
			renderLoginWithError(w, "Invalid username or password")
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if !password.VerifyPasswordHash(req.Password, passwordHash) {
		renderLoginWithError(w, "Invalid username or password")
		return
	}

	sess, err := h.sessionStore.CreateSession(userID)
	if err != nil {
		renderLoginWithError(w, err.Error())
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sess.ID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Expires:  sess.ExpiresAt,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":    "Login successful",
		"csrf_token": sess.CSRFToken,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func renderLoginAsGuest(w http.ResponseWriter) {
	web.RenderTemplate(w, "login.html", web.LoginData{
		BaseData: web.BaseData{
			Name:  "",
			Role:  "Guest",
		},
	})
}

func renderLoginWithError(w http.ResponseWriter, err string) {
	web.RenderTemplate(w, "login.html", web.LoginData{
		BaseData: web.BaseData{
			Name:  "",
			Role:  "Guest",
			Error: err,
		},
	})
}

func redirectLoginSuccess(w http.ResponseWriter, message string) {
	web.RenderTemplate(w, "login.html", web.LoginData{
		BaseData: web.BaseData{
			Name:  "",
			Role:  "Guest",
			Success: message,
		},
	})
}

func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
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
	json.NewEncoder(w).Encode(map[string]string{"message": "Logout successful"})
}