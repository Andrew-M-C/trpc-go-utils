package concurrent

import (
	"context"

	oteltrace "go.opentelemetry.io/otel/trace"
)

func RegisterContextKeyWhenDetach(key any) {
	if key == nil {
		return
	}
	internal.contextKeys.Store(key, struct{}{})
}

// copyTracing 将 from 中的 OTel span 复制到 to，使 NewSpan 能在 detached context 下创建子 span。
func copyTracing(from, to context.Context) context.Context {
	span := oteltrace.SpanFromContext(from)
	if span == nil || !span.SpanContext().IsValid() {
		return to
	}
	return oteltrace.ContextWithSpan(to, span)
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
