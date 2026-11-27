package observability

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"

	"go.opentelemetry.io/otel/trace"
)

type contextHandler struct {
	inner slog.Handler
}

func (h contextHandler) Enabled(ctx context.Context, l slog.Level) bool {
	return h.inner.Enabled(ctx, l)
}

func (h contextHandler) Handle(ctx context.Context, r slog.Record) error {
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		r.AddAttrs(
			slog.String("trace_id", sc.TraceID().String()),
			slog.String("span_id", sc.SpanID().String()),
		)
	}
	if id := RequestIDFromContext(ctx); id != "" {
		r.AddAttrs(slog.String("request_id", id))
	}
	return h.inner.Handle(ctx, r)
}

func (h contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return contextHandler{inner: h.inner.WithAttrs(attrs)}
}

func (h contextHandler) WithGroup(name string) slog.Handler {
	return contextHandler{inner: h.inner.WithGroup(name)}
}

type LoggerConfig struct {
	Service string
	Level   string    // debug|info|warn|error; default: env LOG_LEVEL, then info
	Output  io.Writer // default os.Stdout
}

// slog.InfoContext(ctx, msg, attrs...).
// msg: name of event with key "event"
// attrs: (key, value)
// slog.InfoContext(ctx, "job.enqueued", "job_id", id),
func NewLogger(cfg LoggerConfig) *slog.Logger {
	out := cfg.Output
	if out == nil {
		out = os.Stdout
	}
	levelName := cfg.Level
	if levelName == "" {
		levelName = os.Getenv("LOG_LEVEL")
	}

	base := slog.NewJSONHandler(out, &slog.HandlerOptions{
		Level:       parseLevel(levelName),
		ReplaceAttr: replaceAttr,
	})
	logger := slog.New(contextHandler{inner: base})
	if cfg.Service != "" {
		logger = logger.With("service", cfg.Service)
	}
	return logger
}

func SetupLogger(cfg LoggerConfig) *slog.Logger {
	logger := NewLogger(cfg)
	slog.SetDefault(logger)
	return logger
}

func parseLevel(name string) slog.Level {
	var l slog.Level
	if err := l.UnmarshalText([]byte(strings.TrimSpace(name))); err != nil {
		return slog.LevelInfo
	}
	return l
}

func ErrorAttr(err error) slog.Attr {
	if err == nil {
		return slog.Attr{}
	}
	return slog.String("error", Redact(err.Error()))
}
