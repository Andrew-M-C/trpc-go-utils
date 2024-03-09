package recovery

import (
	"context"

	"github.com/Andrew-M-C/go.util/runtime/caller"
)

// PanicCallback 发生 panic 时的回调函数
type PanicCallback func(ctx context.Context, info any, stack []caller.Caller)

// Option 表示额外参数
type Option func(opt *option)

// WithContext context 参数
func WithContext(ctx context.Context) Option {
	return func(opt *option) {
		opt.ctx = ctx
	}
}

// WithErrorLog 在 panic 时打印详细的错误日志
func WithErrorLog() Option {
	return func(opt *option) {
		opt.errorLog = true
	}
}

// WithMetrics 当 panic 时调用 metrics.Counter(key).Incr()
func WithMetrics(key string) Option {
	return func(opt *option) {
		opt.metricsName = key
	}
}

// WithCallback 出现 panic 时回调
func WithCallback(f PanicCallback) Option {
	return func(o *option) {
		o.callback = f
	}
}

type option struct {
	ctx         context.Context
	errorLog    bool
	callback    PanicCallback
	metricsName string
}

func mergeOptions(opts []Option) *option {
	opt := &option{}
	for _, o := range opts {
		if o != nil {
			o(opt)
		}
	}
	return opt
}
