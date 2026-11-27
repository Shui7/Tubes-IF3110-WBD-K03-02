# Database Management Platform

A web application for managing PostgreSQL database connections, users, groups, and group access permissions. Users can sign in to browse available databases, while administrators can manage connections and access control.

## Project description

For Tugas Besar IF3110 Web Application Development

By K03-02:

- 13524003 Faiq Azzam Nafidz
- 13524018 Muhammad Nafis Habibi
- 13524072 I Gusti Ngurah Alit Dharma Y

## Run with Docker

### Prerequisites

- Docker Engine or Docker Desktop
- Docker Compose v2 (`docker compose`)

### Start the application

From the repository root, run:

```sh
docker compose up --build -d
```

Compose starts the web application, the platform PostgreSQL database, and a sample PostgreSQL database. Wait for the services to become healthy, then open:

<http://localhost:8080>

The application listens on port `8080` by default. The sample PostgreSQL server is published on port `5433` by default.

### Sign in

On a fresh platform database, the application seeds this development administrator account:

- Username: `admin`
- Password: `admin123`

Use this account for local development only. Change the default credentials before exposing the application to other users or networks.

### Useful Docker commands

View service status:

```sh
docker compose ps
```

Follow application logs:

```sh
docker compose logs -f app
```

Stop the services:

```sh
docker compose down
```

The PostgreSQL data is stored in named Docker volumes and is kept when you run `docker compose down`. To remove the database volumes and their data as well, run:

```sh
docker compose down --volumes
```

This permanently deletes the platform and sample database data.

## Configuration

Docker Compose reads environment variables from the shell or a `.env` file in the repository root. The platform database connection must use the Compose service name `dbplatform` as its host; containers should not use `localhost` or a host-mapped port to reach one another.

Common variables:

| Variable | Default | Purpose |
| --- | --- | --- |
| `POSTGRES_USER` | `postgres` | Platform database user |
| `POSTGRES_PASSWORD` | `postgres` | Platform database password |
| `POSTGRES_DB` | `dbplatform` | Platform database name |
| `DATABASE_URL` | Compose-configured connection to `dbplatform:5432` | Connection string used by the application |
| `APP_PORT` | `8080` | Host port for the web application |
| `DBSAMPLES_PORT` | `5433` | Host port for the sample PostgreSQL server |

If you override the platform database credentials, also set `DATABASE_URL` to matching values. For example:

```dotenv
POSTGRES_USER=postgres
POSTGRES_PASSWORD=replace-with-a-local-password
POSTGRES_DB=dbplatform
DATABASE_URL=host=dbplatform port=5432 dbname=dbplatform user=postgres password=replace-with-a-local-password sslmode=disable
```

## Development

Run the Go test suite:

```sh
go test ./...
```

The platform schema is in [`db/platform/schema.sql`](db/platform/schema.sql). Sample database SQL files are loaded by [`db/init-samples.sh`](db/init-samples.sh) when the sample database volume is initialized for the first time.

## Activity logs and tracing

The HTTP server assigns each request a fresh `X-Request-ID`. The ID is included
in application logs and is stored with queued jobs so worker events can be
correlated with the request that created them. Passwords, session identifiers,
CSRF tokens, query parameter values, and returned row data are not logged.
Database query events include the redacted statement, parameter types, elapsed
time, affected/returned row count, and PostgreSQL SQLSTATE on failure. Commit
and rollback outcomes are recorded as separate transaction events.

`OTEL_EXPORTER` controls span output:

| Value | Behavior |
| --- | --- |
| `console` (default) | Write spans to standard error |
| `otlp` | Export spans to the configured OpenTelemetry Collector |
| `none` | Create spans without exporting them |

Example application log:

```json
{"time":"2026-11-27T10:00:00Z","level":"INFO","event":"db.query.completed","service":"app","trace_id":"4bf92f3577b34da6a3ce929d0e0e4736","span_id":"00f067aa0ba902b7","request_id":"b7d1b3d61289490bb876772e0e2e8b2a","operation":"table.count","query":"SELECT COUNT(*) FROM \"public\".\"items\"","parameters":[],"duration_ms":3,"rows":1,"ok":true}
```

Representative console span (the SDK's exact formatting may vary):

```text
Name: db.query
Trace ID: 4bf92f3577b34da6a3ce929d0e0e4736
Span ID: 00f067aa0ba902b7
Attributes:
  db.system = postgresql
  db.operation = table.count
  db.statement = SELECT COUNT(*) FROM "public"."items"
  db.rows_affected = 1
  db.duration_ms = 3
```

## Lock Recovery When a Worker Fails

The per-database lock is stored in the platform database as the `holds_lock` column of `migration_jobs`, protected by the partial unique index `one_lock_per_db`. At most one job per database can hold the lock, across all workers, and the lock survives restarts. The lock is acquired atomically in `Claim()` (using `FOR UPDATE SKIP LOCKED` and strict per-database FIFO order). It is released only when a job reaches `success` or `failed`. While a job is `pending` (waiting for a retry), `uncertain`, or `partial`, the lock stays held so no other migration can slip in.

**Recovery flow.**

1. **Detection.** The worker updates `heartbeat_at` every 5 seconds. If the worker dies, the heartbeat stops.
2. **Reaper.** Every worker runs a periodic reaper. A `running` job whose heartbeat is older than 30 seconds is moved to `uncertain`. The lock is not released, because the SQL may have been committed before the worker died.
3. **Evidence check.** Before retrying, the outcome is verified:
   - *Regular migration:* the target database's migration history is checked for a successful record with this job ID and version. If found, the job is marked `success` (lock released). If not, the job is rescheduled with backoff while attempts remain, otherwise marked `failed` (lock released).
   - *Batch migration:* checkpoints are stored in the target database in the same transaction as each batch's changes, so the job resumes from the last checkpoint without repeating committed batches. If attempts are exhausted, the job becomes `partial` or `failed`.
   - *Evidence unreadable* (e.g. the target database is unavailable): the job stays `uncertain` with the lock held, and the next reaper cycle tries again.

**Other cases.**

- **Slow worker, not dead.** If the reaper has already taken over the job, the old worker's heartbeat fails (`job.lost`). That worker cancels its work and changes no state, so there is only ever one owner of a job.
- **Graceful shutdown (SIGTERM).** The job is returned to the queue without consuming an attempt, and the lock stays held.
- **Ambiguous commit** (connection lost during `COMMIT`). The job is immediately treated as `uncertain` and follows the same evidence check.
- **No double commit.** The unique lock index prevents two workers from running on the same database, and a retry happens only after evidence confirms the SQL was not committed.