// Package log 表示经过封装的日志功能
package log

import (
	"context"
	"strings"

	"github.com/Andrew-M-C/go.util/log/trace"
	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/log"
)

// MARK: tracing 配置

// WithTraceID 更新 trace ID
var WithTraceID = trace.WithTraceID

// TraceID 从 context 中读取 trace ID
var TraceID = trace.TraceID

// EnsureTraceID 确保 context 中有一个 trace ID
var EnsureTraceID = trace.EnsureTraceID

// CloneContextForConcurrency 复制一个用于并发操作的新的 ctx, 包含 timeout 和 cancel 同步
func CloneContextForConcurrency(ctx context.Context) context.Context {
	newCtx, _ := codec.WithCloneContextAndMessage(ctx)
	return copyTracing(ctx, newCtx)
}

// CloneContextForDetach 复制一个用于分离操作的新的 ctx, 不包含 timeout 和 cancel 同步
func CloneContextForDetach(ctx context.Context) context.Context {
	newCtx := trpc.CloneContext(ctx)
	return copyTracing(ctx, newCtx)
}

func copyTracing(from, to context.Context) context.Context {
	if stack := trace.TraceIDStack(from); len(stack) > 0 {
		to = trace.WithTraceIDStack(to, stack)
	}
	if id := trace.TraceID(from); id != "" {
		to = trace.WithTraceID(to, id)
	}
	return to
}

// MARK: 日志级别设置

// SetLevel 设置日志级别, 参数为 debug, info, warn, error, fatal 这些
func SetLevel(levelString string) {
	var lv log.Level

	switch strings.TrimSpace(strings.ToUpper(levelString)) {
	default:
		return
	case debugLevel:
		lv = log.LevelDebug
	case infoLevel:
		lv = log.LevelInfo
	case warnLevel, "WARNING":
		lv = log.LevelWarn
	case errorLevel:
		lv = log.LevelError
	case fatalLevel:
		lv = log.LevelFatal
	}

	log.SetLevel("1", lv)
	log.SetLevel("2", lv)
}

// MARK: 没有 context 的 arg 列表

// Debug 输出 debug 级别参数列表日志
func Debug(v ...any) {
	l := New().misc(v...)
	debugLog(nil, l) //nolint: staticcheck
}

// Info 输出 info 级别参数列表日志
func Info(v ...any) {
	l := New().misc(v...)
	infoLog(nil, l) //nolint: staticcheck
}

// Warn 输出 warn 级别参数列表日志
func Warn(v ...any) {
	l := New().misc(v...)
	warnLog(nil, l) //nolint: staticcheck
}

// Error 输出 error 级别参数列表日志
func Error(v ...any) {
	l := New().misc(v...)
	errorLog(nil, l) //nolint: staticcheck
}

// Fatal 输出 fatal 级别参数列表日志
func Fatal(v ...any) {
	l := New().misc(v...)
	fatalLog(nil, l) //nolint: staticcheck
}

// MARK: 没有 context 的 formatting

// Debugf 格式化输出 debug 级别日志
func Debugf(f string, v ...any) {
	l := New().Format(f, v...)
	debugLog(nil, l) //nolint: staticcheck
}

// Infof 格式化输出 info 级别日志
func Infof(f string, v ...any) {
	l := New().Format(f, v...)
	infoLog(nil, l) //nolint: staticcheck
}

// Warnf 格式化输出 warn 级别日志
func Warnf(f string, v ...any) {
	l := New().Format(f, v...)
	warnLog(nil, l) //nolint: staticcheck
}

// Errorf 格式化输出 error 级别日志
func Errorf(f string, v ...any) {
	l := New().Format(f, v...)
	errorLog(nil, l) //nolint: staticcheck
}

// Fatalf 格式化输出 fatal 级别日志
func Fatalf(f string, v ...any) {
	l := New().Format(f, v...)
	fatalLog(nil, l) //nolint: staticcheck
}

// MARK: 带 context 的参数列表

// DebugContext 输出 debug 级别参数列表日志
func DebugContext(ctx context.Context, v ...any) {
	l := New().misc(v...)
	debugLog(ctx, l)
}

// InfoContext 输出 info 级别参数列表日志
func InfoContext(ctx context.Context, v ...any) {
	l := New().misc(v...)
	infoLog(ctx, l)
}

// WarnContext 输出 warn 级别参数列表日志
func WarnContext(ctx context.Context, v ...any) {
	l := New().misc(v...)
	warnLog(ctx, l)
}

// ErrorContext 输出 error 级别参数列表日志
func ErrorContext(ctx context.Context, v ...any) {
	l := New().misc(v...)
	errorLog(ctx, l)
}

// FatalContext 输出 fatal 级别参数列表日志
func FatalContext(ctx context.Context, v ...any) {
	l := New().misc(v...)
	fatalLog(ctx, l)
}

// MARK: 带 context 的 formatting

// DebugContextf 格式化输出 debug 级别日志
func DebugContextf(ctx context.Context, f string, v ...any) {
	l := New().Format(f, v...)
	debugLog(ctx, l)
}

// InfoContextf 格式化输出 info 级别日志
func InfoContextf(ctx context.Context, f string, v ...any) {
	l := New().Format(f, v...)
	infoLog(ctx, l)
}

// WarnContextf 格式化输出 warn 级别日志
func WarnContextf(ctx context.Context, f string, v ...any) {
	l := New().Format(f, v...)
	warnLog(ctx, l)
}

// ErrorContextf 格式化输出 error 级别日志
func ErrorContextf(ctx context.Context, f string, v ...any) {
	l := New().Format(f, v...)
	errorLog(ctx, l)
}

// FatalContextf 格式化输出 fatal 级别日志
func FatalContextf(ctx context.Context, f string, v ...any) {
	l := New().Format(f, v...)
	fatalLog(ctx, l)
}
