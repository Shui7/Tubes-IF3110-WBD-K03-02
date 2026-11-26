package jobs

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/observability"
	"go.opentelemetry.io/otel/attribute"
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusSuccess   Status = "success"
	StatusFailed    Status = "failed"
	StatusUncertain Status = "uncertain"
	// TODO: status partial / partially_reverted for batch mode.
)

// DefaultMaxPendingPerDatabase is the default limit of waiting jobs per database.
const DefaultMaxPendingPerDatabase = 100

var (
	ErrQueueFull = errors.New("the migration queue for this database is full")
	ErrNotFound  = errors.New("job not found")
)

type Job struct {
	ID           string // same as execution_id
	DatabaseID   int
	Kind         string
	Name         string
	SQL          string
	Version      sql.NullInt64
	Status       Status
	Attempt      int
	MaxAttempts  int
	RunAfter     time.Time
	WorkerID     sql.NullString
	Traceparent  string
	RequestID    string
	RequestedBy  string
	LastError    sql.NullString
	LastSQLState sql.NullString
	CreatedAt    time.Time
	StartedAt    sql.NullTime
	FinishedAt   sql.NullTime
}

type EnqueueRequest struct {
	DatabaseID  int
	Kind        string // "up" (default), "rollback", "forward"
	Name        string
	SQL         string
	RequestedBy string
	RequestID   string
}

type Store struct {
	db         *sql.DB
	maxPending int
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db, maxPending: DefaultMaxPendingPerDatabase}
}

// WithMaxPending sets how many jobs may wait ('pending') per database.
// Values below 1 keep the default.
func (s *Store) WithMaxPending(n int) *Store {
	if n >= 1 {
		s.maxPending = n
	}
	return s
}

const jobColumns = `id, database_id, kind, name, sql_text, version, status, attempt, max_attempts,
	run_after, worker_id, COALESCE(traceparent, ''), COALESCE(request_id, ''), requested_by,
	last_error, last_sqlstate, created_at, started_at, finished_at`

type scanner interface{ Scan(dest ...any) error }

func scanJob(row scanner) (Job, error) {
	var j Job
	err := row.Scan(&j.ID, &j.DatabaseID, &j.Kind, &j.Name, &j.SQL, &j.Version, &j.Status,
		&j.Attempt, &j.MaxAttempts, &j.RunAfter, &j.WorkerID, &j.Traceparent, &j.RequestID,
		&j.RequestedBy, &j.LastError, &j.LastSQLState, &j.CreatedAt, &j.StartedAt, &j.FinishedAt)
	return j, err
}

func newID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	h := hex.EncodeToString(b[:])
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:32]
}

