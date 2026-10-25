package auth

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"
)

const sessionDuration = 24 * time.Hour

type Session struct {
	ID             string
	UserID         int
	Name           string
	Role           string
	CSRFToken      string
	CreatedAt      time.Time
	LastAccessedAt time.Time
	ExpiresAt      time.Time
	PreviousID     sql.NullString
	RotatedAt      sql.NullTime
	IsRevoked      bool
}

type SessionStore struct {
	db           *sql.DB
	idleTimeout  time.Duration
	absoluteLife time.Duration
	gracePeriod  time.Duration
}

func NewSessionStore(db *sql.DB, idleTimeout, absoluteLife, gracePeriod time.Duration) *SessionStore {
	return &SessionStore{
		db:           db,
		idleTimeout:  idleTimeout,
		absoluteLife: absoluteLife,
		gracePeriod:  gracePeriod,
	}
}

func generateToken(length int) (string, error) {
	token := make([]byte, length)
	if _, err := rand.Read(token); err != nil {
		return "", err
	}
	return hex.EncodeToString(token), nil
}

func (s *SessionStore) CreateSession(userID int) (*Session, error) {
	sessionID, err := generateToken(32) // Menghasilkan ID acak 64 karakter hex
	if err != nil {
		return nil, err
	}
	csrfToken, err := generateToken(32)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	expiresAt := now.Add(s.absoluteLife) // Absolute expiration

	query := `
		INSERT INTO sessions (id, user_id, csrf_token, created_at, last_accessed_at, expires_at, is_revoked) 
		VALUES ($1, $2, $3, $4, $4, $5, false)
	`
	_, err = s.db.Exec(query, sessionID, userID, csrfToken, now, expiresAt)
	if err != nil {
		return nil, err
	}

	return &Session{
		ID:             sessionID,
		UserID:         userID,
		CSRFToken:      csrfToken,
		CreatedAt:      now,
		LastAccessedAt: now,
		ExpiresAt:      expiresAt,
		IsRevoked:      false,
	}, nil
}

func (s *SessionStore) GetSession(sessionID string) (*Session, error) {
	now := time.Now()

	var session Session
	query := `
		SELECT s.id, s.user_id, u.name, u.role, s.csrf_token, s.created_at, s.last_accessed_at, s.expires_at, s.previous_id, s.rotated_at, s.is_revoked
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.id = $1 AND s.is_revoked = false
	`
	err := s.db.QueryRow(query, sessionID).Scan(
		&session.ID, &session.UserID, &session.Name, &session.Role, &session.CSRFToken, &session.CreatedAt,
		&session.LastAccessedAt, &session.ExpiresAt, &session.PreviousID, &session.RotatedAt, &session.IsRevoked,
	)

	if err == nil {
		if session.RotatedAt.Valid {
			if now.After(session.RotatedAt.Time.Add(s.gracePeriod)) {
				return nil, errors.New("session rotated and grace period expired")
			}
		}

		if now.After(session.ExpiresAt) {
			return nil, errors.New("session expired (absolute)")
		}

		if now.After(session.LastAccessedAt.Add(s.idleTimeout)) {
			return nil, errors.New("session expired (idle)")
		}

		return &session, nil
	}

	var oldRotatedSql sql.NullTime
	err = s.db.QueryRow(`SELECT rotated_at FROM sessions WHERE id = $1`, sessionID).Scan(&oldRotatedSql)
	if err == nil && oldRotatedSql.Valid {
		if now.After(oldRotatedSql.Time.Add(s.gracePeriod)) {
			return nil, errors.New("grace period expired for old session")
		}

		queryPrev := `
			SELECT s.id, s.user_id, u.name, u.role, s.csrf_token, s.created_at, s.last_accessed_at, s.expires_at, s.previous_id, s.rotated_at, s.is_revoked
			FROM sessions s
			JOIN users u ON u.id = s.user_id
			WHERE s.previous_id = $1 AND s.is_revoked = false
		`
		var newSess Session
		err = s.db.QueryRow(queryPrev, sessionID).Scan(
			&newSess.ID, &newSess.UserID, &newSess.Name, &newSess.Role, &newSess.CSRFToken, &newSess.CreatedAt,
			&newSess.LastAccessedAt, &newSess.ExpiresAt, &newSess.PreviousID, &newSess.RotatedAt, &newSess.IsRevoked,
		)
		if err == nil {
			if now.After(newSess.ExpiresAt) || now.After(newSess.LastAccessedAt.Add(s.idleTimeout)) {
				return nil, errors.New("session expired")
			}
			return &newSess, nil
		}
	}

	return nil, errors.New("invalid session")
}

func (s *SessionStore) TouchSession(sessionID string) error {
	now := time.Now()
	query := `UPDATE sessions SET last_accessed_at = $1 WHERE id = $2 AND is_revoked = false`
	_, err := s.db.Exec(query, now, sessionID)
	return err
}

func (s *SessionStore) RotateSession(oldSessionID string) (*Session, error) {
	now := time.Now()

	oldSess, err := s.GetSession(oldSessionID)
	if err != nil {
		return nil, err
	}

	newID, err := generateToken(32)
	if err != nil {
		return nil, err
	}
	newCSRF, err := generateToken(32)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	insertQuery := `
		INSERT INTO sessions (id, user_id, csrf_token, created_at, last_accessed_at, expires_at, previous_id, is_revoked)
		VALUES ($1, $2, $3, $4, $5, $6, $7, false)
	`
	_, err = tx.Exec(insertQuery, newID, oldSess.UserID, newCSRF, oldSess.CreatedAt, now, oldSess.ExpiresAt, oldSessionID)
	if err != nil {
		return nil, err
	}

	updateQuery := `UPDATE sessions SET rotated_at = $1 WHERE id = $2`
	_, err = tx.Exec(updateQuery, now, oldSessionID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &Session{
		ID:             newID,
		UserID:         oldSess.UserID,
		CSRFToken:      newCSRF,
		CreatedAt:      oldSess.CreatedAt,
		LastAccessedAt: now,
		ExpiresAt:      oldSess.ExpiresAt,
		PreviousID:     sql.NullString{String: oldSessionID, Valid: true},
		RotatedAt:      sql.NullTime{Valid: false},
		IsRevoked:      false,
	}, nil
}

func (s *SessionStore) RevokeSession(sessionID string) error {
	query := `UPDATE sessions SET is_revoked = true WHERE id = $1 OR previous_id = $1`
	_, err := s.db.Exec(query, sessionID)
	return err
}
