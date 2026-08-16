package log

import (
	"context"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"
	"trpc.group/trpc-go/trpc-go/codec"
	thttp "trpc.group/trpc-go/trpc-go/http"
)

var (
	tracePropagator = propagation.TraceContext{}
	tracer          = otel.Tracer("github.com/Andrew-M-C/trpc-go-utils/log")
)

func startServerSpanFromHTTP(ctx context.Context) (context.Context, oteltrace.Span) {
	extracted, ok := extractHTTPRemoteContext(ctx)
	if !ok {
		return ctx, nil
	}

	name := codec.Message(ctx).ServerRPCName()
	if name == "" {
		name = "server"
	}
	ctx, span := tracer.Start(extracted, name, oteltrace.WithSpanKind(oteltrace.SpanKindServer))
	return withSpanIDs(ctx, span), span
}

func startClientSpan(ctx context.Context) (context.Context, oteltrace.Span) {
	name := codec.Message(ctx).ClientRPCName()
	if name == "" {
		name = "client"
	}
	ctx, span := tracer.Start(ctx, name, oteltrace.WithSpanKind(oteltrace.SpanKindClient))
	return withSpanIDs(ctx, span), span
}

func withSpanIDs(ctx context.Context, span oteltrace.Span) context.Context {
	if span == nil {
		return ctx
	}
	sc := span.SpanContext()
	if !sc.IsValid() {
		return ctx
	}
	l := GetLogger(ctx)
	if sc.HasTraceID() {
		l = l.With("trace_id", sc.TraceID().String())
	}
	if sc.HasSpanID() {
		l = l.With("span_id", sc.SpanID().String())
	}
	return WithLogger(ctx, l)
}

func extractHTTPRemoteContext(ctx context.Context) (context.Context, bool) {
	r := thttp.Request(ctx)
	if r == nil || r.Header == nil {
		return ctx, false
	}
	if r.Header.Get("traceparent") == "" {
		return ctx, false
	}

	extracted := tracePropagator.Extract(ctx, propagation.HeaderCarrier(r.Header))
	sc := oteltrace.SpanContextFromContext(extracted)
	if !sc.IsValid() {
		return ctx, false
	}
	return extracted, true
}

func injectClientTraceHeaders(ctx context.Context) {
	injectedHTTP := injectHTTPClientTraceHeaders(ctx)
	if !injectedHTTP {
		injectClientMetaDataTraceHeaders(ctx)
	}
}

func injectHTTPClientTraceHeaders(ctx context.Context) bool {
	head, ok := codec.Message(ctx).ClientReqHead().(*thttp.ClientReqHeader)
	if !ok || head == nil {
		return false
	}
	if head.Header == nil {
		head.Header = make(http.Header)
	}
	tracePropagator.Inject(ctx, propagation.HeaderCarrier(head.Header))
	return true
}

func injectClientMetaDataTraceHeaders(ctx context.Context) {
	msg := codec.Message(ctx)
	carrier := propagation.MapCarrier{}
	tracePropagator.Inject(ctx, carrier)

	traceparent := carrier.Get("traceparent")
	if traceparent == "" {
		return
	}

	md := msg.ClientMetaData()
	if md == nil {
		md = codec.MetaData{}
	}
	md["traceparent"] = []byte(traceparent)
	if ts := carrier.Get("tracestate"); ts != "" {
		md["tracestate"] = []byte(ts)
	}
	msg.WithClientMetaData(md)
}
