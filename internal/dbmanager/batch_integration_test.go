package dbmanager

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"
)

// batchSQL builds a batch statement for table (id BIGINT PRIMARY KEY, n INT):
// it changes at most $2 rows after cursor $1 and returns (new cursor, rows changed).
func batchSQL(table, setExpr string) string {
	return `WITH picked AS (
			SELECT id FROM "` + table + `" WHERE id > COALESCE($1::bigint, 0) ORDER BY id LIMIT $2::int
		), upd AS (
			UPDATE "` + table + `" t SET n = ` + setExpr + ` FROM picked p WHERE t.id = p.id RETURNING t.id
		)
		SELECT MAX(id)::text, COUNT(*) FROM upd`
}

func newBatchTable(t *testing.T, db *sql.DB, rows int) (table, executionPrefix string) {
	t.Helper()
	table = fmt.Sprintf("batch_it_%d", time.Now().UnixNano())
	executionPrefix = table + "_exec"
	if _, err := db.Exec(`CREATE TABLE "` + table + `" (id BIGINT PRIMARY KEY, n INT NOT NULL DEFAULT 0)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO "`+table+`" (id) SELECT g FROM generate_series(1, $1) g`, rows); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec(`DROP TABLE IF EXISTS "` + table + `"`)
		_, _ = db.Exec(`DELETE FROM db_platform_checkpoints WHERE execution_id LIKE $1`, executionPrefix+"%")
		_, _ = db.Exec(`DELETE FROM db_platform_migrations WHERE sql_text LIKE $1`, "%"+table+"%")
	})
	return table, executionPrefix
}

// runBatches executes batches the way the worker does. maxBatches <= 0 means no limit.
func runBatches(ctx context.Context, m *Manager, executionID, batchStatement string, size, maxBatches int) (BatchProgress, error) {
	p, err := m.BatchProgress(ctx, executionID)
	if err != nil {
		return p, err
	}
	for i := 0; maxBatches <= 0 || i < maxBatches; i++ {
		step, err := m.ExecuteBatch(ctx, executionID, p.Batches+1, batchStatement, p.Cursor, size)
		if err != nil {
			return p, err
		}
		if step.Done {
			break
		}
		p.Batches++
		p.Rows += step.Rows
		p.Cursor = step.Cursor
	}
	return p, nil
}

