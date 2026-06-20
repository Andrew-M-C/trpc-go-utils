package log_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Andrew-M-C/trpc-go-utils/log"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"trpc.group/trpc-go/trpc-go"
	trpclog "trpc.group/trpc-go/trpc-go/log"
)

func TestMain(m *testing.M) {
	tp := sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.AlwaysSample()))
	otel.SetTracerProvider(tp)
	trpc.NewServer() // 读取 trpc_go.yaml 配置
	log.SetLevel("INFO")
	os.Exit(m.Run())
}

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

func TestIsDying(t *testing.T) {
	ctx := context.Background()
	if log.IsDying(ctx) {
		t.Fatal("background context should not be dying")
	}

	ctx = log.Dying(ctx)
	if !log.IsDying(ctx) {
		t.Fatal("dying context should be marked")
	}
	if log.IsDying(context.Background()) {
		t.Fatal("dying mark should not leak to other contexts")
	}
}

type stubTRPCLogger struct {
	mu     sync.Mutex
	levels map[string][]string
}

func newStubTRPCLogger() *stubTRPCLogger {
	return &stubTRPCLogger{levels: make(map[string][]string)}
}

func (s *stubTRPCLogger) record(level string, args ...interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.levels[level] = append(s.levels[level], fmt.Sprint(args...))
}

func (s *stubTRPCLogger) Trace(args ...interface{}) { s.record("trace", args...) }
func (s *stubTRPCLogger) Tracef(f string, args ...interface{}) {
	s.record("trace", fmt.Sprintf(f, args...))
}
func (s *stubTRPCLogger) Debug(args ...interface{}) { s.record("debug", args...) }
func (s *stubTRPCLogger) Debugf(f string, args ...interface{}) {
	s.record("debug", fmt.Sprintf(f, args...))
}
func (s *stubTRPCLogger) Info(args ...interface{}) { s.record("info", args...) }
func (s *stubTRPCLogger) Infof(f string, args ...interface{}) {
	s.record("info", fmt.Sprintf(f, args...))
}
func (s *stubTRPCLogger) Warn(args ...interface{}) { s.record("warn", args...) }
func (s *stubTRPCLogger) Warnf(f string, args ...interface{}) {
	s.record("warn", fmt.Sprintf(f, args...))
}
func (s *stubTRPCLogger) Error(args ...interface{}) { s.record("error", args...) }
func (s *stubTRPCLogger) Errorf(f string, args ...interface{}) {
	s.record("error", fmt.Sprintf(f, args...))
}
func (s *stubTRPCLogger) Fatal(args ...interface{}) { s.record("fatal", args...) }
func (s *stubTRPCLogger) Fatalf(f string, args ...interface{}) {
	s.record("fatal", fmt.Sprintf(f, args...))
}
func (s *stubTRPCLogger) Sync() error                          { return nil }
func (s *stubTRPCLogger) SetLevel(string, trpclog.Level)       {}
func (s *stubTRPCLogger) GetLevel(string) trpclog.Level        { return trpclog.LevelDebug }
func (s *stubTRPCLogger) With(...trpclog.Field) trpclog.Logger { return s }

func (s *stubTRPCLogger) lines(level string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.levels[level]...)
}

func withStubTRPCLogger(t *testing.T) *stubTRPCLogger {
	t.Helper()

	stub := newStubTRPCLogger()
	old := trpclog.GetDefaultLogger()
	trpclog.SetLogger(stub)
	t.Cleanup(func() {
		trpclog.SetLogger(old)
	})

	log.SetLevel("debug")
	return stub
}

func parseLogFields(t *testing.T, s string) map[string]any {
	t.Helper()
	raw := s
	if idx := strings.Index(s, "\t"); idx >= 0 {
		raw = s[idx+1:]
	}
	var fields map[string]any
	if err := json.Unmarshal([]byte(raw), &fields); err != nil {
		t.Fatalf("parse log json: %v, raw=%q", err, raw)
	}
	return fields
}

func TestDebugContextWhenDying(t *testing.T) {
	stub := withStubTRPCLogger(t)

	log.DebugContext(context.Background(), "normal debug")
	if got := stub.lines("debug"); len(got) != 1 {
		t.Fatalf("expected 1 debug log, got %d: %v", len(got), got)
	}
	if got := stub.lines("info"); len(got) != 0 {
		t.Fatalf("expected no info log, got %v", got)
	}

	stub = withStubTRPCLogger(t)
	ctx := log.Dying(context.Background())
	log.DebugContext(ctx, "dying debug")

	if got := stub.lines("debug"); len(got) != 0 {
		t.Fatalf("dying debug should not use debug level, got %v", got)
	}
	infoLines := stub.lines("info")
	if len(infoLines) != 1 {
		t.Fatalf("expected 1 info log, got %d: %v", len(infoLines), infoLines)
	}

	fields := parseLogFields(t, infoLines[0])
	if fields["DYING"] != true {
		t.Fatalf("DYING = %v, want true", fields["DYING"])
	}
}

func TestDebugContextfWhenDying(t *testing.T) {
	stub := withStubTRPCLogger(t)
	ctx := log.Dying(context.Background())
	log.DebugContextf(ctx, "dying %s %d", "debug", 42)

	infoLines := stub.lines("info")
	if len(infoLines) != 1 {
		t.Fatalf("expected 1 info log, got %d: %v", len(infoLines), infoLines)
	}
	if got := stub.lines("debug"); len(got) != 0 {
		t.Fatalf("dying debug should not use debug level, got %v", got)
	}
	if !strings.Contains(infoLines[0], "dying debug 42") {
		t.Fatalf("formatted message missing from log: %q", infoLines[0])
	}

	fields := parseLogFields(t, infoLines[0])
	if fields["DYING"] != true {
		t.Fatalf("DYING = %v, want true", fields["DYING"])
	}
}
