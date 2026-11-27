package observability

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const RequestIDHeader = "X-Request-ID"

type requestIDContextKey struct{}

func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDContextKey{}, id)
}

func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDContextKey{}).(string)
	return id
}

func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var idBytes [16]byte
		if _, err := rand.Read(idBytes[:]); err != nil {
			slog.ErrorContext(r.Context(), "http.request_id.generation_failed", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		requestID := hex.EncodeToString(idBytes[:])
		w.Header().Set(RequestIDHeader, requestID)
		ctx := WithRequestID(r.Context(), requestID)
		trace.SpanFromContext(ctx).SetAttributes(attribute.String("request.id", requestID))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
