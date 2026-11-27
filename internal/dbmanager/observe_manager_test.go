package dbmanager

import (
	"errors"
	"strings"
	"testing"

	"github.com/lib/pq"
)

func TestSafeStatementRedactsLiteralsAndComments(t *testing.T) {
	query := "INSERT INTO sessions (id, note) VALUES ('opaque-session-value', $1) -- user@example.test"
	got := safeStatement(query)
	for _, secret := range []string{"opaque-session-value", "user@example.test"} {
		if strings.Contains(got, secret) {
			t.Fatalf("safeStatement leaked %q: %s", secret, got)
		}
	}
	if !strings.Contains(got, "VALUES ('?', $1)") || !strings.Contains(got, "-- redacted") {
		t.Fatalf("safeStatement did not preserve query shape: %s", got)
	}
}

func TestParameterTypesDoNotExposeValues(t *testing.T) {
	got := parameterTypes([]any{"secret-value", int64(42), nil})
	want := []string{"$1:string", "$2:int64", "$3:<nil>"}
	if len(got) != len(want) {
		t.Fatalf("parameterTypes() = %#v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("parameterTypes() = %#v, want %#v", got, want)
		}
	}
	if strings.Contains(strings.Join(got, " "), "secret-value") {
		t.Fatalf("parameterTypes exposed a value: %#v", got)
	}
}

func TestDatabaseSQLState(t *testing.T) {
	err := &pq.Error{Code: "23505"}
	if got := databaseSQLState(err); got != "23505" {
		t.Fatalf("databaseSQLState() = %q, want 23505", got)
	}
	if got := databaseSQLState(errors.New("connection failed")); got != "" {
		t.Fatalf("databaseSQLState() = %q, want empty", got)
	}
}
