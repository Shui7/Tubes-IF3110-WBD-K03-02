package auth

import (
	"context"
	"net/http"
)

type contextKey string

const SessionContextKey contextKey = "session"

type AuthMiddleware struct {
	sessionStore *SessionStore
}

func NewAuthMiddleware(sessionStore *SessionStore) *AuthMiddleware {
	return &AuthMiddleware{
		sessionStore: sessionStore,
	}
}

// RequireAuth adalah middleware untuk memvalidasi sesi dan token CSRF
func (m *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Ambil cookie session_id dari request
		cookie, err := r.Cookie("session_id")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusFound);
			return
		}

		// 2. Validasi sesi melalui SessionStore (mengecek idle timeout, absolute expiration, & grace period)[cite: 1]
		session, err := m.sessionStore.GetSession(cookie.Value)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusFound);
			return
		}

		// 3. Perbarui last_accessed_at (mencegah idle timeout jika pengguna aktif)[cite: 1]
		_ = m.sessionStore.TouchSession(session.ID)

		// 4. Proteksi CSRF untuk request yang mengubah state (POST, PUT, DELETE, PATCH)[cite: 1]
		if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodDelete || r.Method == http.MethodPatch {
			clientCSRF := r.Header.Get("X-CSRF-Token")
			if clientCSRF == "" {
				clientCSRF = r.FormValue("csrf_token")
			}
			if clientCSRF == "" || clientCSRF != session.CSRFToken {
				http.Error(w, "Forbidden: invalid or missing CSRF token", http.StatusForbidden)
				return
			}
		}

		// 5. Masukkan objek sesi ke dalam context request agar dapat digunakan oleh handler selanjutnya
		ctx := context.WithValue(r.Context(), SessionContextKey, session)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
