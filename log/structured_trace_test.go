package log

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func TestMain(m *testing.M) {
	tp := sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.AlwaysSample()))
	otel.SetTracerProvider(tp)
	SetLevel("info")
	os.Exit(m.Run())
}

func startSpan(ctx context.Context, name string) (context.Context, oteltrace.Span) {
	return otel.Tracer("github.com/Andrew-M-C/trpc-go-utils/log_test").Start(ctx, name)
}

func formatLogEntry(ctx context.Context, level string, l *Logger) string {
	return logStringer{ctx: ctx, level: level, logger: l}.String()
}

func parseLogFields(t *testing.T, s string) map[string]any {
	t.Helper()
	raw := s
	if idx := strings.Index(s, "\t"); idx >= 0 {
		raw = s[idx+1:]
	}
	var fields map[string]any
	if err := json.Unmarshal([]byte(raw), &fields); err != nil {
		t.Fatalf("parse log json: %v, raw=%q", err, raw)
	}
	return fields
}

func TestFormatLogEntryWithoutTrace(t *testing.T) {
	out := formatLogEntry(context.Background(), "WARN", New().Text("no trace"))
	fields := parseLogFields(t, out)

	if _, ok := fields["trace_id"]; ok {
		t.Fatalf("unexpected trace_id in log fields: %v", fields)
	}
	if _, ok := fields["span_id"]; ok {
		t.Fatalf("unexpected span_id in log fields: %v", fields)
	}
}

func TestFormatLogEntryWithTrace(t *testing.T) {
	ctx, span := startSpan(context.Background(), "test-format-log-entry")
	defer span.End()

	sc := oteltrace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		t.Fatal("expected valid span context in test setup")
	}

	New().Text("这里应该带有 trace").InfoContext(ctx)

	out := formatLogEntry(ctx, "WARN", New().Text("with trace"))
	fields := parseLogFields(t, out)

	traceID, ok := fields["trace_id"].(string)
	if !ok || traceID == "" {
		t.Fatalf("expected trace_id in log fields: %v", fields)
	}
	spanID, ok := fields["span_id"].(string)
	if !ok || spanID == "" {
		t.Fatalf("expected span_id in log fields: %v", fields)
	}
	if traceID != sc.TraceID().String() {
		t.Fatalf("trace_id = %q, want %q", traceID, sc.TraceID().String())
	}
	if spanID != sc.SpanID().String() {
		t.Fatalf("span_id = %q, want %q", spanID, sc.SpanID().String())
	}
}

func TestWarnContextWithTrace(t *testing.T) {
	ctx, span := startSpan(context.Background(), "test-warn-context")
	defer span.End()

	sc := oteltrace.SpanContextFromContext(ctx)
	WarnContext(ctx, New().Text("request handled"))

	out := formatLogEntry(ctx, "WARN", New().Text("request handled"))
	fields := parseLogFields(t, out)
	if fields["trace_id"] != sc.TraceID().String() {
		t.Fatalf("trace_id = %v, want %q", fields["trace_id"], sc.TraceID().String())
	}
	if fields["span_id"] != sc.SpanID().String() {
		t.Fatalf("span_id = %v, want %q", fields["span_id"], sc.SpanID().String())
	}
}

func TestChildSpanKeepsSameTraceID(t *testing.T) {
	rootCtx, rootSpan := startSpan(context.Background(), "root")
	defer rootSpan.End()

	childCtx, childSpan := startSpan(rootCtx, "child")
	defer childSpan.End()

	rootSC := oteltrace.SpanContextFromContext(rootCtx)
	childSC := oteltrace.SpanContextFromContext(childCtx)
	if rootSC.TraceID() != childSC.TraceID() {
		t.Fatalf("trace ids differ: root=%s child=%s", rootSC.TraceID(), childSC.TraceID())
	}
	if rootSC.SpanID() == childSC.SpanID() {
		t.Fatalf("expected different span ids for parent and child")
	}

	out := formatLogEntry(childCtx, "INFO", New().Text("child span"))
	fields := parseLogFields(t, out)
	if fields["trace_id"] != childSC.TraceID().String() {
		t.Fatalf("trace_id = %v, want %q", fields["trace_id"], childSC.TraceID().String())
	}
	if fields["span_id"] != childSC.SpanID().String() {
		t.Fatalf("span_id = %v, want %q", fields["span_id"], childSC.SpanID().String())
	}
}
