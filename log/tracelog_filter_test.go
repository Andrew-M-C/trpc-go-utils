package log_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Andrew-M-C/trpc-go-utils/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/filter"
	thttp "trpc.group/trpc-go/trpc-go/http"
)

func TestServerFilterStartsSpanFromHTTPTraceparent(t *testing.T) {
	sf, _ := tracelogFilters(t)

	parentCtx, parentSpan := otel.Tracer("test").Start(context.Background(), "upstream")
	defer parentSpan.End()
	parentSC := oteltrace.SpanContextFromContext(parentCtx)
	if !parentSC.IsValid() {
		t.Fatal("expected valid parent span")
	}

	headers := http.Header{}
	propagation.TraceContext{}.Inject(parentCtx, propagation.HeaderCarrier(headers))

	ctx := newHTTPServerCtx(t, "/trpc.app.svc/Hello", headers)

	var gotCtx context.Context
	_, err := sf(ctx, "req", func(ctx context.Context, req any) (any, error) {
		gotCtx = ctx
		sc := oteltrace.SpanContextFromContext(ctx)
		if !sc.IsValid() {
			t.Fatal("expected server span in ctx")
		}
		if sc.TraceID() != parentSC.TraceID() {
			t.Fatalf("trace_id = %s, want %s", sc.TraceID(), parentSC.TraceID())
		}
		if sc.SpanID() == parentSC.SpanID() {
			t.Fatal("server span should be a child, not the remote parent")
		}

		l := log.GetLogger(ctx)
		if l == nil {
			t.Fatal("expected logger with trace fields in ctx")
		}
		return "rsp", nil
	})
	if err != nil {
		t.Fatalf("server filter: %v", err)
	}
	if gotCtx == nil {
		t.Fatal("next was not called")
	}

	fields := loggerFields(t, gotCtx)
	if fields["trace_id"] != parentSC.TraceID().String() {
		t.Fatalf("logger trace_id = %v, want %s", fields["trace_id"], parentSC.TraceID())
	}
	if fields["span_id"] == "" || fields["span_id"] == parentSC.SpanID().String() {
		t.Fatalf("logger span_id = %v, want child span", fields["span_id"])
	}
}

func TestServerFilterSkipsSpanWithoutHTTPTraceparent(t *testing.T) {
	sf, _ := tracelogFilters(t)
	ctx := newHTTPServerCtx(t, "/trpc.app.svc/Hello", nil)

	_, err := sf(ctx, "req", func(ctx context.Context, req any) (any, error) {
		if oteltrace.SpanContextFromContext(ctx).IsValid() {
			t.Fatal("should not start span without traceparent")
		}
		if log.GetLogger(ctx) != nil {
			t.Fatal("should not inject logger without tracing")
		}
		return "rsp", nil
	})
	if err != nil {
		t.Fatalf("server filter: %v", err)
	}
}

func TestServerFilterSkipsSpanForNonHTTP(t *testing.T) {
	sf, _ := tracelogFilters(t)
	ctx, msg := codec.WithNewMessage(context.Background())
	msg.WithServerRPCName("/trpc.app.svc/Hello")

	_, err := sf(ctx, "req", func(ctx context.Context, req any) (any, error) {
		if oteltrace.SpanContextFromContext(ctx).IsValid() {
			t.Fatal("non-http request should not start span")
		}
		return "rsp", nil
	})
	if err != nil {
		t.Fatalf("server filter: %v", err)
	}
}

func TestServerFilterSkipsInvalidTraceparent(t *testing.T) {
	sf, _ := tracelogFilters(t)
	headers := http.Header{}
	headers.Set("traceparent", "not-a-valid-traceparent")
	ctx := newHTTPServerCtx(t, "/trpc.app.svc/Hello", headers)

	_, err := sf(ctx, "req", func(ctx context.Context, req any) (any, error) {
		if oteltrace.SpanContextFromContext(ctx).IsValid() {
			t.Fatal("invalid traceparent should not start span")
		}
		return "rsp", nil
	})
	if err != nil {
		t.Fatalf("server filter: %v", err)
	}
}

