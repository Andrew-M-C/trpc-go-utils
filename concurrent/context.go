package concurrent

import (
	"context"

	"github.com/Andrew-M-C/go.util/log/trace"
)

func init() {
	// 支持一下 log 所使用的 trace ID 功能, 复制一下
	RegisterContextKeyWhenDetach(trace.TraceIDContextKey())
}

func RegisterContextKeyWhenDetach(key any) {
	if key == nil {
		return
	}
	internal.contextKeys.Store(key, struct{}{})
}

func copyContextValues(to, from context.Context) context.Context {
	internal.contextKeys.Range(func(key, _ any) bool {
		if v := from.Value(key); v != nil {
			to = context.WithValue(to, key, v)
		}
		return true
	})
	return to
}
