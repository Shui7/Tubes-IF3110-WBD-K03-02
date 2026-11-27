package dbmanager

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

func TestValidateIdentifier(t *testing.T) {
	tests := []struct {
		name       string
		identifier string
		wantError  bool
	}{
		{name: "simple", identifier: "public"},
		{name: "underscore", identifier: "user_profiles"},
		{name: "starts with number", identifier: "1table", wantError: true},
		{name: "injection", identifier: "users; DROP TABLE users", wantError: true},
		{name: "empty", identifier: "", wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateIdentifier(tt.identifier)
			if (err != nil) != tt.wantError {
				t.Fatalf("validateIdentifier(%q) error = %v, wantError %v", tt.identifier, err, tt.wantError)
			}
		})
	}
}

func TestQuoteIdentifier(t *testing.T) {
	if got, want := quoteIdentifier(`table"name`), `"table""name"`; got != want {
		t.Fatalf("quoteIdentifier() = %q, want %q", got, want)
	}
}

func TestHiddenBySuccessfulRollbackUsesExactExecutionIDs(t *testing.T) {
	migrations := []Migration{
		{ExecutionID: 2, Version: 1, Action: "up", Status: "success"},
		{ExecutionID: 3, Version: 1, Action: "rollback", Status: "failed", TargetExecutionID: sql.NullInt64{Int64: 2, Valid: true}},
		{ExecutionID: 4, Version: 1, Action: "rollback", Status: "success", TargetExecutionID: sql.NullInt64{Int64: 2, Valid: true}},
		{ExecutionID: 5, Version: 1, Action: "forward", Status: "success", TargetExecutionID: sql.NullInt64{Int64: 4, Valid: true}},
		{ExecutionID: 6, Version: 2, Action: "rollback", Status: "success"},
	}

	got := HiddenBySuccessfulRollback(migrations)
	want := []int64{2, 4, 6}
	if len(got) != len(want) {
		t.Fatalf("hidden execution IDs = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("hidden execution IDs = %v, want %v", got, want)
		}
	}
}

func TestOpenRejectsEmptyConnectionString(t *testing.T) {
	if _, err := Open(context.Background(), " "); err == nil {
		t.Fatal("Open accepted an empty connection string")
	}
}

func TestOpenReportsConnectionFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	if _, err := Open(ctx, "host=127.0.0.1 port=1 dbname=missing user=missing connect_timeout=1"); err == nil {
		t.Fatal("Open unexpectedly connected to an unavailable database")
	}
}
