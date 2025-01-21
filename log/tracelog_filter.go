package log

import (
	"context"
	"encoding/json"
	"reflect"
	"time"

	"github.com/Andrew-M-C/go.util/log/trace"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/filter"
	thttp "trpc.group/trpc-go/trpc-go/http"
)

const (
	// tracelog 的 filter 名称
	FilterName = "tracelog"
	// TraceIDMetadataKey 定义用于传递 trace ID 的 trpc metadata 字段
	TraceIDStackMetadataKey = "trace_id_stack"
)

// RegisterTraceLogFilter 注册 tracelog filter。请在 trpc.NewServer 之前调用。
func RegisterTraceLogFilter() {
	filter.Register(FilterName, serverFilter, clientFilter)
}

func serverFilter(ctx context.Context, req any, next filter.ServerHandleFunc) (rsp any, err error) {
	ctx, msg := codec.EnsureMessage(ctx)
	var stack []string
	if b := msg.ServerMetaData()[TraceIDStackMetadataKey]; len(b) > 0 {
		_ = json.Unmarshal(b, &stack)
	}
	if len(stack) == 0 {
		ctx = trace.EnsureTraceID(ctx)
	} else {
		ctx = trace.WithTraceIDStack(ctx, stack)
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
		With("http_req", ToJSON(httpReq)).
		With("server_metadata", ToJSON(metadata)).
		With("req", ToJSON(req)).
		With("req_type", reflect.TypeOf(req)).
		With("rsp", ToJSON(rsp)).
		With("rsp_type", reflect.TypeOf(rsp))

	if err != nil {
		logger.Err(err).WarnContext(ctx)
	} else {
		logger.DebugContext(ctx)
	}
	return
}

func clientFilter(ctx context.Context, req, rsp any, next filter.ClientHandleFunc) (err error) {
	ctx = trace.EnsureTraceID(ctx)

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
		With("http_req", ToJSON(httpReq)).
		With("server_metadata", ToJSON(metadata)).
		With("req", ToJSON(req)).
		With("req_type", reflect.TypeOf(req)).
		With("rsp", ToJSON(rsp)).
		With("rsp_type", reflect.TypeOf(rsp))

	if err != nil {
		logger.Err(err).WarnContext(ctx)
	} else {
		logger.DebugContext(ctx)
	}
	return err
}
