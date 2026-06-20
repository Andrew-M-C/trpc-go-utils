// Package log 表示经过封装的日志功能
package log

import (
	"context"
	"strings"

	"trpc.group/trpc-go/trpc-go/log"
)

// MARK: 日志级别设置

// SetLevel 设置日志级别, 参数为 debug, info, warn, error, fatal 这些
func SetLevel(levelString string) {
	var lv log.Level

	switch strings.TrimSpace(strings.ToUpper(levelString)) {
	default:
		return
	case "DEBUG", "DBG":
		lv = log.LevelDebug
	case "INFO", "INF":
		lv = log.LevelInfo
	case "WARN", "WRN", "WARNING":
		lv = log.LevelWarn
	case "ERROR", "ERR":
		lv = log.LevelError
	case "FATAL", "FTL":
		lv = log.LevelFatal
	}

	log.SetLevel("1", lv)
	log.SetLevel("2", lv)
}

// MARK: 没有 context 的 arg 列表

// Debug 输出 debug 级别参数列表日志
func Debug(v ...any) {
	l := New().misc(v...)
	debugLog(context.Background(), l)
}

// Info 输出 info 级别参数列表日志
func Info(v ...any) {
	l := New().misc(v...)
	infoLog(context.Background(), l)
}

// Warn 输出 warn 级别参数列表日志
func Warn(v ...any) {
	l := New().misc(v...)
	warnLog(context.Background(), l)
}

// Error 输出 error 级别参数列表日志
func Error(v ...any) {
	l := New().misc(v...)
	errorLog(context.Background(), l)
}

// Fatal 输出 fatal 级别参数列表日志
func Fatal(v ...any) {
	l := New().misc(v...)
	fatalLog(context.Background(), l)
}

// MARK: 没有 context 的 formatting

// Debugf 格式化输出 debug 级别日志
func Debugf(f string, v ...any) {
	l := New().Format(f, v...)
	debugLog(context.Background(), l)
}

// Infof 格式化输出 info 级别日志
func Infof(f string, v ...any) {
	l := New().Format(f, v...)
	infoLog(context.Background(), l)
}

// Warnf 格式化输出 warn 级别日志
func Warnf(f string, v ...any) {
	l := New().Format(f, v...)
	warnLog(context.Background(), l)
}

// Errorf 格式化输出 error 级别日志
func Errorf(f string, v ...any) {
	l := New().Format(f, v...)
	errorLog(context.Background(), l)
}

// Fatalf 格式化输出 fatal 级别日志
func Fatalf(f string, v ...any) {
	l := New().Format(f, v...)
	fatalLog(context.Background(), l)
}

// MARK: 带 context 的参数列表

// DebugContext 输出 debug 级别参数列表日志
func DebugContext(ctx context.Context, v ...any) {
	// 如果染色, 那么提升至 INFO 级别
	if IsDying(ctx) {
		l := New().With("DYING", true).misc(v...)
		infoLog(ctx, l)
		return
	}

	// 正常 DEBUG 日志
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
	// 如果染色, 那么提升至 INFO 级别
	if IsDying(ctx) {
		l := New().With("DYING", true).Format(f, v...)
		infoLog(ctx, l)
		return
	}

	// 正常 DEBUG 日志
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
