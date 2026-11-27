package jobs

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/auth"
	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/authz"
	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/observability"
	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/workflow"
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

// ResumeHandler serves POST /api/jobs/{id}/resume by submitting a new approval
// request. The job remains partial until every required group leader approves.
// Unknown/hidden jobs return 404, users without edit access get 403, and
// non-partial or non-batch jobs get 409.
func ResumeHandler(store *Store, platformDB *sql.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if platformDB == nil {
			slog.ErrorContext(r.Context(), "job.resume_approval_unavailable")
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}
		session, ok := r.Context().Value(auth.SessionContextKey).(*auth.Session)
		if !ok || session == nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
			return
		}

		job, err := store.Get(r.Context(), r.PathValue("id"))
		if errors.Is(err, ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "job not found"})
			return
		}
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}

		level, err := authz.GetUserDatabaseAccess(r.Context(), platformDB, session.UserID, job.DatabaseID)
		if err != nil {
			slog.ErrorContext(r.Context(), "job.resume_access_check_failed", "job_id", job.ID, "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}
		if level == authz.AccessNone {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "job not found"})
			return
		}
		if level != authz.AccessEdit {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "edit access required"})
			return
		}
		if job.Status != StatusPartial || job.Mode != ModeBatch || job.Kind != "up" {
			writeJSON(w, http.StatusConflict, map[string]string{"error": ErrNotResumable.Error()})
			return
		}
		request, err := workflow.CreateResumeRequest(platformDB, job.ID, session.UserID, session.Name)
		if err != nil {
			slog.WarnContext(r.Context(), "job.resume_approval_request_failed", "job_id", job.ID, "error", err)
			if errors.Is(err, workflow.ErrResumeRequestPending) || errors.Is(err, workflow.ErrResumeNotEligible) {
				writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not submit continuation for approval"})
			return
		}
		slog.InfoContext(r.Context(), "job.resume_approval_requested", "job_id", job.ID,
			"database_id", job.DatabaseID, "request_id", request.ID)
		writeJSON(w, http.StatusAccepted, map[string]any{
			"job_id": job.ID, "database_id": job.DatabaseID,
			"request_id": request.ID, "status": "pending_review",
		})
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
