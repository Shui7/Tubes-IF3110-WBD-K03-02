package workflow

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"testing"
)

func TestValidateSQLHash(t *testing.T) {
	const sqlText = "UPDATE items SET value = 1"
	sum := sha256.Sum256([]byte(sqlText))
	validHash := hex.EncodeToString(sum[:])

	if err := validateSQLHash(sqlText, validHash); err != nil {
		t.Fatalf("valid SQL hash rejected: %v", err)
	}
	for _, hash := range []string{"", "not-the-hash"} {
		err := validateSQLHash(sqlText, hash)
		if !errors.Is(err, ErrInvalidExecution) {
			t.Errorf("validateSQLHash(%q) error = %v, want ErrInvalidExecution", hash, err)
		}
	}
}

func TestValidateRequestMatchesExecution(t *testing.T) {
	request := Request{
		ID: 42, DatabaseID: 7, RequesterID: 9, RequesterName: "Request User",
		Action: "up", SQLUp: "CREATE TABLE widgets (id INT)",
		SQLDown: "DROP TABLE widgets", Mode: "plain",
	}
	job := ExecutionJob{
		ID: "job-id", DatabaseID: 7, Kind: "up", Mode: "plain",
		SQL: request.SQLUp, SQLDown: request.SQLDown, RequestedBy: request.RequesterName,
		WorkflowRequestID: sql.NullInt64{Int64: request.ID, Valid: true},
	}
	if err := validateRequestMatchesExecution(request, job); err != nil {
		t.Fatalf("matching request rejected: %v", err)
	}

	job.SQL = "DROP TABLE users"
	if err := validateRequestMatchesExecution(request, job); !errors.Is(err, ErrInvalidExecution) {
		t.Fatalf("mismatched SQL error = %v, want ErrInvalidExecution", err)
	}

	job.SQL = request.SQLUp
	request.ResumeJobID = sql.NullString{String: "another-job", Valid: true}
	if err := validateRequestMatchesExecution(request, job); !errors.Is(err, ErrInvalidExecution) {
		t.Fatalf("mismatched resume job error = %v, want ErrInvalidExecution", err)
	}
}
