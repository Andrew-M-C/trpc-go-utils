package log

import (
	"context"
)

type dyingKey struct{}

// Dying 标记染色
func Dying(ctx context.Context) context.Context {
	return context.WithValue(ctx, dyingKey{}, struct{}{})
}

// IsDying 判断是否染色
func IsDying(ctx context.Context) bool {
	return ctx.Value(dyingKey{}) != nil
}
