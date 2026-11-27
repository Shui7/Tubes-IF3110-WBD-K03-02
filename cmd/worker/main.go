package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/jobs"
	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/managed"
	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/observability"
	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/workflow"
	_ "github.com/lib/pq"
)

// Generate workers that executes queued migration jobs.
func main() {
	// Create context for database/tracing/workers,
	// automatically cancelled when user do Ctrl+C from terminal (os.Interrupt) or recieve SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger := observability.SetupLogger(observability.LoggerConfig{Service: "worker"})
	shutdownTracing, err := observability.SetupTracing(ctx, observability.TracingConfig{
		ServiceName: "worker",
		Insecure:    true, // only used when OTEL_EXPORTER=otlp
	})
	if err != nil {
		logger.Error("tracing.setup_failed", "error", err)
		os.Exit(1)
	}
	defer func() { _ = shutdownTracing(context.Background()) }()

	if err := run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("worker.failed", "error", err)
		_ = shutdownTracing(context.Background())
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return errors.New("DATABASE_URL is empty")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("open platform database: %w", err)
	}
	defer db.Close()

	// Check platform database connection. May still be starting, retry for a while.
	for i := 0; ; i++ {
		if err = db.PingContext(ctx); err == nil {
			break
		}
		if i >= 30 {
			return fmt.Errorf("ping platform database: %w", err)
		}
		slog.WarnContext(ctx, "platformdb.waiting", "error", err)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	if err := jobs.EnsureSchema(ctx, db); err != nil {
		return err
	}

	registry := managed.NewRegistry(db)
	defer registry.Close()

	hostname, _ := os.Hostname()
	worker := jobs.NewWorker(
		jobs.NewStore(db),
		&jobs.ManagerExecutor{Registry: registry},
		workflow.ExecutionValidator{PlatformDB: db, Registry: registry},
		jobs.WorkerConfig{
			ID:          fmt.Sprintf("%s-%d", hostname, os.Getpid()),
			Concurrency: envInt("WORKER_CONCURRENCY", 4),
		},
	)
	return worker.Run(ctx)
}

func envInt(key string, def int) int {
	if v, err := strconv.Atoi(os.Getenv(key)); err == nil && v > 0 {
		return v
	}
	return def
}