func nullIfEmpty(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

// Enqueue stores a pending job and returns immediately. It is accepted even when
// the database is busy or blocked (uncertain/partial job), except when queue is full.
//
// The current trace context is saved in the job so that the worker's spans
// become children of this request's trace.
func (s *Store) Enqueue(ctx context.Context, req EnqueueRequest) (Job, error) {
	ctx, span := observability.Tracer("jobs").Start(ctx, "job.enqueue")
	defer span.End()

	if strings.TrimSpace(req.SQL) == "" {
		return Job{}, errors.New("migration SQL is required")
	}
	if strings.TrimSpace(req.RequestedBy) == "" {
		return Job{}, errors.New("migration executor is required")
	}
	kind := req.Kind
	if kind == "" {
		kind = "up"
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		observability.RecordError(span, err)
		return Job{}, fmt.Errorf("begin enqueue: %w", err)
	}
	defer tx.Rollback()

	// Serialize enqueues per database so the limit check and the insert cannot race.
	if _, err := tx.ExecContext(ctx,
		`SELECT pg_advisory_xact_lock(hashtext('migration_jobs.enqueue'), $1::int)`, req.DatabaseID); err != nil {
		observability.RecordError(span, err)
		return Job{}, fmt.Errorf("lock enqueue: %w", err)
	}
	var waiting int
	if err := tx.QueryRowContext(ctx,
		`SELECT count(*) FROM migration_jobs WHERE database_id = $1 AND status = 'pending'`,
		req.DatabaseID).Scan(&waiting); err != nil {
		observability.RecordError(span, err)
		return Job{}, fmt.Errorf("count waiting jobs: %w", err)
	}
	span.SetAttributes(attribute.Int("queue.waiting", waiting))
	if waiting >= s.maxPending {
		return Job{}, fmt.Errorf("%w (limit %d)", ErrQueueFull, s.maxPending)
	}

	sum := sha256.Sum256([]byte(req.SQL))
	id := newID()
	span.SetAttributes(attribute.String("job.id", id), attribute.Int("database.id", req.DatabaseID))

	job, err := scanJob(tx.QueryRowContext(ctx, `
		INSERT INTO migration_jobs
			(id, database_id, kind, name, sql_text, sql_sha256, traceparent, request_id, requested_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING `+jobColumns,
		id, req.DatabaseID, kind, req.Name, req.SQL, hex.EncodeToString(sum[:]),
		nullIfEmpty(observability.InjectTraceparent(ctx)), nullIfEmpty(req.RequestID), req.RequestedBy))
	if err != nil {
		observability.RecordError(span, err)
		return Job{}, fmt.Errorf("enqueue job: %w", err)
	}
	if err := tx.Commit(); err != nil {
		observability.RecordError(span, err)
		return Job{}, fmt.Errorf("commit enqueue: %w", err)
	}
	return job, nil
}

// Claim atomically takes the oldest runnable job and acquires the per-database
// lock for it. It returns (nil, nil) when there is nothing to do.
func (s *Store) Claim(ctx context.Context, workerID string) (*Job, error) {
	job, err := scanJob(s.db.QueryRowContext(ctx, `
		UPDATE migration_jobs SET
			status = 'running', holds_lock = TRUE, worker_id = $1, heartbeat_at = now(),
			attempt = attempt + 1, started_at = COALESCE(started_at, now())
		WHERE id = (
			SELECT j.id FROM migration_jobs j
			WHERE j.status = 'pending' AND j.run_after <= now()
			  -- the database is not locked by another job (running, retrying, uncertain, partial)
			  AND NOT EXISTS (SELECT 1 FROM migration_jobs h
			                  WHERE h.database_id = j.database_id AND h.holds_lock AND h.id <> j.id)
			  -- strict FIFO per database: no older job is still waiting
			  AND NOT EXISTS (SELECT 1 FROM migration_jobs o
			                  WHERE o.database_id = j.database_id AND o.status = 'pending'
			                    AND (o.created_at, o.id) < (j.created_at, j.id))
			ORDER BY j.created_at, j.id
			FOR UPDATE SKIP LOCKED
			LIMIT 1)
		RETURNING `+jobColumns, workerID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		// Another worker already claimed a job for this database. Try again later.
		if SQLState(err) == "23505" {
			return nil, nil
		}
		return nil, fmt.Errorf("claim job: %w", err)
	}
	return &job, nil
}

func (s *Store) Get(ctx context.Context, id string) (Job, error) {
	job, err := scanJob(s.db.QueryRowContext(ctx,
		`SELECT `+jobColumns+` FROM migration_jobs WHERE id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) || SQLState(err) == "22P02" { // invalid uuid text
		return Job{}, ErrNotFound
	}
	if err != nil {
		return Job{}, fmt.Errorf("get job: %w", err)
	}
	return job, nil
}

// Heartbeat extends the lease. It returns false when the worker no longer owns
// the job (for example the reaper took it over).
func (s *Store) Heartbeat(ctx context.Context, id, workerID string) (bool, error) {
	res, err := s.db.ExecContext(ctx, `
		UPDATE migration_jobs SET heartbeat_at = now()
		WHERE id = $1 AND worker_id = $2 AND status = 'running'`, id, workerID)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
}

func (s *Store) SetVersion(ctx context.Context, id string, version int64) error {
	_, err := s.db.ExecContext(ctx, `UPDATE migration_jobs SET version = $2 WHERE id = $1`, id, version)
	return err
}

func (s *Store) MarkSuccess(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE migration_jobs SET status = 'success', holds_lock = FALSE, worker_id = NULL,
			last_error = NULL, last_sqlstate = NULL, finished_at = now()
		WHERE id = $1 AND status IN ('running','uncertain')`, id)
	return err
}

func (s *Store) MarkFailed(ctx context.Context, id, message, sqlstate string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE migration_jobs SET status = 'failed', holds_lock = FALSE, worker_id = NULL,
			last_error = $2, last_sqlstate = $3, finished_at = now()
		WHERE id = $1 AND status IN ('running','uncertain')`,
		id, message, nullIfEmpty(sqlstate))
	return err
}

// ScheduleRetry puts the job back in the queue after a delay. The lock stays
// held so no other migration can slip in between attempts.
func (s *Store) ScheduleRetry(ctx context.Context, id string, delay time.Duration, message, sqlstate string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE migration_jobs SET status = 'pending', worker_id = NULL, heartbeat_at = NULL,
			run_after = now() + ($2::double precision * interval '1 second'),
			last_error = $3, last_sqlstate = $4
		WHERE id = $1 AND status IN ('running','uncertain')`,
		id, delay.Seconds(), message, nullIfEmpty(sqlstate))
	return err
}

// Requeue returns a job to the queue without counting the attempt (graceful shutdown).
func (s *Store) Requeue(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE migration_jobs SET status = 'pending', worker_id = NULL, heartbeat_at = NULL,
			attempt = GREATEST(attempt - 1, 0)
		WHERE id = $1 AND status = 'running'`, id)
	return err
}

// MarkUncertain keeps the lock held: the outcome of the last attempt is unknown.
func (s *Store) MarkUncertain(ctx context.Context, id, message string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE migration_jobs SET status = 'uncertain', worker_id = NULL, last_error = $2
		WHERE id = $1 AND status IN ('running','uncertain')`, id, message)
	return err
}

// ReapStale moves running jobs whose heartbeat stopped to 'uncertain'.
func (s *Store) ReapStale(ctx context.Context, staleAfter time.Duration) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
		UPDATE migration_jobs SET status = 'uncertain', worker_id = NULL,
			last_error = 'worker heartbeat lost'
		WHERE status = 'running'
		  AND heartbeat_at < now() - ($1::double precision * interval '1 second')`,
		staleAfter.Seconds())
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *Store) ListUncertain(ctx context.Context) ([]Job, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+jobColumns+` FROM migration_jobs WHERE status = 'uncertain' ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var jobs []Job
	for rows.Next() {
		j, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

// schemaLockKey is an arbitrary constant used with pg_advisory_lock so that the
// app and the worker do not run EnsureSchema concurrently at startup.
const schemaLockKey = 727001

const schemaSQL = `
CREATE TABLE IF NOT EXISTS migration_jobs (
    id            UUID PRIMARY KEY,                 -- also used as execution_id
    database_id   INT  NOT NULL REFERENCES database_connections(id) ON DELETE CASCADE,
    kind          TEXT NOT NULL DEFAULT 'up'    CHECK (kind IN ('up','rollback','forward')),
    mode          TEXT NOT NULL DEFAULT 'plain' CHECK (mode IN ('plain','batch')),
    name          TEXT NOT NULL DEFAULT '',
    sql_text      TEXT NOT NULL,
    sql_sha256    TEXT NOT NULL,
    version       BIGINT,                           -- migration version chosen by the worker
    status        TEXT NOT NULL DEFAULT 'pending' CHECK (status IN
                  ('pending','running','success','failed','partial','uncertain','partially_reverted')),
    holds_lock    BOOLEAN NOT NULL DEFAULT FALSE,   -- per-database lock, see one_lock_per_db
    attempt       INT NOT NULL DEFAULT 0,
    max_attempts  INT NOT NULL DEFAULT 3,
    run_after     TIMESTAMPTZ NOT NULL DEFAULT now(),
    worker_id     TEXT,
    heartbeat_at  TIMESTAMPTZ,
    traceparent   TEXT,
    request_id    TEXT,
    requested_by  TEXT NOT NULL,
    last_error    TEXT,
    last_sqlstate TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    started_at    TIMESTAMPTZ,
    finished_at   TIMESTAMPTZ
);

-- At most one job per database may hold the lock (across workers and across batches).
CREATE UNIQUE INDEX IF NOT EXISTS one_lock_per_db ON migration_jobs(database_id) WHERE holds_lock;
-- Queues are long: many jobs may wait per database (the limit is enforced in Enqueue).
-- This index existed in an earlier draft and forbade more than one waiting job.
DROP INDEX IF EXISTS one_pending_per_db;
CREATE INDEX IF NOT EXISTS migration_jobs_claim ON migration_jobs(status, run_after);
CREATE INDEX IF NOT EXISTS migration_jobs_db_status ON migration_jobs(database_id, status, created_at);
`

// EnsureSchema creates the queue tables in the platform database.
// Safe to call from several processes at the same time.
func EnsureSchema(ctx context.Context, db *sql.DB) error {
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("jobs schema: get connection: %w", err)
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, `SELECT pg_advisory_lock($1)`, schemaLockKey); err != nil {
		return fmt.Errorf("jobs schema: lock: %w", err)
	}
	defer func() {
		_, _ = conn.ExecContext(context.WithoutCancel(ctx), `SELECT pg_advisory_unlock($1)`, schemaLockKey)
	}()

	if _, err := conn.ExecContext(ctx, schemaSQL); err != nil {
		return fmt.Errorf("jobs schema: create tables: %w", err)
	}
	return nil
}
