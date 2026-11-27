package jobs

import (
	"database/sql/driver"
	"errors"
	"net"
	"testing"

	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/dbmanager"
	"github.com/lib/pq"
)

type testNetError struct{}

func (testNetError) Error() string   { return "network timeout" }
func (testNetError) Timeout() bool   { return true }
func (testNetError) Temporary() bool { return true }

func TestIsTransient(t *testing.T) {
	var networkErr net.Error = testNetError{}
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"bad connection", driver.ErrBadConn, true},
		{"wrapped bad connection", errors.Join(errors.New("query failed"), driver.ErrBadConn), true},
		{"network timeout", testNetError{}, true},
		{"wrapped network timeout", errors.Join(errors.New("query failed"), networkErr), true},
		{"connection SQLSTATE", &pq.Error{Code: "08006"}, true},
		{"serialization failure", &pq.Error{Code: "40001"}, true},
		{"deadlock", &pq.Error{Code: "40P01"}, true},
		{"too many connections", &pq.Error{Code: "53300"}, true},
		{"syntax error", &pq.Error{Code: "42601"}, false},
		{"permission error", &pq.Error{Code: "42501"}, false},
		{"plain error", errors.New("failed"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsTransient(tt.err); got != tt.want {
				t.Fatalf("IsTransient(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestIsCommitUncertain(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"not commit error", errors.New("failure"), false},
		{"connection loss", &dbmanager.CommitError{Version: 1, Err: errors.New("connection lost")}, true},
		{"server rejection", &dbmanager.CommitError{Version: 1, Err: &pq.Error{Code: "40001"}}, false},
		{"nil commit cause", &dbmanager.CommitError{Version: 1}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsCommitUncertain(tt.err); got != tt.want {
				t.Fatalf("IsCommitUncertain(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
