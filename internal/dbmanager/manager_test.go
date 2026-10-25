package dbmanager

import (
	"context"
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
