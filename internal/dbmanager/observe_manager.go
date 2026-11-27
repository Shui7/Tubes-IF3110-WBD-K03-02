package dbmanager

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"time"

	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/observability"
	"github.com/lib/pq"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type contextExecer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

type contextQueryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

type contextQueryRower interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func parameterTypes(args []any) []string {
	parameters := make([]string, len(args))
	for i, arg := range args {
		if arg == nil {
			parameters[i] = fmt.Sprintf("$%d:<nil>", i+1)
			continue
		}
		parameters[i] = fmt.Sprintf("$%d:%s", i+1, reflect.TypeOf(arg))
	}
	return parameters
}

func safeStatement(query string) string {
	var out strings.Builder
	for i := 0; i < len(query); {
		if strings.HasPrefix(query[i:], "--") {
			for i < len(query) && query[i] != '\n' {
				i++
			}
			out.WriteString("-- redacted")
			continue
		}
		if strings.HasPrefix(query[i:], "/*") {
			i += 2
			depth := 1
			for i < len(query) && depth > 0 {
				switch {
				case strings.HasPrefix(query[i:], "/*"):
					depth++
					i += 2
				case strings.HasPrefix(query[i:], "*/"):
					depth--
					i += 2
				default:
					i++
				}
			}
			out.WriteString("/* redacted */")
			continue
		}
		if query[i] == '\'' {
			i++
			for i < len(query) {
				if query[i] == '\'' {
					i++
					if i < len(query) && query[i] == '\'' {
						i++
						continue
					}
					break
				}
				if query[i] == '\\' && i+1 < len(query) {
					i += 2
					continue
				}
				i++
			}
			out.WriteString("'?'")
			continue
		}
		if query[i] == '$' {
			end := i + 1
			for end < len(query) && (query[end] == '_' || query[end] >= 'a' && query[end] <= 'z' ||
				query[end] >= 'A' && query[end] <= 'Z' || query[end] >= '0' && query[end] <= '9') {
				end++
			}
			if end < len(query) && query[end] == '$' {
				delimiter := query[i : end+1]
				if closeAt := strings.Index(query[end+1:], delimiter); closeAt >= 0 {
					i = end + 1 + closeAt + len(delimiter)
					out.WriteString(delimiter + "?" + delimiter)
					continue
				}
			}
		}
		out.WriteByte(query[i])
		i++
	}
	return observability.Redact(out.String())
}

func startQuery(ctx context.Context, operation, query string, args []any) (context.Context, trace.Span, time.Time, []string) {
	parameters := parameterTypes(args)
	attributes := []attribute.KeyValue{
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", operation),
		attribute.String("db.statement", safeStatement(query)),
		attribute.Int("db.parameters.count", len(args)),
	}
	if requestID := observability.RequestIDFromContext(ctx); requestID != "" {
		attributes = append(attributes, attribute.String("request.id", requestID))
	}
	ctx, span := observability.Tracer("dbmanager").Start(ctx, "db.query",
		trace.WithAttributes(attributes...))
	return ctx, span, time.Now(), parameters
}

func finishQuery(ctx context.Context, span trace.Span, started time.Time, operation, query string, parameters []string, rows int64, err error) {
	duration := time.Since(started)
	span.SetAttributes(
		attribute.Int64("db.rows_affected", rows),
		attribute.Int64("db.duration_ms", duration.Milliseconds()),
	)
	attrs := []any{
		"operation", operation,
		"query", safeStatement(query),
		"parameters", parameters,
		"duration_ms", duration.Milliseconds(),
		"rows", rows,
		"ok", err == nil,
	}
	if state := databaseSQLState(err); state != "" {
		span.SetAttributes(attribute.String("db.sqlstate", state))
		attrs = append(attrs, "sqlstate", state)
	}
	if err != nil {
		span.SetStatus(codes.Error, "database query failed")
		attrs = append(attrs, "error", "database query failed")
		slog.ErrorContext(ctx, "db.query.failed", attrs...)
	} else {
		span.SetStatus(codes.Ok, "")
		slog.InfoContext(ctx, "db.query.completed", attrs...)
	}
	span.End()
}

func execObserved(ctx context.Context, execer contextExecer, operation, query string, args ...any) (sql.Result, error) {
	ctx, span, started, parameters := startQuery(ctx, operation, query, args)
	result, err := execer.ExecContext(ctx, query, args...)
	var affected int64
	if err == nil {
		affected, err = result.RowsAffected()
	}
	finishQuery(ctx, span, started, operation, query, parameters, affected, err)
	return result, err
}

func queryObserved(ctx context.Context, queryer contextQueryer, operation, query string, args ...any) (*sql.Rows, func(int64, error), error) {
	ctx, span, started, parameters := startQuery(ctx, operation, query, args)
	rows, err := queryer.QueryContext(ctx, query, args...)
	if err != nil {
		finishQuery(ctx, span, started, operation, query, parameters, 0, err)
		return nil, nil, err
	}
	return rows, func(count int64, iterationErr error) {
		finishQuery(ctx, span, started, operation, query, parameters, count, iterationErr)
	}, nil
}

func queryRowObserved(ctx context.Context, queryer contextQueryRower, operation, query string, args ...any) (*sql.Row, func(int64, error)) {
	ctx, span, started, parameters := startQuery(ctx, operation, query, args)
	row := queryer.QueryRowContext(ctx, query, args...)
	finish := func(rows int64, err error) {
		finishQuery(ctx, span, started, operation, query, parameters, rows, err)
	}
	return row, finish
}

func commitObserved(ctx context.Context, tx *sql.Tx, operation string) error {
	started := time.Now()
	err := tx.Commit()
	logTransaction(ctx, operation, "commit", started, err)
	return err
}

func rollbackObserved(ctx context.Context, tx *sql.Tx, operation string) {
	started := time.Now()
	err := tx.Rollback()
	if errors.Is(err, sql.ErrTxDone) {
		return
	}
	logTransaction(ctx, operation, "rollback", started, err)
}

func logTransaction(ctx context.Context, operation, action string, started time.Time, err error) {
	duration := time.Since(started)
	attributes := []attribute.KeyValue{
		attribute.String("db.system", "postgresql"),
		attribute.String("db.transaction.operation", operation),
		attribute.Int64("db.duration_ms", duration.Milliseconds()),
	}
	if requestID := observability.RequestIDFromContext(ctx); requestID != "" {
		attributes = append(attributes, attribute.String("request.id", requestID))
	}
	_, span := observability.Tracer("dbmanager").Start(ctx, "db.transaction."+action, trace.WithAttributes(attributes...))
	attrs := []any{"operation", operation, "duration_ms", duration.Milliseconds()}
	if state := databaseSQLState(err); state != "" {
		span.SetAttributes(attribute.String("db.sqlstate", state))
		attrs = append(attrs, "sqlstate", state)
	}
	if err != nil {
		span.SetStatus(codes.Error, "database transaction failed")
		attrs = append(attrs, "error", "database transaction failed")
		slog.ErrorContext(ctx, "db.transaction."+action+".failed", attrs...)
	} else {
		span.SetStatus(codes.Ok, "")
		slog.InfoContext(ctx, "db.transaction."+action+".completed", attrs...)
	}
	span.End()
}

func databaseSQLState(err error) string {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return string(pqErr.Code)
	}
	return ""
}

func databaseErrorSummary(err error) string {
	if state := databaseSQLState(err); state != "" {
		return "PostgreSQL statement failed (SQLSTATE " + state + ")"
	}
	return "database statement failed"
}
