// Package log 表示经过封装的日志功能
package log

import (
	"context"
	"strings"

	"trpc.group/trpc-go/trpc-go/log"
)

// 日志级别设置

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
	l := newLogger(debugLevel)
	l.args = v
	log.Debug(l)
}

// Info 输出 info 级别参数列表日志
func Info(v ...any) {
	l := newLogger(infoLevel)
	l.args = v
	log.Info(l)
}

// Warn 输出 warn 级别参数列表日志
func Warn(v ...any) {
	l := newLogger(warnLevel)
	l.args = v
	log.Warn(l)
}

// Error 输出 error 级别参数列表日志
func Error(v ...any) {
	l := newLogger(errorLevel)
	l.args = v
	log.Error(l)
}

// Fatal 输出 fatal 级别参数列表日志
func Fatal(v ...any) {
	l := newLogger(fatalLevel)
	l.args = v
	l.setCallerStacks()
	log.Fatal(l)
}

// MARK: 没有 context 的 formatting

// Debugf 格式化输出 debug 级别日志
func Debugf(f string, v ...any) {
	l := newLogger(debugLevel)
	l.setFormatting(f, v)
	log.Debug(l)
}

// Infof 格式化输出 info 级别日志
func Infof(f string, v ...any) {
	l := newLogger(infoLevel)
	l.setFormatting(f, v)
	log.Info(l)
}

// Warnf 格式化输出 warn 级别日志
func Warnf(f string, v ...any) {
	l := newLogger(warnLevel)
	l.setFormatting(f, v)
	log.Warn(l)
}

// Errorf 格式化输出 error 级别日志
func Errorf(f string, v ...any) {
	l := newLogger(errorLevel)
	l.setFormatting(f, v)
	log.Error(l)
}

// Fatalf 格式化输出 fatal 级别日志
func Fatalf(f string, v ...any) {
	l := newLogger(fatalLevel)
	l.setFormatting(f, v)
	l.setCallerStacks()
	log.Fatal(l)
}

// MARK: 带 context 的参数列表

// DebugContext 输出 debug 级别参数列表日志
func DebugContext(ctx context.Context, v ...any) {
	l := newLogger(debugLevel)
	l.setTracing(ctx)
	l.args = v
	log.Debug(l)
}

// InfoContext 输出 info 级别参数列表日志
func InfoContext(ctx context.Context, v ...any) {
	l := newLogger(infoLevel)
	l.setTracing(ctx)
	l.args = v
	log.Info(l)
}

// WarnContext 输出 warn 级别参数列表日志
func WarnContext(ctx context.Context, v ...any) {
	l := newLogger(warnLevel)
	l.setTracing(ctx)
	l.args = v
	log.Warn(l)
}

// ErrorContext 输出 error 级别参数列表日志
func ErrorContext(ctx context.Context, v ...any) {
	l := newLogger(errorLevel)
	l.setTracing(ctx)
	l.args = v
	log.Error(l)
}

// FatalContext 输出 fatal 级别参数列表日志
func FatalContext(ctx context.Context, v ...any) {
	l := newLogger(fatalLevel)
	l.setTracing(ctx)
	l.args = v
	l.setCallerStacks()
	log.Fatal(l)
}

// MARK: 带 context 的 formatting

// DebugContextf 格式化输出 debug 级别日志
func DebugContextf(ctx context.Context, f string, v ...any) {
	l := newLogger(debugLevel)
	l.setTracing(ctx)
	l.setFormatting(f, v)
	log.Debug(l)
}

// InfoContextf 格式化输出 info 级别日志
func InfoContextf(ctx context.Context, f string, v ...any) {
	l := newLogger(infoLevel)
	l.setTracing(ctx)
	l.setFormatting(f, v)
	log.Info(l)
}

// WarnContextf 格式化输出 warn 级别日志
func WarnContextf(ctx context.Context, f string, v ...any) {
	l := newLogger(warnLevel)
	l.setTracing(ctx)
	l.setFormatting(f, v)
	log.Warn(l)
}

// ErrorContextf 格式化输出 error 级别日志
func ErrorContextf(ctx context.Context, f string, v ...any) {
	l := newLogger(errorLevel)
	l.setTracing(ctx)
	l.setFormatting(f, v)
	log.Error(l)
}

// FatalContextf 格式化输出 fatal 级别日志
func FatalContextf(ctx context.Context, f string, v ...any) {
	l := newLogger(fatalLevel)
	l.setTracing(ctx)
	l.setFormatting(f, v)
	log.Fatal(l)
}
