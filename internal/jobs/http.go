package jobs

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/observability"
)

type jobResponse struct {
	ID               string     `json:"job_id"`
	DatabaseID       int        `json:"database_id"`
	Kind             string     `json:"kind"`
	Mode             string     `json:"mode"`
	Status           Status     `json:"status"`
	BatchSize        int        `json:"batch_size,omitempty"`
	BatchTable       string     `json:"batch_table,omitempty"`
	BatchColumn      string     `json:"batch_column,omitempty"`
	BatchesCommitted int        `json:"batches_committed"`
	RowsCommitted    int64      `json:"rows_committed"`
	Attempt          int        `json:"attempt"`
	MaxAttempt       int        `json:"max_attempts"`
	Version          *int64     `json:"version,omitempty"`
	Error            string     `json:"error,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	StartedAt        *time.Time `json:"started_at,omitempty"`
	FinishedAt       *time.Time `json:"finished_at,omitempty"`
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
		writeJSON(w, http.StatusOK, toResponse(job))
	})
}

// ResumeHandler serves POST /api/jobs/{id}/resume: it puts a partial batch job
// back in the queue so the worker continues from the last checkpoint.
// Unknown jobs and jobs the user cannot see return 404, users without edit
// access get 403, and a job that is not partial gets 409.
func ResumeHandler(store *Store, canView, canEdit func(r *http.Request, databaseID int) bool) http.Handler {
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
		if !canEdit(r, job.DatabaseID) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "edit access required"})
			return
		}
		resumed, err := store.Resume(r.Context(), job.ID)
		if errors.Is(err, ErrNotResumable) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
			return
		}
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}
		slog.InfoContext(r.Context(), "job.resumed", "job_id", resumed.ID, "database_id", resumed.DatabaseID,
			"batches", resumed.BatchesCommitted)
		writeJSON(w, http.StatusAccepted, toResponse(resumed))
	})
}

func toResponse(job Job) jobResponse {
	resp := jobResponse{
		ID: job.ID, DatabaseID: job.DatabaseID, Kind: job.Kind, Mode: job.Mode, Status: job.Status,
		BatchSize: job.BatchSize, BatchTable: job.BatchTable, BatchColumn: job.BatchColumn,
		BatchesCommitted: job.BatchesCommitted, RowsCommitted: job.RowsCommitted,
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
	return resp
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
