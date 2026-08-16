package log

import (
	"context"
	"reflect"
	"time"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/filter"
	thttp "trpc.group/trpc-go/trpc-go/http"
)

const (
	// tracelog 的 filter 名称
	FilterName = "tracelog"
)

// RegisterTraceLogFilter 注册 tracelog filter。请在 trpc.NewServer 之前调用。
func RegisterTraceLogFilter() {
	filter.Register(FilterName, serverFilter, clientFilter)
}

func serverFilter(ctx context.Context, req any, next filter.ServerHandleFunc) (rsp any, err error) {
	ctx, _ = codec.EnsureMessage(ctx)
	ctx, span := startServerSpanFromHTTP(ctx)
	if span != nil {
		defer span.End()
	}

	caller := func() string {
		if addr := codec.Message(ctx).RemoteAddr(); addr != nil {
			return addr.String()
		}
		return "unknown"
	}()
	metadata := codec.Message(ctx).ServerMetaData()
	httpReq := thttp.Request(ctx)

	start := time.Now()
	rsp, err = next(ctx, req)
	ela := time.Since(start)

	logger := New().
		With("caller", caller).
		With("elapse", ela).
		WithJSON("http_req", httpReq).
		WithJSON("server_metadata", metadata).
		WithJSON("req", req).
		With("req_type", reflect.TypeOf(req)).
		WithJSON("rsp", rsp).
		With("rsp_type", reflect.TypeOf(rsp))

	if err != nil {
		logger.Text("server 返回失败").Err(err).WarnContext(ctx)
	} else {
		logger.Text("server 返回成功").DebugContext(ctx)
	}
	return
}

func clientFilter(ctx context.Context, req, rsp any, next filter.ClientHandleFunc) (err error) {
	ctx, _ = codec.EnsureMessage(ctx)
	ctx, span := startClientSpan(ctx)
	defer span.End()
	injectClientTraceHeaders(ctx)

	callee := func() string {
		if addr := codec.Message(ctx).RemoteAddr(); addr != nil {
			return addr.String()
		}
		return "unknown"
	}()
	metadata := codec.Message(ctx).ServerMetaData()
	httpReq := thttp.Request(ctx)

	start := time.Now()
	err = next(ctx, req, rsp)
	ela := time.Since(start)

	logger := New().
		With("callee", callee).
		With("elapse", ela).
		WithJSON("http_req", httpReq).
		WithJSON("server_metadata", metadata).
		WithJSON("req", req).
		With("req_type", reflect.TypeOf(req)).
		WithJSON("rsp", rsp).
		With("rsp_type", reflect.TypeOf(rsp))

	if err != nil {
		logger.Text("client 返回失败").Err(err).WarnContext(ctx)
	} else {
		logger.Text("client 返回成功").DebugContext(ctx)
	}
	return err
}
