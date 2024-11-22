package tracelog

import (
	"context"
	"encoding/json"
	"reflect"
	"time"

	"github.com/Andrew-M-C/trpc-go-utils/log"
	"github.com/Andrew-M-C/trpc-go-utils/tracelog/tracing"
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
		ctx = EnsureTraceID(ctx)
	} else {
		ctx = WithTraceIDStack(ctx, stack)
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

	logger := log.New(ctx).
		Str("caller", caller).
		Stringer("elapse", ela).
		Stringer("http_req", ToJSON(httpReq)).
		Stringer("server_metadata", ToJSON(metadata)).
		Stringer("req", ToJSON(req)).
		Stringer("req_type", reflect.TypeOf(req)).
		Stringer("rsp", ToJSON(rsp)).
		Stringer("rsp_type", reflect.TypeOf(rsp))

	if err != nil {
		logger.Err(err).Warn()
	} else {
		logger.Debug()
	}
	return
}

func clientFilter(ctx context.Context, req, rsp any, next filter.ClientHandleFunc) (err error) {
	ctx = tracing.EnsureTraceID(ctx)

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

	logger := log.New(ctx).
		Str("callee", callee).
		Stringer("elapse", ela).
		Stringer("http_req", ToJSON(httpReq)).
		Stringer("server_metadata", ToJSON(metadata)).
		Stringer("req", ToJSON(req)).
		Stringer("req_type", reflect.TypeOf(req)).
		Stringer("rsp", ToJSON(rsp)).
		Stringer("rsp_type", reflect.TypeOf(rsp))

	if err != nil {
		logger.Err(err).Warn()
	} else {
		logger.Debug()
	}
	return err
}
