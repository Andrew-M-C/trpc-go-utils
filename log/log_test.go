package log_test

import (
	"context"
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

	if false {
		ctx = tracelog.WithTraceID(ctx, "another_id")
		log.FatalContext(ctx, "看看有没有 tracing 和 stack")
	}

	// log.Fatal("尝试一下 fatal")
}
