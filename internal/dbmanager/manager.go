package dbmanager

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
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
	ExecutionID        int64
	Version            int64
	MigrationID        sql.NullInt64
	JobID              sql.NullString
	RequestID          sql.NullString
	Revision           int
	SQL                string
	ExecutedSQL        string
	SQLDown            string
	Irreversible       bool
	IrreversibleReason sql.NullString
	ExecutedBy         string
	Action             string
	TargetExecutionID  sql.NullInt64
	Status             string
	Error              sql.NullString
	ExecutedAt         sql.NullTime
}

type MigrationMetadata struct {
	MigrationID sql.NullInt64 // ID of the workflow migration request
	JobID       sql.NullString
	RequestID   string
	Revision    int // worker attempt number
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
