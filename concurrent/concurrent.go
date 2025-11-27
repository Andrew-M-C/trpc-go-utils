// Package concurrent 定义一些并发相关的功能
package concurrent

import (
	"context"
	"fmt"
	"time"

	"github.com/Andrew-M-C/trpc-go-utils/log"
	"github.com/Andrew-M-C/trpc-go-utils/recovery"
	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
)

// Detach 分离一个新的后台任务, 不等待其返回
func Detach(ctx context.Context, task func(context.Context), recoveryOpts ...recovery.Option) {
	newCtx, _ := codec.WithCloneContextAndMessage(ctx)
	newCtx = copyContextValues(newCtx, ctx)

	recoveryOpts = append(recoveryOpts, recovery.WithContext(ctx))
	if id := log.TraceID(ctx); id != "" {
		detachTraceID := "detach-" + time.Now().Format("0102-150405")
		newCtx = log.WithTraceID(newCtx, detachTraceID)
	}

	go func() {
		defer recovery.CatchPanic(recoveryOpts...)
		task(newCtx)
	}()
}

// DetachAndWait 分离多个任务, 并等待所有任务完成
func DetachAndWait(ctx context.Context, tasks ...func(context.Context) error) error {
	if len(tasks) == 0 {
		return nil
	}

	handlers := make([]func() error, 0, len(tasks))
	for i, t := range tasks {
		if t == nil {
			continue
		}
		handlers = append(handlers, func() error {
			ctx := ctx
			if id := log.TraceID(ctx); id != "" {
				ctx = log.WithTraceID(ctx, fmt.Sprintf("concurrent.detach_and_wait.%d", i))
			}
			return t(ctx)
		})
	}

	return trpc.GoAndWait(handlers...)
}