func TestClientFilterCreatesSpanAndInjectsHTTPHeaders(t *testing.T) {
	_, cf := tracelogFilters(t)

	parentCtx, parentSpan := otel.Tracer("test").Start(context.Background(), "server")
	defer parentSpan.End()
	parentSC := oteltrace.SpanContextFromContext(parentCtx)

	ctx, head := newHTTPClientCtx(t, parentCtx, "/trpc.app.svc/Call")

	var childSC oteltrace.SpanContext
	err := cf(ctx, "req", "rsp", func(ctx context.Context, req, rsp any) error {
		childSC = oteltrace.SpanContextFromContext(ctx)
		if !childSC.IsValid() {
			t.Fatal("expected client span in ctx")
		}
		if childSC.TraceID() != parentSC.TraceID() {
			t.Fatalf("trace_id = %s, want %s", childSC.TraceID(), parentSC.TraceID())
		}
		if childSC.SpanID() == parentSC.SpanID() {
			t.Fatal("client span should be a child of the parent")
		}

		fields := loggerFields(t, ctx)
		if fields["trace_id"] != childSC.TraceID().String() {
			t.Fatalf("logger trace_id = %v, want %s", fields["trace_id"], childSC.TraceID())
		}
		if fields["span_id"] != childSC.SpanID().String() {
			t.Fatalf("logger span_id = %v, want %s", fields["span_id"], childSC.SpanID())
		}
		return nil
	})
	if err != nil {
		t.Fatalf("client filter: %v", err)
	}

	traceparent := head.Header.Get("traceparent")
	if traceparent == "" {
		t.Fatal("expected traceparent on client http header")
	}
	if !strings.Contains(strings.ToLower(traceparent), parentSC.TraceID().String()) {
		t.Fatalf("traceparent %q does not contain parent trace_id %s", traceparent, parentSC.TraceID())
	}
	if !strings.Contains(strings.ToLower(traceparent), childSC.SpanID().String()) {
		t.Fatalf("traceparent %q does not contain client span_id %s", traceparent, childSC.SpanID())
	}
}

func TestClientFilterCreatesRootSpanWithoutParent(t *testing.T) {
	_, cf := tracelogFilters(t)
	ctx, head := newHTTPClientCtx(t, context.Background(), "/trpc.app.svc/Call")

	err := cf(ctx, "req", "rsp", func(ctx context.Context, req, rsp any) error {
		sc := oteltrace.SpanContextFromContext(ctx)
		if !sc.IsValid() {
			t.Fatal("expected a new root client span")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("client filter: %v", err)
	}
	if head.Header.Get("traceparent") == "" {
		t.Fatal("expected traceparent even without parent span")
	}
}

func TestClientFilterInjectsMetaDataWithoutHTTPHeader(t *testing.T) {
	_, cf := tracelogFilters(t)
	ctx, msg := codec.WithNewMessage(context.Background())
	msg.WithClientRPCName("/trpc.app.svc/Call")

	err := cf(ctx, "req", "rsp", func(ctx context.Context, req, rsp any) error {
		md := codec.Message(ctx).ClientMetaData()
		if string(md["traceparent"]) == "" {
			t.Fatal("expected traceparent in client metadata")
		}
		if _, ok := codec.Message(ctx).ClientReqHead().(*thttp.ClientReqHeader); ok {
			t.Fatal("should not invent http client req header")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("client filter: %v", err)
	}
}

func tracelogFilters(t *testing.T) (filter.ServerFilter, filter.ClientFilter) {
	t.Helper()
	log.RegisterTraceLogFilter()
	sf := filter.GetServer(log.FilterName)
	cf := filter.GetClient(log.FilterName)
	if sf == nil || cf == nil {
		t.Fatal("tracelog filter not registered")
	}
	return sf, cf
}

func newHTTPServerCtx(t *testing.T, rpcName string, headers http.Header) context.Context {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "http://example.com"+rpcName, nil)
	for k, vs := range headers {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	ctx, msg := codec.WithNewMessage(context.Background())
	msg.WithServerRPCName(rpcName)
	return thttp.WithHeader(ctx, &thttp.Header{Request: req})
}

func newHTTPClientCtx(t *testing.T, parent context.Context, rpcName string) (context.Context, *thttp.ClientReqHeader) {
	t.Helper()
	ctx, msg := codec.WithNewMessage(parent)
	msg.WithClientRPCName(rpcName)
	head := &thttp.ClientReqHeader{Header: make(http.Header)}
	msg.WithClientReqHead(head)
	return ctx, head
}

func loggerFields(t *testing.T, ctx context.Context) map[string]any {
	t.Helper()
	stub := withStubTRPCLogger(t)
	log.New().Text("probe").InfoContext(ctx)
	lines := stub.lines("info")
	if len(lines) == 0 {
		t.Fatal("expected info log")
	}
	return parseLogFields(t, lines[0])
}
