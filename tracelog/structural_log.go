package tracelog

import (
	"context"
	"fmt"

	"github.com/Andrew-M-C/go.util/runtime/caller"
	"go.uber.org/zap"
	"trpc.group/trpc-go/trpc-go/log"
)

// StructuralLogger 结构化日志
type StructuralLogger struct {
	fields []any
}

// Structural 构建结构化日志
func Structural(message string) *StructuralLogger {
	l := &StructuralLogger{}
	l.fields = append(l.fields, message)
	c := caller.GetCaller(1)
	l.fields = append(l.fields, zap.Int("__LINE__", c.Line))
	l.fields = append(l.fields, zap.String("__FUNC__", string(c.Func)))
	l.fields = append(l.fields, zap.String("__FILE__", string(c.File)))
	return l
}

// MARK 各级日志实现

func (l *StructuralLogger) Debug() {
	log.Debug(l.fields...)
}

func (l *StructuralLogger) DebugContext(ctx context.Context) {
	log.DebugContext(ctx, l.fields...)
}

func (l *StructuralLogger) Info() {
	log.Info(l.fields...)
}

func (l *StructuralLogger) InfoContext(ctx context.Context) {
	log.InfoContext(ctx, l.fields...)
}

func (l *StructuralLogger) Warn() {
	log.Warn(l.fields...)
}

func (l *StructuralLogger) WarnContext(ctx context.Context) {
	log.WarnContext(ctx, l.fields...)
}

func (l *StructuralLogger) Error() {
	log.Error(l.fields...)
}

func (l *StructuralLogger) ErrorContext(ctx context.Context) {
	log.ErrorContext(ctx, l.fields...)
}

func (l *StructuralLogger) Fatal() {
	stack := caller.GetAllCallers()
	l.fields = append(l.fields, zap.Stringer("__STACK__", ToJSON(stack)))
	log.Fatal(l.fields...)
}

func (l *StructuralLogger) FatalContext(ctx context.Context) {
	stack := caller.GetAllCallers()
	l.fields = append(l.fields, zap.Stringer("__STACK__", ToJSON(stack)))
	log.FatalContext(ctx, l.fields...)
}

// MARK zap 封装

func (l *StructuralLogger) String(key, value string) *StructuralLogger {
	f := zap.String(key, value)
	l.fields = append(l.fields, f)
	return l
}

func (l *StructuralLogger) Int(key string, i int) *StructuralLogger {
	f := zap.Int(key, i)
	l.fields = append(l.fields, f)
	return l
}

func (l *StructuralLogger) Int64(key string, i int64) *StructuralLogger {
	f := zap.Int64(key, i)
	l.fields = append(l.fields, f)
	return l
}

func (l *StructuralLogger) Int32(key string, i int32) *StructuralLogger {
	f := zap.Int32(key, i)
	l.fields = append(l.fields, f)
	return l
}

func (l *StructuralLogger) Bool(key string, b bool) *StructuralLogger {
	f := zap.Bool(key, b)
	l.fields = append(l.fields, f)
	return l
}

func (l *StructuralLogger) Any(key string, v any) *StructuralLogger {
	f := zap.Any(key, v)
	l.fields = append(l.fields, f)
	return l
}

func (l *StructuralLogger) Stringer(key string, v fmt.Stringer) *StructuralLogger {
	f := zap.Stringer(key, v)
	l.fields = append(l.fields, f)
	return l
}

func (l *StructuralLogger) GoError(err error) *StructuralLogger {
	f := zap.Error(err)
	l.fields = append(l.fields, f)
	return l
}
