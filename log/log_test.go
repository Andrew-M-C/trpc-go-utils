package log_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Andrew-M-C/trpc-go-utils/log"
)

func TestLogger(*testing.T) {
	log.Debug("Hello", "world", "!")
	log.Infof("formatting %d - %v", 1234, time.Now())
	log.WarnContextf(context.Background(), "无 trace context 时不应输出 trace_id / span_id")

	testFatal := false
	if testFatal {
		log.FatalContext(context.Background(), "看看有没有 tracing 和 stack")
	}
	if testFatal {
		log.Fatal("尝试一下 fatal")
	}
}

func TestStructured(*testing.T) {
	log.New().With("msg", "Hello, world!").Debug()
	log.New().With("time", time.Now()).With("int", 1234).Info()
	log.New().Text("无 trace context").WarnContext(context.Background())

	if false {
		log.New().Text("看看有没有 tracing 和 stack").FatalContext(context.Background())
		log.FatalContext(context.Background(), "看看有没有 tracing 和 stack")
	}
}

func TestContext(*testing.T) {
	ctx := context.Background()
	l := log.New().With("context", true)
	ctx = log.WithLogger(ctx, l)
	log.InfoContextf(ctx, "看看有没有 context 字段")

	l = log.GetLogger(ctx)
	l = l.With("double", -12345.6)
	ctx = log.WithLogger(ctx, l)
	log.WarnContextf(ctx, "看看有没有 double 和 context 字段")

	log.New().Err(errors.New("自定义错误")).With("msg", "看看有没有 double, context, msg, ERR 字段").WarnContext(ctx)

	l.WithCallerStack().Text("看看有没有 CALLER_STACK").Info()
}

func TestFatal(*testing.T) {
	// log.Fatal("看看有没有 CALLER_STACK")
}
