package tracelog

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/filter"
)

const (
	// tracelog 的 filter 名称
	FilterName = "tracelog"
	// TraceIDMetadataKey 定义用于传递 trace ID 的 trpc metadata 字段
	TraceIDStackMetadataKey = "trace_id_stack"
)

func init() {
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

	return next(ctx, req)
}

func clientFilter(ctx context.Context, req, rsp any, next filter.ClientHandleFunc) error {
	// ctx = EnsureTraceID(ctx)
	if stack := TraceIDStack(ctx); len(stack) > 0 {
		b, _ := json.Marshal(stack)
		trpc.SetMetaData(ctx, TraceIDStackMetadataKey, b)
	} else {
		b, _ := json.Marshal([]string{uuid.NewString()})
		trpc.SetMetaData(ctx, TraceIDStackMetadataKey, b)
	}
	return next(ctx, req, rsp)
}
