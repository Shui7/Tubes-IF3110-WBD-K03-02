package jobs

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/observability"
)

type jobResponse struct {
	ID         string     `json:"job_id"`
	DatabaseID int        `json:"database_id"`
	Kind       string     `json:"kind"`
	Status     Status     `json:"status"`
	Attempt    int        `json:"attempt"`
	MaxAttempt int        `json:"max_attempts"`
	Version    *int64     `json:"version,omitempty"`
	Error      string     `json:"error,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
}

// StatusHandler serves GET /api/jobs/{id}. canView decides whether the current
// user may see jobs of a database (view access or higher).
// Unknown and forbidden jobs both return 404.
func StatusHandler(store *Store, canView func(r *http.Request, databaseID int) bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		job, err := store.Get(r.Context(), r.PathValue("id"))
		if errors.Is(err, ErrNotFound) || (err == nil && !canView(r, job.DatabaseID)) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "job not found"})
			return
		}
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}
		resp := jobResponse{
			ID: job.ID, DatabaseID: job.DatabaseID, Kind: job.Kind, Status: job.Status,
			Attempt: job.Attempt, MaxAttempt: job.MaxAttempts, CreatedAt: job.CreatedAt,
		}
		if job.Version.Valid {
			resp.Version = &job.Version.Int64
		}
		if job.LastError.Valid {
			resp.Error = observability.Redact(job.LastError.String)
		}
		if job.StartedAt.Valid {
			resp.StartedAt = &job.StartedAt.Time
		}
		if job.FinishedAt.Valid {
			resp.FinishedAt = &job.FinishedAt.Time
		}
		writeJSON(w, http.StatusOK, resp)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
