package log_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/Andrew-M-C/go.util/log/trace"
	"github.com/Andrew-M-C/trpc-go-utils/log"
)

func TestMain(m *testing.M) {
	log.SetLevel("info")
	os.Exit(m.Run())
}

func TestLogger(*testing.T) {
	log.Debug("Hello", "world", "!")
	log.Infof("formatting %d - %v", 1234, time.Now())

	ctx := context.Background()
	ctx = trace.WithTraceID(ctx, "some_id")
	log.WarnContextf(ctx, "看看有没有 tracing '%v'", trace.TraceID(ctx))

	testFatal := false
	if testFatal {
		ctx = trace.WithTraceID(ctx, "another_id")
		log.FatalContext(ctx, "看看有没有 tracing 和 stack")
	}
	if testFatal {
		log.Fatal("尝试一下 fatal")
	}
}

func TestStructured(*testing.T) {
	log.New().With("msg", "Hello, world!").Debug()
	log.New().With("time", time.Now()).With("int", 1234).Info()

	ctx := context.Background()
	ctx = trace.WithTraceID(ctx, "some_id")
	log.New().Text("看看有没有 tracing").With("trace_id", trace.TraceID(ctx)).WarnContext(ctx)

	if false {
		ctx = trace.WithTraceID(ctx, "another_id")
		log.New().Text("看看有没有 tracing 和 stack").FatalContext(ctx)
		log.FatalContext(ctx, "看看有没有 tracing 和 stack")
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
