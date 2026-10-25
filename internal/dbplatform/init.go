package dbplatform

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

const schemaSQL = `
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(100) NOT NULL,
    role VARCHAR(20) NOT NULL DEFAULT 'User',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sessions (
    id VARCHAR(128) PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    csrf_token VARCHAR(128) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_accessed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    previous_id VARCHAR(128),
    rotated_at TIMESTAMPTZ,
    is_revoked BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS groups (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_groups (
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    group_id INT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, group_id)
);

CREATE TABLE IF NOT EXISTS database_connections (
    id SERIAL PRIMARY KEY,
    database_name VARCHAR(100) NOT NULL,
    host VARCHAR(255) NOT NULL,
    port INT NOT NULL DEFAULT 5432,
    username VARCHAR(100) NOT NULL,
    password TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS group_database_access (
    id SERIAL PRIMARY KEY,
    group_id INT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    database_id INT NOT NULL REFERENCES database_connections(id) ON DELETE CASCADE,
    access_level VARCHAR(20) NOT NULL CHECK (access_level IN ('view', 'edit')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (group_id, database_id)
);
`

func InitPlatformDB(connStr string) (*sql.DB, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if _, err := db.Exec(schemaSQL); err != nil {
		return nil, fmt.Errorf("failed to execute schema: %w", err)
	}

	if err := seedDefaultAdmin(db); err != nil {
		return nil, fmt.Errorf("failed to seed admin user: %w", err)
	}

	return db, nil
}

func seedDefaultAdmin(db *sql.DB) error {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users WHERE username = 'admin'").Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		return nil
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	_, err = db.Exec(
		"INSERT INTO users (username, password_hash, name, role) VALUES ($1, $2, $3, $4)",
		"admin", string(hashedPassword), "System Administrator", "Admin",
	)
	return err
}
