// Package tracelog 提供带 tracing 功能的日志功能以及一些工具
package tracelog

import (
	"context"

	"github.com/Andrew-M-C/trpc-go-utils/tracelog/tracing"
)

// WithTraceID 更新 trace ID
//
// Deprecated: 请改使用 tracing 包
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return tracing.WithTraceID(ctx, traceID)
}

// WithTraceIDStack 完全替换整个 trace ID 栈
//
// Deprecated: 请改使用 tracing 包
func WithTraceIDStack(ctx context.Context, traceIDStack []string) context.Context {
	return tracing.WithTraceIDStack(ctx, traceIDStack)
}

// TraceID 从 context 中读取 trace ID
//
// Deprecated: 请改使用 tracing 包
func TraceID(ctx context.Context) string {
	return tracing.TraceID(ctx)
}

// TraceIDStack 从 context 中读取历史 trace ID 栈
//
// Deprecated: 请改使用 tracing 包
func TraceIDStack(ctx context.Context) []string {
	return tracing.TraceIDStack(ctx)
}

// EnsureTraceID 确保 context 中有一个 trace ID, 协程不安全
//
// Deprecated: 请改使用 tracing 包
func EnsureTraceID(ctx context.Context) context.Context {
	return tracing.EnsureTraceID(ctx)
}
