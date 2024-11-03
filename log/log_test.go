package log_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/Andrew-M-C/trpc-go-utils/log"
	"github.com/Andrew-M-C/trpc-go-utils/tracelog"
)

func TestMain(m *testing.M) {
	log.SetLevel("info")
	os.Exit(m.Run())
}

func TestLogger(t *testing.T) {
	log.Debug("Hello", "world", "!")
	log.Infof("formatting %d - %v", 1234, time.Now())

	ctx := context.Background()
	ctx = tracelog.WithTraceID(ctx, "some_id")
	log.WarnContextf(ctx, "看看有没有 tracing '%v'", tracelog.TraceID(ctx))

	testFatal := false
	if testFatal {
		ctx = tracelog.WithTraceID(ctx, "another_id")
		log.FatalContext(ctx, "看看有没有 tracing 和 stack")
	}
	if testFatal {
		log.Fatal("尝试一下 fatal")
	}
}

func TestStructured(t *testing.T) {
	log.New().Str("msg", "Hello, world!").Debug()
	log.New().Any("time", time.Now()).Int("int", 1234).Info()

	ctx := context.Background()
	ctx = tracelog.WithTraceID(ctx, "some_id")
	log.New(ctx).Text("看看有没有 tracing").Str("trace_id", tracelog.TraceID(ctx)).Warn()

	if false {
		ctx = tracelog.WithTraceID(ctx, "another_id")
		log.New(ctx).Text("看看有没有 tracing 和 stack").Fatal()
		log.FatalContext(ctx, "看看有没有 tracing 和 stack")
	}
}

func TestContext(t *testing.T) {
	ctx := context.Background()
	ctx = log.CtxWithBool(ctx, "context", true)
	log.InfoContextf(ctx, "看看有没有 context 字段")

	ctx = log.CtxWithInt64(ctx, "double", -12345)
	log.WarnContextf(ctx, "看看有没有 double 和 context 字段")

	log.New(ctx).Err(errors.New("自定义错误")).Str("msg", "看看有没有 double, context, msg, ERROR 字段").Warn()
}
