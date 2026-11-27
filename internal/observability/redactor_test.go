package observability

import (
	"context"
	"strings"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func TestRedactSensitiveValues(t *testing.T) {
	input := "password='my secret' postgres://dbuser:p455word@localhost:5432/app Cookie: session=abc"
	got := Redact(input)
	for _, secret := range []string{"my secret", "p455word", "session=abc"} {
		if strings.Contains(got, secret) {
			t.Errorf("Redact(%q) leaked %q: %q", input, secret, got)
		}
	}
	for _, expected := range []string{"password=[REDACTED]", "postgres://dbuser:[REDACTED]@", "Cookie: [REDACTED]"} {
		if !strings.Contains(got, expected) {
			t.Errorf("Redact(%q) = %q, want substring %q", input, got, expected)
		}
	}
}

func TestTraceparentRoundTrip(t *testing.T) {
	old := propagation.TextMapPropagator(otel.GetTextMapPropagator())
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() { otel.SetTextMapPropagator(old) })

	traceID, err := trace.TraceIDFromHex("0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	spanID, err := trace.SpanIDFromHex("0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	spanContext := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: traceID, SpanID: spanID, TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), spanContext)
	traceparent := InjectTraceparent(ctx)
	if traceparent == "" {
		t.Fatal("InjectTraceparent returned an empty value")
	}

	extracted := trace.SpanContextFromContext(ContextWithTraceparent(context.Background(), traceparent))
	if !extracted.IsValid() || extracted.TraceID() != traceID || !extracted.IsSampled() {
		t.Fatalf("extracted span context = %+v, want trace ID %s and sampled flag", extracted, traceID)
	}
}
