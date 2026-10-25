package dbmanager

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

func openIntegrationDatabase(t *testing.T) *sql.DB {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestMigrationSuccessAndRollback(t *testing.T) {
	db := openIntegrationDatabase(t)
	ctx := context.Background()
	manager := New(db)

	if err := manager.EnsureMigrationTable(ctx); err != nil {
		t.Fatal(err)
	}
	version, err := manager.NextMigrationVersion(ctx)
	if err != nil {
		t.Fatal(err)
	}

	table := fmt.Sprintf("integration_%d", time.Now().UnixNano())
	retryTable := table + "_retry"
	t.Cleanup(func() {
		_, _ = db.Exec(`DROP TABLE IF EXISTS "` + table + `"`)
		_, _ = db.Exec(`DROP TABLE IF EXISTS "` + retryTable + `"`)
		_, _ = db.Exec(`DELETE FROM db_platform_migrations WHERE version >= $1`, version)
	})

	if err := manager.ExecuteMigration(ctx, version,
		`CREATE TABLE "`+table+`" (id INTEGER PRIMARY KEY)`, "integration"); err != nil {
		t.Fatalf("successful migration: %v", err)
	}

	var exists bool
	if err := db.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name=$1)`, table,
	).Scan(&exists); err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatal("successful migration did not create its table")
	}

	failedVersion := version + 1
	rollbackTable := table + "_rollback"
	if err := manager.ExecuteMigration(ctx, failedVersion,
		`CREATE TABLE "`+rollbackTable+`" (id INTEGER); INSERT INTO "`+rollbackTable+`" (missing_column) VALUES (1)`,
		"integration"); err == nil {
		t.Fatal("invalid migration unexpectedly succeeded")
	}

	if err := db.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name=$1)`, rollbackTable,
	).Scan(&exists); err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatal("failed migration left an earlier statement committed")
	}

	var status string
	if err := db.QueryRowContext(ctx,
		`SELECT status FROM db_platform_migrations WHERE version=$1`, failedVersion,
	).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "failed" {
		t.Fatalf("failed migration status = %q, want failed", status)
	}

	if err := manager.ExecuteMigration(ctx, failedVersion,
		`CREATE TABLE "`+retryTable+`" (id INTEGER PRIMARY KEY)`, "integration"); err != nil {
		t.Fatalf("retry failed migration version: %v", err)
	}
}

func TestMigrationRejectsVersionGap(t *testing.T) {
	db := openIntegrationDatabase(t)
	manager := New(db)
	ctx := context.Background()
	if err := manager.EnsureMigrationTable(ctx); err != nil {
		t.Fatal(err)
	}
	version, err := manager.NextMigrationVersion(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.ExecuteMigration(ctx, version+1, "SELECT 1", "integration"); err == nil {
		t.Fatal("version gap unexpectedly succeeded")
	}
}

func TestMigrationDraftLifecycle(t *testing.T) {
	db := openIntegrationDatabase(t)
	manager := New(db)
	ctx := context.Background()
	draft, err := manager.CreateMigrationDraft(ctx, "add users", "CREATE TABLE users (id INTEGER)", "integration")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = manager.DeleteMigrationDraft(ctx, draft.ID)
	})

	updated, err := manager.UpdateMigrationDraft(ctx, draft.ID, "add accounts", "CREATE TABLE accounts (id INTEGER)")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "add accounts" || updated.SQL != "CREATE TABLE accounts (id INTEGER)" {
		t.Fatalf("updated draft = %+v", updated)
	}

	got, err := manager.GetMigrationDraft(ctx, draft.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != draft.ID {
		t.Fatalf("got draft ID = %d, want %d", got.ID, draft.ID)
	}
}

func TestDatabaseViewer(t *testing.T) {
	db := openIntegrationDatabase(t)
	manager := New(db)
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	parent := "parent_" + suffix
	child := "child_" + suffix
	t.Cleanup(func() {
		_, _ = db.Exec(`DROP TABLE IF EXISTS "` + child + `"`)
		_, _ = db.Exec(`DROP TABLE IF EXISTS "` + parent + `"`)
	})

	_, err := db.Exec(`
		CREATE TABLE "` + parent + `" (
			tenant_id INTEGER NOT NULL,
			parent_id INTEGER NOT NULL,
			PRIMARY KEY (tenant_id, parent_id)
		);
		CREATE TABLE "` + child + `" (
			id SERIAL PRIMARY KEY,
			tenant_id INTEGER NOT NULL,
			parent_id INTEGER NOT NULL,
			CONSTRAINT child_parent_fk FOREIGN KEY (tenant_id, parent_id)
				REFERENCES "` + parent + `" (tenant_id, parent_id)
		);
		INSERT INTO "` + parent + `" VALUES (1, 10), (1, 20), (1, 30);
		INSERT INTO "` + child + `" (tenant_id, parent_id) VALUES (1, 10);`)
	if err != nil {
		t.Fatal(err)
	}

	details, err := manager.ListTableDetails(ctx, "public")
	if err != nil {
		t.Fatal(err)
	}
	var foundChild bool
	for _, detail := range details {
		if detail.Name == child {
			foundChild = true
			if len(detail.Columns) != 4 || len(detail.Constraints) < 2 {
				t.Fatalf("child details = %+v", detail)
			}
		}
	}
	if !foundChild {
		t.Fatalf("table %q was not listed", child)
	}

	relationships, err := manager.ListRelationships(ctx, "public")
	if err != nil {
		t.Fatal(err)
	}
	var relationshipColumns int
	for _, relationship := range relationships {
		if relationship.FromTable == child && relationship.ToTable == parent {
			relationshipColumns++
		}
	}
	if relationshipColumns != 2 {
		t.Fatalf("composite relationship columns = %d, want 2", relationshipColumns)
	}

	rows, err := manager.PreviewRowsPage(ctx, "public", parent, 2, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("first page length = %d, want 2", len(rows))
	}
	rows, err = manager.PreviewRowsPage(ctx, "public", parent, 2, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("second page length = %d, want 1", len(rows))
	}

	if _, err := manager.PreviewRows(ctx, "public", "missing_"+suffix, 10); err == nil {
		t.Fatal("missing table unexpectedly returned rows")
	}
}

func TestMissingMigrationHistoryIsNotFatal(t *testing.T) {
	db := openIntegrationDatabase(t)
	manager := New(db)
	ctx := context.Background()
	_, _ = db.Exec(`DROP TABLE IF EXISTS db_platform_migrations`)
	t.Cleanup(func() { _ = manager.EnsureMigrationTable(ctx) })

	if _, err := manager.ListMigrations(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.GetMigration(ctx, 1); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("GetMigration error = %v, want sql.ErrNoRows", err)
	}
}
