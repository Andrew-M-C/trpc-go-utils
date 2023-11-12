// Package tracelog 提供带 tracing 功能的日志功能以及一些工具
package tracelog

import (
	"context"

	"github.com/google/uuid"
	"golang.org/x/exp/slices"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/log"
)

// 保存 ctx 中的 trace ID 字段
type traceIDKey struct{}

type traceIDStackyValue []string

func (t traceIDStackyValue) id() string {
	if len(t) == 0 {
		return ""
	}
	return t[len(t)-1]
}

// WithTraceID 更新 trace ID
func WithTraceID(ctx context.Context, traceID string) context.Context {
	if traceID == "" {
		return ctx
	}
	stack := traceIDStack(ctx)
	if stack.id() == traceID {
		return ctx
	}
	stack = slices.Clone(stack)
	stack = append(stack, traceID)
	return cloneCtxAndGenerateLog(ctx, stack)
}

// WithTraceIDStack 完全替换整个 trace ID 栈
func WithTraceIDStack(ctx context.Context, traceIDStack []string) context.Context {
	if len(traceIDStack) == 0 {
		return ctx
	}
	stack := traceIDStackyValue(slices.Clone(traceIDStack))
	return cloneCtxAndGenerateLog(ctx, stack)
}

func cloneCtxAndGenerateLog(ctx context.Context, stack traceIDStackyValue) context.Context {
	ctx, _ = codec.WithCloneMessage(ctx)

	fields := []log.Field{{
		Key:   "trace_id",
		Value: stack.id(),
	}}
	if le := len(stack); le > 1 {
		fields = append(fields, log.Field{
			Key:   "trace_id_stack",
			Value: stack[:le-1],
		})
	}
	// TODO: dyeing
	// if msg.Dyeing() {
	// 	fields = append(fields, log.Field{
	// 		Key:   "dyeing",
	// 		Value: true,
	// 	})
	// }
	logger := log.GetDefaultLogger().With(fields...)
	ctx, msg := codec.EnsureMessage(ctx)
	msg.WithLogger(logger)

	return context.WithValue(ctx, traceIDKey{}, stack)
}

func traceIDStack(ctx context.Context) traceIDStackyValue {
	v := ctx.Value(traceIDKey{})
	if v == nil {
		return traceIDStackyValue{}
	}
	st, _ := v.(traceIDStackyValue)
	return st
}

// TraceID 从 context 中读取 trace ID
func TraceID(ctx context.Context) string {
	v := ctx.Value(traceIDKey{})
	if v == nil {
		return ""
	}
	s, _ := v.(traceIDStackyValue)
	return s.id()
}

// TraceIDStack 从 context 中读取历史 trace ID 栈
func TraceIDStack(ctx context.Context) []string {
	v := ctx.Value(traceIDKey{})
	if v == nil {
		return nil
	}
	s, _ := v.(traceIDStackyValue)
	return slices.Clone(s)
}

// EnsureTraceID 确保 context 中有一个 trace ID, 协程不安全
func EnsureTraceID(ctx context.Context) context.Context {
	traceID := TraceID(ctx)
	if traceID == "" {
		return WithTraceID(ctx, uuid.NewString())
	}
	return ctx
}
