// Package recovery 提供适配 tRPC 的捕获异常工具
package recovery

import (
	"context"
	"reflect"
	"strings"

	"github.com/Andrew-M-C/go.util/runtime/caller"
	"github.com/Andrew-M-C/trpc-go-utils/log"
	"trpc.group/trpc-go/trpc-go/metrics"
)

// CatchPanic 捕获异常
func CatchPanic(opts ...Option) {
	info := recover()
	if info == nil {
		return
	}

	// 以下参照 github.com/Andrew-M-C/go.util/recovery
	stack := caller.GetAllCallers()
	stack = stack[2:]
	for len(stack) > 0 {
		// 查找 stack 直至找到业务代码
		s := stack[0]
		if strings.HasSuffix(string(s.File), "/runtime/panic.go") {
			stack = stack[1:]
		} else {
			break
		}
	}

	opt := mergeOptions(opts)

	if opt.metricsName != "" {
		metrics.Counter(opt.metricsName).Incr()
	}

	ctx := context.Background()
	if opt.ctx != nil {
		ctx = opt.ctx
	}

	if opt.errorLog {
		log.New().
			Text("Panic caught").
			With("PANIC_TYPE", reflect.TypeOf(info)).
			With("PANIC_INFO", info).
			WithCallerStack().
			ErrorContext(ctx)
	}
	if opt.callback != nil {
		opt.callback(ctx, info, stack)
	}
}
