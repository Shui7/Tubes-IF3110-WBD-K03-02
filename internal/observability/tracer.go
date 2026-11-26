package observability

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// TracingConfig configures the trace provider. All fields are optional except ServiceName.
type TracingConfig struct {
	ServiceName string // "app" or "worker": used as service.name

	// Exporter: "console" (default), "otlp" (to an OpenTelemetry Collector), or "none".
	// Defaults to the OTEL_EXPORTER env var.
	Exporter string

	// Console: destination for span output. Defaults to os.Stderr so that stdout
	// only contains JSON logs. One span = one JSON line.
	ConsoleOut io.Writer

	// OTLP (gRPC). Empty = use the standard OTEL_EXPORTER_OTLP_ENDPOINT env var.
	OTLPEndpoint string
	// Insecure=true for a Collector without TLS (local compose).
	Insecure bool
}

// SetupTracing installs the global TracerProvider and the W3C TraceContext
// propagator. Call shutdown when the process stops so the last spans are flushed:
//
//	shutdown, err := observability.SetupTracing(ctx, observability.TracingConfig{ServiceName: "app"})
//	if err != nil { ... }
//	defer shutdown(context.Background())
func SetupTracing(ctx context.Context, cfg TracingConfig) (shutdown func(context.Context) error, err error) {
	if cfg.ServiceName == "" {
		return nil, errors.New("observability: ServiceName is required")
	}
	exporterName := strings.ToLower(strings.TrimSpace(cfg.Exporter))
	if exporterName == "" {
		exporterName = strings.ToLower(strings.TrimSpace(os.Getenv("OTEL_EXPORTER")))
	}
	if exporterName == "" {
		exporterName = "console"
	}

	// The propagator is installed regardless of the exporter: it is needed to
	// carry trace context through the queue even when the exporter is "none".
	otel.SetTextMapPropagator(propagation.TraceContext{})

	res, err := resource.Merge(
		resource.Default(),
		resource.NewSchemaless(attribute.String("service.name", cfg.ServiceName)),
	)
	if err != nil {
		return nil, fmt.Errorf("observability: build resource: %w", err)
	}

	opts := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.AlwaysSample())),
	}

	switch exporterName {
	case "console":
		out := cfg.ConsoleOut
		if out == nil {
			out = os.Stderr
		}
		exp, err := stdouttrace.New(stdouttrace.WithWriter(out))
		if err != nil {
			return nil, fmt.Errorf("observability: console exporter: %w", err)
		}
		// Syncer: spans are printed immediately and are not lost if the process dies abruptly.
		opts = append(opts, sdktrace.WithSyncer(exp))
	case "otlp":
		clientOpts := []otlptracegrpc.Option{}
		if cfg.OTLPEndpoint != "" {
			clientOpts = append(clientOpts, otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint))
		}
		if cfg.Insecure {
			clientOpts = append(clientOpts, otlptracegrpc.WithInsecure())
		}
		exp, err := otlptracegrpc.New(ctx, clientOpts...)
		if err != nil {
			return nil, fmt.Errorf("observability: otlp exporter: %w", err)
		}
		opts = append(opts, sdktrace.WithBatcher(exp))
	case "none":
		// No exporter: spans are still created (trace_id still appears in logs).
	default:
		return nil, fmt.Errorf("observability: unknown OTEL_EXPORTER %q (use console|otlp|none)", exporterName)
	}

	tp := sdktrace.NewTracerProvider(opts...)
	otel.SetTracerProvider(tp)
	return tp.Shutdown, nil
}

// Tracer returns a tracer for a component, e.g. Tracer("worker").
func Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}

// InjectTraceparent converts the active span in ctx into a W3C `traceparent`
// string to be stored in the job column (migration_jobs.traceparent).
// Returns "" if there is no active span.
func InjectTraceparent(ctx context.Context) string {
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	return carrier.Get("traceparent")
}

// ContextWithTraceparent returns a derived ctx that carries the trace context
// from a `traceparent` string (the result of InjectTraceparent). The worker uses
// it before starting the job.process span so that the span becomes a child of
// the API request span.
func ContextWithTraceparent(ctx context.Context, traceparent string) context.Context {
	if traceparent == "" {
		return ctx
	}
	carrier := propagation.MapCarrier{"traceparent": traceparent}
	return otel.GetTextMapPropagator().Extract(ctx, carrier)
}

// RecordError marks a span as failed: it records the (redacted) error and sets the Error status.
func RecordError(span trace.Span, err error) {
	if err == nil {
		return
	}
	msg := Redact(err.Error())
	span.RecordError(errors.New(msg))
	span.SetStatus(codes.Error, msg)
}
