package jobs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/dbmanager"
	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/workflow"
	"github.com/lib/pq"
)

func TestStoreClaimRetryLockAndReaper(t *testing.T) {
	db := openJobsIntegrationDB(t)
	ctx := context.Background()
	databaseID := createJobsTestDatabase(t, db)
	store := NewStore(db)
	first := enqueueJobsTestJob(t, ctx, store, databaseID, EnqueueRequest{SQL: "SELECT 1"})
	second := enqueueJobsTestJob(t, ctx, store, databaseID, EnqueueRequest{SQL: "SELECT 2"})
	if _, err := db.ExecContext(ctx, `UPDATE migration_jobs SET created_at=created_at+interval '1 second' WHERE id=$1`, second.ID); err != nil {
		t.Fatalf("order queued jobs: %v", err)
	}

	claimed, err := store.Claim(ctx, "worker-one")
	if err != nil || claimed == nil || claimed.ID != first.ID {
		t.Fatalf("first Claim = %+v, %v; want oldest job %s", claimed, err, first.ID)
	}
	if owned, err := store.Heartbeat(ctx, first.ID, "wrong-worker"); err != nil || owned {
		t.Fatalf("wrong-owner Heartbeat = %v, %v; want false", owned, err)
	}
	if owned, err := store.Heartbeat(ctx, first.ID, "worker-one"); err != nil || !owned {
		t.Fatalf("owner Heartbeat = %v, %v; want true", owned, err)
	}
	if next, err := store.Claim(ctx, "worker-two"); err != nil || next != nil {
		t.Fatalf("Claim while database lock held = %+v, %v; want no job", next, err)
	}

	if err := store.ScheduleRetry(ctx, first.ID, time.Hour, "serialization failure", "40001"); err != nil {
		t.Fatalf("ScheduleRetry: %v", err)
	}
	if next, err := store.Claim(ctx, "worker-two"); err != nil || next != nil {
		t.Fatalf("Claim while retry is waiting = %+v, %v; want no job", next, err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE migration_jobs SET run_after=now() WHERE id=$1`, first.ID); err != nil {
		t.Fatalf("make retry runnable: %v", err)
	}
	retried, err := store.Claim(ctx, "worker-one")
	if err != nil || retried == nil || retried.ID != first.ID || retried.Attempt != 2 {
		t.Fatalf("retry Claim = %+v, %v; want first job attempt 2", retried, err)
	}
	if err := store.MarkSuccess(ctx, first.ID); err != nil {
		t.Fatalf("MarkSuccess: %v", err)
	}

	claimed, err = store.Claim(ctx, "worker-two")
	if err != nil || claimed == nil || claimed.ID != second.ID {
		t.Fatalf("Claim after lock release = %+v, %v; want second job %s", claimed, err, second.ID)
	}
	if _, err := db.ExecContext(ctx, `UPDATE migration_jobs SET heartbeat_at=now()-interval '1 hour' WHERE id=$1`, second.ID); err != nil {
		t.Fatalf("age worker heartbeat: %v", err)
	}
	if reaped, err := store.ReapStale(ctx, time.Minute); err != nil || reaped != 1 {
		t.Fatalf("ReapStale = %d, %v; want one uncertain job", reaped, err)
	}
	uncertain, err := store.ListUncertain(ctx)
	if err != nil || len(uncertain) != 1 || uncertain[0].ID != second.ID {
		t.Fatalf("ListUncertain = %+v, %v; want second job", uncertain, err)
	}
	if next, err := store.Claim(ctx, "worker-three"); err != nil || next != nil {
		t.Fatalf("Claim while uncertain lock held = %+v, %v; want no job", next, err)
	}
}

func TestStoreEnqueueRejectsFullQueue(t *testing.T) {
	db := openJobsIntegrationDB(t)
	ctx := context.Background()
	databaseID := createJobsTestDatabase(t, db)
	store := NewStore(db).WithMaxPending(1)
	enqueueJobsTestJob(t, ctx, store, databaseID, EnqueueRequest{SQL: "SELECT 1"})
	if _, err := store.Enqueue(ctx, EnqueueRequest{
		DatabaseID: databaseID, SQL: "SELECT 2", RequestedBy: "integration-test",
	}); !errors.Is(err, ErrQueueFull) {
		t.Fatalf("second Enqueue error = %v, want ErrQueueFull", err)
	}
}

type workerTestExecutor struct {
	executeErr    error
	wasApplied    bool
	wasAppliedErr error
}

func (e workerTestExecutor) NextVersion(context.Context, int) (int64, error) {
	return 1, nil
}

func (e workerTestExecutor) Execute(context.Context, Job, int64) error {
	return e.executeErr
}

func (e workerTestExecutor) WasApplied(context.Context, Job, int64) (bool, error) {
	return e.wasApplied, e.wasAppliedErr
}

type workerTestValidator struct{ err error }

func (v workerTestValidator) Validate(context.Context, workflow.ExecutionJob) error { return v.err }

func TestWorkerRetriesFailsAndResolvesUncertainCommit(t *testing.T) {
	db := openJobsIntegrationDB(t)
	ctx := context.Background()
	store := NewStore(db)
	tests := []struct {
		name         string
		exec         workerTestExecutor
		wantStatus   Status
		wantSQLState string
	}{
		{"transient retry", workerTestExecutor{executeErr: &pq.Error{Code: "40001"}}, StatusPending, "40001"},
		{"permanent failure", workerTestExecutor{executeErr: errors.New("invalid SQL")}, StatusFailed, ""},
		{"uncertain commit applied", workerTestExecutor{
			executeErr: &dbmanager.CommitError{Version: 1, Err: errors.New("connection lost")},
			wasApplied: true,
		}, StatusSuccess, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			databaseID := createJobsTestDatabase(t, db)
			job := enqueueJobsTestJob(t, ctx, store, databaseID, EnqueueRequest{SQL: "SELECT 1"})
			claimed, err := store.Claim(ctx, "integration-worker")
			if err != nil || claimed == nil || claimed.ID != job.ID {
				t.Fatalf("Claim = %+v, %v", claimed, err)
			}
			worker := NewWorker(store, tt.exec, workerTestValidator{}, WorkerConfig{
				ID: "integration-worker", BaseBackoff: time.Hour, HeartbeatInterval: time.Hour,
			})
			now := time.Now()
			worker.process(ctx, *claimed, now, now)
			got, err := store.Get(ctx, job.ID)
			if err != nil {
				t.Fatal(err)
			}
			if got.Status != tt.wantStatus {
				t.Fatalf("job status = %q, want %q (last error %q)", got.Status, tt.wantStatus, got.LastError.String)
			}
			if tt.wantSQLState != "" && got.LastSQLState.String != tt.wantSQLState {
				t.Fatalf("last SQLSTATE = %q, want %q", got.LastSQLState.String, tt.wantSQLState)
			}
		})
	}
}

func TestWorkerReaperResolvesUncertainJob(t *testing.T) {
	db := openJobsIntegrationDB(t)
	ctx := context.Background()
	store := NewStore(db)
	databaseID := createJobsTestDatabase(t, db)
	job := enqueueJobsTestJob(t, ctx, store, databaseID, EnqueueRequest{SQL: "SELECT 1"})
	claimed, err := store.Claim(ctx, "crashed-worker")
	if err != nil || claimed == nil {
		t.Fatalf("Claim = %+v, %v", claimed, err)
	}
	if err := store.SetVersion(ctx, job.ID, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE migration_jobs SET heartbeat_at=now()-interval '1 hour' WHERE id=$1`, job.ID); err != nil {
		t.Fatal(err)
	}
	worker := NewWorker(store, workerTestExecutor{wasApplied: true}, workerTestValidator{}, WorkerConfig{
		ID: "recovery-worker", StaleAfter: time.Minute,
	})
	worker.reapOnce(ctx)
	got, err := store.Get(ctx, job.ID)
	if err != nil || got.Status != StatusSuccess {
		t.Fatalf("recovered job = %+v, %v; want success", got, err)
	}
}
