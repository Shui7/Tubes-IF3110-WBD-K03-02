package dbmanager

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Schema struct {
	Name string
}

type Table struct {
	Schema string
	Name   string
	Type   string
}

type Column struct {
	Schema      string
	Table       string
	Name        string
	DataType    string
	IsNullable  bool
	DefaultExpr sql.NullString
	Ordinal     int
}

type TableDetails struct {
	Schema      string
	Name        string
	Columns     []Column
	Constraints []Constraint
	Indexes     []Index
}

type Constraint struct {
	Name       string
	Type       string
	Definition string
}

type Index struct {
	Name       string
	Definition string
}

type Relationship struct {
	FromSchema string
	FromTable  string
	FromColumn string
	ToSchema   string
	ToTable    string
	ToColumn   string
}

type Row map[string]any

type Migration struct {
	Version    int64
	SQL        string
	ExecutedBy string
	Status     string
	Error      sql.NullString
	ExecutedAt sql.NullTime
}

type MigrationDraft struct {
	ID        int64
	Name      string
	SQL       string
	CreatedBy string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Manager struct {
	db *sql.DB
}

// manager type shi
func Open(ctx context.Context, connectionString string) (*Manager, error) {
	if strings.TrimSpace(connectionString) == "" {
		return nil, errors.New("database connection string is required")
	}

	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, fmt.Errorf("open postgres connection: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return New(db), nil
}

func New(db *sql.DB) *Manager {
	return &Manager{db: db}
}

func (m *Manager) Close() error {
	return m.db.Close()
}
