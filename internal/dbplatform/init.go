package dbplatform

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

const schemaSQL = `
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    password VARCHAR(255),
    name VARCHAR(100) NOT NULL DEFAULT '',
    role VARCHAR(20) NOT NULL DEFAULT 'User',
    is_admin BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sessions (
    id VARCHAR(128) PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    csrf_token VARCHAR(128) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_accessed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    previous_id VARCHAR(128),
    rotated_at TIMESTAMPTZ,
    is_revoked BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS groups (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_groups (
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    group_id INT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, group_id)
);

CREATE TABLE IF NOT EXISTS database_connections (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100),
    database_name VARCHAR(100) NOT NULL,
    dbname VARCHAR(100),
    host VARCHAR(255) NOT NULL,
    port INT NOT NULL DEFAULT 5432,
    username VARCHAR(100) NOT NULL,
    password TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS group_database_access (
    id SERIAL PRIMARY KEY,
    group_id INT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    database_id INT REFERENCES database_connections(id) ON DELETE CASCADE,
    db_id INT REFERENCES database_connections(id) ON DELETE CASCADE,
    access_level VARCHAR(20) NOT NULL DEFAULT 'view',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
`

var migrations = []string{
	"ALTER TABLE users ADD COLUMN IF NOT EXISTS name VARCHAR(100) NOT NULL DEFAULT ''",
	"ALTER TABLE users ADD COLUMN IF NOT EXISTS role VARCHAR(20) NOT NULL DEFAULT 'User'",
	"ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash VARCHAR(255)",
	"ALTER TABLE users ADD COLUMN IF NOT EXISTS password VARCHAR(255)",
	"ALTER TABLE users ADD COLUMN IF NOT EXISTS is_admin BOOLEAN NOT NULL DEFAULT FALSE",
	`DO $$
	BEGIN
		IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'full_name') THEN
			UPDATE users SET name = full_name WHERE (name = '' OR name IS NULL) AND full_name IS NOT NULL;
		END IF;
	END $$;`,
	"UPDATE users SET password_hash = password WHERE password_hash IS NULL AND password IS NOT NULL",
	"UPDATE users SET password = password_hash WHERE password IS NULL AND password_hash IS NOT NULL",
	"UPDATE users SET role = 'Admin', is_admin = TRUE WHERE is_admin = TRUE",
	"UPDATE users SET name = 'System Administrator' WHERE username = 'admin' AND (name = '' OR name IS NULL)",

	"ALTER TABLE database_connections ADD COLUMN IF NOT EXISTS database_name VARCHAR(100)",
	"ALTER TABLE database_connections ADD COLUMN IF NOT EXISTS name VARCHAR(100)",
	"ALTER TABLE database_connections ADD COLUMN IF NOT EXISTS dbname VARCHAR(100)",
	`DO $$
	BEGIN
		IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'database_connections' AND column_name = 'name') THEN
			ALTER TABLE database_connections ALTER COLUMN name DROP NOT NULL;
		END IF;
		IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'database_connections' AND column_name = 'dbname') THEN
			ALTER TABLE database_connections ALTER COLUMN dbname DROP NOT NULL;
		END IF;
	END $$;`,
	"UPDATE database_connections SET database_name = COALESCE(NULLIF(database_name, ''), NULLIF(name, ''), dbname, '') WHERE database_name IS NULL OR database_name = ''",
	"UPDATE database_connections SET name = database_name WHERE name IS NULL OR name = '';",
	"UPDATE database_connections SET dbname = database_name WHERE dbname IS NULL OR dbname = '';",

	"ALTER TABLE group_database_access ADD COLUMN IF NOT EXISTS id SERIAL",
	"ALTER TABLE group_database_access ADD COLUMN IF NOT EXISTS database_id INT",
	"ALTER TABLE group_database_access ADD COLUMN IF NOT EXISTS db_id INT",
	"ALTER TABLE group_database_access ADD COLUMN IF NOT EXISTS access_level VARCHAR(20)",
	`DO $$
	BEGIN
		IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'group_database_access' AND column_name = 'access') THEN
			ALTER TABLE group_database_access ALTER COLUMN access DROP NOT NULL;
			UPDATE group_database_access SET access_level = access::text WHERE access_level IS NULL AND access IS NOT NULL;
		END IF;
	END $$;`,
	"UPDATE group_database_access SET database_id = db_id WHERE database_id IS NULL AND db_id IS NOT NULL",
	"UPDATE group_database_access SET db_id = database_id WHERE db_id IS NULL AND database_id IS NOT NULL",
	"UPDATE group_database_access SET access_level = 'view' WHERE access_level IS NULL",

	"ALTER TABLE sessions ADD COLUMN IF NOT EXISTS csrf_token VARCHAR(128) NOT NULL DEFAULT ''",
	"ALTER TABLE sessions ADD COLUMN IF NOT EXISTS last_accessed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()",
	"ALTER TABLE sessions ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ",
	"ALTER TABLE sessions ADD COLUMN IF NOT EXISTS is_revoked BOOLEAN NOT NULL DEFAULT FALSE",
}

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

	for _, stmt := range migrations {
		if _, err := db.Exec(stmt); err != nil {
			log.Printf("schema migration notice: %v", err)
		}
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
		"INSERT INTO users (username, password_hash, password, name, role, is_admin) VALUES ($1, $2, $2, $3, $4, $5)",
		"admin", string(hashedPassword), "System Administrator", "Admin", true,
	)
	return err
}