func countWhere(t *testing.T, db *sql.DB, table, where string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM "` + table + `" WHERE ` + where).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestBatchRunsAllBatchesWithCheckpoints(t *testing.T) {
	db := openIntegrationDatabase(t)
	ctx := context.Background()
	m := New(db)
	table, prefix := newBatchTable(t, db, 25)

	p, err := runBatches(ctx, m, prefix+"_a", batchSQL(table, "t.n + 1"), 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if p.Batches != 3 || p.Rows != 25 {
		t.Fatalf("progress = %d batches, %d rows; want 3 batches, 25 rows", p.Batches, p.Rows)
	}
	if got := countWhere(t, db, table, "n = 1"); got != 25 {
		t.Fatalf("%d rows updated exactly once, want 25", got)
	}

	// The checkpoint table is the source of truth.
	stored, err := m.BatchProgress(ctx, prefix+"_a")
	if err != nil {
		t.Fatal(err)
	}
	if stored.Batches != 3 || stored.Rows != 25 || stored.Cursor.String != "25" {
		t.Fatalf("stored progress = %+v, want 3 batches, 25 rows, cursor 25", stored)
	}
}

func TestBatchResumeDoesNotRepeatCommittedBatches(t *testing.T) {
	db := openIntegrationDatabase(t)
	ctx := context.Background()
	m := New(db)
	table, prefix := newBatchTable(t, db, 25)
	exec := prefix + "_b"
	stmt := batchSQL(table, "t.n + 1") // not idempotent: a repeated batch would make n = 2

	// First run stops after one batch (like a crash), then a new run continues.
	if _, err := runBatches(ctx, m, exec, stmt, 10, 1); err != nil {
		t.Fatal(err)
	}
	p, err := runBatches(ctx, m, exec, stmt, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if p.Batches != 3 || p.Rows != 25 {
		t.Fatalf("progress after resume = %d batches, %d rows; want 3 and 25", p.Batches, p.Rows)
	}
	if got := countWhere(t, db, table, "n = 1"); got != 25 {
		t.Fatalf("%d rows updated exactly once, want 25 (a batch was repeated or skipped)", got)
	}
}

func TestBatchFailureKeepsPreviousBatchesAndRollsBackTheFailedOne(t *testing.T) {
	db := openIntegrationDatabase(t)
	ctx := context.Background()
	m := New(db)
	table, prefix := newBatchTable(t, db, 25)
	exec := prefix + "_c"

	// Row 15 (second batch of 10) causes division by zero.
	failing := batchSQL(table, "t.n + 1 / (CASE WHEN t.id = 15 THEN 0 ELSE 1 END)")
	p, err := runBatches(ctx, m, exec, failing, 10, 0)
	if err == nil {
		t.Fatal("expected the second batch to fail")
	}
	if p.Batches != 1 || p.Rows != 10 {
		t.Fatalf("progress = %d batches, %d rows; want 1 batch, 10 rows", p.Batches, p.Rows)
	}
	if got := countWhere(t, db, table, "id <= 10 AND n = 1"); got != 10 {
		t.Fatalf("first batch: %d rows updated, want 10 (committed batch must stay)", got)
	}
	if got := countWhere(t, db, table, "id > 10 AND n <> 0"); got != 0 {
		t.Fatalf("failed batch left %d changed rows, want 0 (it must roll back)", got)
	}
	stored, err := m.BatchProgress(ctx, exec)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Batches != 1 {
		t.Fatalf("stored batches = %d, want 1 (no checkpoint for the failed batch)", stored.Batches)
	}

	// After fixing the SQL, resuming finishes the remaining rows only.
	p, err = runBatches(ctx, m, exec, batchSQL(table, "t.n + 1"), 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if p.Batches != 3 || p.Rows != 25 {
		t.Fatalf("progress after fix = %d batches, %d rows; want 3 and 25", p.Batches, p.Rows)
	}
	if got := countWhere(t, db, table, "n = 1"); got != 25 {
		t.Fatalf("%d rows updated exactly once, want 25", got)
	}
}

func TestBatchRejectsBadBatchResults(t *testing.T) {
	db := openIntegrationDatabase(t)
	ctx := context.Background()
	m := New(db)

	t.Run("cursor does not advance", func(t *testing.T) {
		table, prefix := newBatchTable(t, db, 5)
		// Always changes rows but always reports the same cursor.
		stmt := `WITH upd AS (UPDATE "` + table + `" SET n = n + 1 WHERE id <= $2::int RETURNING id)
			SELECT '1', COUNT(*) FROM upd WHERE $1::text IS NOT DISTINCT FROM $1::text`
		_, err := runBatches(ctx, m, prefix+"_d", stmt, 5, 0)
		if !errors.Is(err, ErrCursorNotAdvancing) {
			t.Fatalf("error = %v, want ErrCursorNotAdvancing", err)
		}
	})

	t.Run("changed rows but NULL cursor", func(t *testing.T) {
		table, prefix := newBatchTable(t, db, 5)
		stmt := `WITH upd AS (UPDATE "` + table + `" SET n = n + 1 WHERE id <= $2::int RETURNING id)
			SELECT NULL::text, COUNT(*) FROM upd WHERE $1::text IS NOT DISTINCT FROM $1::text`
		_, err := m.ExecuteBatch(ctx, prefix+"_e", 1, stmt, sql.NullString{}, 5)
		if !errors.Is(err, ErrInvalidBatchResult) {
			t.Fatalf("error = %v, want ErrInvalidBatchResult", err)
		}
		if got := countWhere(t, db, table, "n <> 0"); got != 0 {
			t.Fatalf("invalid batch left %d changed rows, want 0", got)
		}
	})

	t.Run("invalid arguments", func(t *testing.T) {
		_, prefix := newBatchTable(t, db, 1)
		if _, err := m.ExecuteBatch(ctx, "", 1, "SELECT 1", sql.NullString{}, 10); err == nil {
			t.Fatal("empty execution id must be rejected")
		}
		if _, err := m.ExecuteBatch(ctx, prefix+"_f", 1, "SELECT 1", sql.NullString{}, 0); err == nil {
			t.Fatal("batch size 0 must be rejected")
		}
	})
}

func TestRecordMigrationSuccessIsIdempotent(t *testing.T) {
	db := openIntegrationDatabase(t)
	ctx := context.Background()
	m := New(db)
	table, _ := newBatchTable(t, db, 1)

	if err := m.EnsureMigrationTable(ctx); err != nil {
		t.Fatal(err)
	}
	version, err := m.NextMigrationVersion(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM db_platform_migrations WHERE version >= $1`, version) })

	stmt := "-- batch migration for " + table
	for i := 0; i < 2; i++ { // second call must be a no-op
		if err := m.RecordMigrationSuccess(ctx, version, stmt, "integration"); err != nil {
			t.Fatalf("call %d: %v", i+1, err)
		}
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM db_platform_migrations WHERE version = $1 AND status = 'success'`, version).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("%d success rows for version %d, want 1", count, version)
	}

	if err := m.RecordMigrationSuccess(ctx, version+5, stmt, "integration"); err == nil {
		t.Fatal("a version that skips ahead must be rejected")
	}
}

// runAttributeBatches executes attribute batches the way the worker does.
// maxBatches <= 0 means no limit.
func runAttributeBatches(ctx context.Context, m *Manager, executionID, statement, table, column string, size, maxBatches int) (BatchProgress, error) {
	p, err := m.BatchProgress(ctx, executionID)
	if err != nil {
		return p, err
	}
	for i := 0; maxBatches <= 0 || i < maxBatches; i++ {
		step, err := m.ExecuteAttributeBatch(ctx, executionID, p.Batches+1, statement, table, column, p.Cursor, size)
		if err != nil {
			return p, err
		}
		if step.Done {
			break
		}
		p.Batches++
		p.Rows += step.Rows
		p.Cursor = step.Cursor
	}
	return p, nil
}

func TestAttributeBatchRunsAllRowsOnce(t *testing.T) {
	db := openIntegrationDatabase(t)
	ctx := context.Background()
	m := New(db)
	table, prefix := newBatchTable(t, db, 25)

	stmt := `UPDATE "` + table + `" SET n = n + 1 WHERE {{batch_filter}}`
	p, err := runAttributeBatches(ctx, m, prefix+"_g", stmt, table, "id", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if p.Batches != 3 || p.Rows != 25 || p.Cursor.String != "25" {
		t.Fatalf("progress = %+v, want 3 batches, 25 rows, cursor 25", p)
	}
	if got := countWhere(t, db, table, "n = 1"); got != 25 {
		t.Fatalf("%d rows updated exactly once, want 25", got)
	}
}

func TestAttributeBatchResumeAndSchemaQualifiedTable(t *testing.T) {
	db := openIntegrationDatabase(t)
	ctx := context.Background()
	m := New(db)
	table, prefix := newBatchTable(t, db, 25)
	exec := prefix + "_h"

	stmt := `UPDATE "` + table + `" SET n = n + 1 WHERE {{batch_filter}}`
	qualified := "public." + table

	// Stop after one batch (like a crash), then continue from the checkpoint.
	if _, err := runAttributeBatches(ctx, m, exec, stmt, qualified, "id", 10, 1); err != nil {
		t.Fatal(err)
	}
	p, err := runAttributeBatches(ctx, m, exec, stmt, qualified, "id", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if p.Batches != 3 || p.Rows != 25 {
		t.Fatalf("progress after resume = %d batches, %d rows; want 3 and 25", p.Batches, p.Rows)
	}
	if got := countWhere(t, db, table, "n = 1"); got != 25 {
		t.Fatalf("%d rows updated exactly once, want 25 (a batch was repeated or skipped)", got)
	}
}

func TestAttributeBatchCommitsRangesWithZeroMatches(t *testing.T) {
	db := openIntegrationDatabase(t)
	ctx := context.Background()
	m := New(db)
	table, prefix := newBatchTable(t, db, 30)

	// Only rows with an even id are changed, plus an extra filter that matches
	// nothing in the second range. Ranges with zero changed rows must still advance.
	stmt := `UPDATE "` + table + `" SET n = n + 1 WHERE {{batch_filter}} AND id % 2 = 0 AND id NOT BETWEEN 11 AND 20`
	p, err := runAttributeBatches(ctx, m, prefix+"_i", stmt, table, "id", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if p.Batches != 3 {
		t.Fatalf("batches = %d, want 3 (the empty range must still be committed)", p.Batches)
	}
	if p.Rows != 10 { // even ids in 1..10 and 21..30 are 5 + 5
		t.Fatalf("rows = %d, want 10", p.Rows)
	}
	if got := countWhere(t, db, table, "n = 1"); got != 10 {
		t.Fatalf("%d rows changed, want 10", got)
	}
}

func TestAttributeBatchFailureRollsBackOnlyThatBatch(t *testing.T) {
	db := openIntegrationDatabase(t)
	ctx := context.Background()
	m := New(db)
	table, prefix := newBatchTable(t, db, 25)
	exec := prefix + "_j"

	failing := `UPDATE "` + table + `" SET n = n + 1 / (CASE WHEN id = 15 THEN 0 ELSE 1 END) WHERE {{batch_filter}}`
	p, err := runAttributeBatches(ctx, m, exec, failing, table, "id", 10, 0)
	if err == nil {
		t.Fatal("expected the second batch to fail")
	}
	if p.Batches != 1 || p.Rows != 10 {
		t.Fatalf("progress = %d batches, %d rows; want 1 and 10", p.Batches, p.Rows)
	}
	if got := countWhere(t, db, table, "id <= 10 AND n = 1"); got != 10 {
		t.Fatalf("first batch: %d rows updated, want 10", got)
	}
	if got := countWhere(t, db, table, "id > 10 AND n <> 0"); got != 0 {
		t.Fatalf("failed batch left %d changed rows, want 0", got)
	}
}

func TestAttributeBatchRejectsBadConfig(t *testing.T) {
	db := openIntegrationDatabase(t)
	ctx := context.Background()
	m := New(db)
	table, prefix := newBatchTable(t, db, 5)
	good := `UPDATE "` + table + `" SET n = n + 1 WHERE {{batch_filter}}`

	cases := []struct {
		name, statement, table, column string
	}{
		{"missing placeholder", `UPDATE "` + table + `" SET n = 1`, table, "id"},
		{"empty table", good, "", "id"},
		{"empty column", good, table, ""},
		{"qualified column", good, table, "t.id"},
		{"too many table parts", good, "a.b.c", "id"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := m.ExecuteAttributeBatch(ctx, prefix+"_k", 1, c.statement, c.table, c.column, sql.NullString{}, 10)
			if !errors.Is(err, ErrInvalidBatchConfig) {
				t.Fatalf("error = %v, want ErrInvalidBatchConfig", err)
			}
		})
	}

	t.Run("injection in names stays inside the quotes", func(t *testing.T) {
		// The name is quoted as one identifier, so it is an unknown table, not SQL.
		_, err := m.ExecuteAttributeBatch(ctx, prefix+"_l", 1, good,
			`x"; DROP TABLE "`+table+`"; --`, "id", sql.NullString{}, 10)
		if err == nil {
			t.Fatal("expected an error for an unknown table")
		}
		if got := countWhere(t, db, table, "TRUE"); got != 5 {
			t.Fatalf("table has %d rows, want 5 (it must not be dropped or changed)", got)
		}
	})
}
