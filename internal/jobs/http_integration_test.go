package jobs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/auth"
)

func TestJobHTTPHandlersAuthorizationAndStatus(t *testing.T) {
	db := openJobsIntegrationDB(t)
	ctx := context.Background()
	databaseID := createJobsTestDatabase(t, db)
	editUserID, editName := createJobsTestUser(t, db, "User")
	viewUserID, viewName := createJobsTestUser(t, db, "User")
	hiddenUserID, _ := createJobsTestUser(t, db, "User")
	grantJobsTestAccess(t, db, databaseID, editUserID, "edit")
	grantJobsTestAccess(t, db, databaseID, viewUserID, "view")
	store := NewStore(db)
	job := enqueueJobsTestJob(t, ctx, store, databaseID, EnqueueRequest{SQL: "SELECT 1"})

	statusHandler := StatusHandler(store, func(r *http.Request, id int) bool {
		session, ok := r.Context().Value(auth.SessionContextKey).(*auth.Session)
		return ok && session != nil && (session.UserID == editUserID || session.UserID == viewUserID) && id == databaseID
	})
	statusRequest := httptest.NewRequest(http.MethodGet, "/api/jobs/"+job.ID, nil)
	statusRequest.SetPathValue("id", job.ID)
	statusRequest = statusRequest.WithContext(context.WithValue(statusRequest.Context(), auth.SessionContextKey,
		&auth.Session{UserID: editUserID, Name: editName}))
	statusResponse := httptest.NewRecorder()
	statusHandler.ServeHTTP(statusResponse, statusRequest)
	if statusResponse.Code != http.StatusOK {
		t.Fatalf("authorized status response = %d, want 200: %s", statusResponse.Code, statusResponse.Body.String())
	}
	var response jobResponse
	if err := json.Unmarshal(statusResponse.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.ID != job.ID || response.DatabaseID != databaseID || response.Status != StatusPending {
		t.Fatalf("status response = %+v, want pending job %s for database %d", response, job.ID, databaseID)
	}

	for _, tt := range []struct {
		name       string
		userID     int
		statusCode int
	}{
		{"view-only user", viewUserID, http.StatusForbidden},
		{"user without access", hiddenUserID, http.StatusNotFound},
		{"edit user cannot resume pending plain job", editUserID, http.StatusConflict},
	} {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/jobs/"+job.ID+"/resume", nil)
			request.SetPathValue("id", job.ID)
			request = request.WithContext(context.WithValue(request.Context(), auth.SessionContextKey,
				&auth.Session{UserID: tt.userID}))
			result := httptest.NewRecorder()
			ResumeHandler(store, db).ServeHTTP(result, request)
			if result.Code != tt.statusCode {
				t.Fatalf("resume response = %d, want %d: %s", result.Code, tt.statusCode, result.Body.String())
			}
		})
	}

	unauthenticated := httptest.NewRequest(http.MethodPost, "/api/jobs/"+job.ID+"/resume", nil)
	unauthenticated.SetPathValue("id", job.ID)
	unauthenticatedResponse := httptest.NewRecorder()
	ResumeHandler(store, db).ServeHTTP(unauthenticatedResponse, unauthenticated)
	if unauthenticatedResponse.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated resume response = %d, want 401", unauthenticatedResponse.Code)
	}

	hiddenStatus := httptest.NewRequest(http.MethodGet, "/api/jobs/"+job.ID, nil)
	hiddenStatus.SetPathValue("id", job.ID)
	hiddenStatus = hiddenStatus.WithContext(context.WithValue(hiddenStatus.Context(), auth.SessionContextKey,
		&auth.Session{UserID: hiddenUserID}))
	hiddenResponse := httptest.NewRecorder()
	statusHandler.ServeHTTP(hiddenResponse, hiddenStatus)
	if hiddenResponse.Code != http.StatusNotFound {
		t.Fatalf("hidden status response = %d, want 404", hiddenResponse.Code)
	}
	viewStatus := httptest.NewRequest(http.MethodGet, "/api/jobs/"+job.ID, nil)
	viewStatus.SetPathValue("id", job.ID)
	viewStatus = viewStatus.WithContext(context.WithValue(viewStatus.Context(), auth.SessionContextKey,
		&auth.Session{UserID: viewUserID, Name: viewName}))
	viewResponse := httptest.NewRecorder()
	statusHandler.ServeHTTP(viewResponse, viewStatus)
	if viewResponse.Code != http.StatusOK {
		t.Fatalf("view-only status response = %d, want 200", viewResponse.Code)
	}

	const batchSQL = `UPDATE batch_items SET value = value + 1 WHERE id > COALESCE($1::bigint,0) AND id <= COALESCE($1::bigint,0) + $2::int`
	var sourceRequestID int64
	if err := db.QueryRow(`
		INSERT INTO migration_change_requests
			(database_id,requester_user_id,requester_name,action,sql_up,sql_down,
			 is_irreversible,irreversible_reason,mode,batch_size,status)
		VALUES ($1,$2,$3,'up',$4,'',TRUE,'Integration test','batch',10,'partial')
		RETURNING id`, databaseID, editUserID, editName, batchSQL).Scan(&sourceRequestID); err != nil {
		t.Fatalf("insert partial batch request: %v", err)
	}
	sum := sha256.Sum256([]byte(batchSQL))
	resumableJob, err := store.Enqueue(ctx, EnqueueRequest{
		DatabaseID: databaseID, Kind: "up", Mode: ModeBatch, BatchSize: 10,
		SQL: batchSQL, Irreversible: true, IrreversibleReason: "Integration test",
		WorkflowRequestID: sourceRequestID, RequestedBy: editName,
	})
	if err != nil {
		t.Fatalf("enqueue resumable batch: %v", err)
	}
	if _, err := db.Exec(`
		UPDATE migration_jobs SET status='partial',holds_lock=TRUE,batches_committed=1,rows_committed=10
		WHERE id=$1 AND sql_sha256=$2`, resumableJob.ID, hex.EncodeToString(sum[:])); err != nil {
		t.Fatalf("make batch job partial: %v", err)
	}
	resumeRequest := httptest.NewRequest(http.MethodPost, "/api/jobs/"+resumableJob.ID+"/resume", nil)
	resumeRequest.SetPathValue("id", resumableJob.ID)
	resumeRequest = resumeRequest.WithContext(context.WithValue(resumeRequest.Context(), auth.SessionContextKey,
		&auth.Session{UserID: editUserID, Name: editName}))
	resumeResponse := httptest.NewRecorder()
	ResumeHandler(store, db).ServeHTTP(resumeResponse, resumeRequest)
	if resumeResponse.Code != http.StatusAccepted {
		t.Fatalf("eligible resume response = %d, want 202: %s", resumeResponse.Code, resumeResponse.Body.String())
	}
	var accepted struct {
		JobID     string `json:"job_id"`
		RequestID int64  `json:"request_id"`
		Status    string `json:"status"`
	}
	if err := json.Unmarshal(resumeResponse.Body.Bytes(), &accepted); err != nil {
		t.Fatal(err)
	}
	if accepted.JobID != resumableJob.ID || accepted.RequestID < 1 || accepted.Status != "pending_review" {
		t.Fatalf("accepted resume response = %+v, want pending approval for job %s", accepted, resumableJob.ID)
	}

	notFound := httptest.NewRequest(http.MethodGet, "/api/jobs/not-a-job", nil)
	notFound.SetPathValue("id", "not-a-job")
	notFoundResponse := httptest.NewRecorder()
	statusHandler.ServeHTTP(notFoundResponse, notFound)
	if notFoundResponse.Code != http.StatusNotFound {
		t.Fatalf("unknown job response = %d, want 404", notFoundResponse.Code)
	}
}
