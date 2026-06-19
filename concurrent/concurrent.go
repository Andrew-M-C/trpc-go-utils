// Package concurrent 定义一些并发相关的功能
package concurrent

import (
	"context"

	"github.com/Andrew-M-C/trpc-go-utils/recovery"
	"go.opentelemetry.io/otel"
	oteltrace "go.opentelemetry.io/otel/trace"
	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
)

// Detach 分离一个新的后台任务, 不等待其返回
func Detach(ctx context.Context, spanName string, task func(context.Context), recoveryOpts ...recovery.Option) {
	newCtx, _ := codec.WithCloneContextAndMessage(ctx)
	newCtx = copyContextValues(newCtx, ctx)
	newCtx = copyTracing(ctx, newCtx)

	newCtx, span := NewSpan(newCtx, spanName)
	recoveryOpts = append(recoveryOpts, recovery.WithContext(newCtx))

	go func() {
		defer span.End()
		defer recovery.CatchPanic(recoveryOpts...)
		task(newCtx)
	}()
}

// DetachAndWait 分离多个任务, 并等待所有任务完成
func DetachAndWait(ctx context.Context, tasksWithSpanNames map[string]func(context.Context) error) error {
	if len(tasksWithSpanNames) == 0 {
		return nil
	}

	handlers := make([]func() error, 0, len(tasksWithSpanNames))
	for spanName, task := range tasksWithSpanNames {
		if task == nil {
			continue
		}

		newCtx, _ := codec.WithCloneContextAndMessage(ctx)
		newCtx = copyContextValues(newCtx, ctx)
		newCtx = copyTracing(ctx, newCtx)
		newCtx, span := NewSpan(newCtx, spanName)

		handlers = append(handlers, func() error {
			defer span.End()
			defer recovery.CatchPanic(
				recovery.WithContext(newCtx), recovery.WithErrorLog(),
			)
			return task(newCtx)
		})
	}

	return trpc.GoAndWait(handlers...)
}

// NewSpan 是对 otel.Tracer.Start 的封装, 创建一个 span, 并返回新的 context。
// 返回的 span 需要 defer span.End()
func NewSpan(
	ctx context.Context, spanName string, opts ...oteltrace.SpanStartOption,
) (context.Context, oteltrace.Span) {
	tracer := otel.Tracer("github.com/Andrew-M-C/trpc-go-utils/concurrent")
	return tracer.Start(ctx, spanName, opts...)
}
