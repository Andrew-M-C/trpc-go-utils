package log

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	jsonvalue "github.com/Andrew-M-C/go.jsonvalue"
	"github.com/Andrew-M-C/go.util/log/trace"
	"github.com/Andrew-M-C/go.util/runtime/caller"
	"github.com/Andrew-M-C/go.util/unsafe"
	"trpc.group/trpc-go/trpc-go/log"
)

const (
	debugLevel = "DEBUG"
	infoLevel  = "INFO"
	warnLevel  = "WARN"
	errorLevel = "ERROR"
	fatalLevel = "FATAL"
)

type loggerKey struct{}

// 往 context 中注入 logger
func WithLogger(ctx context.Context, l *Logger) context.Context {
	if l == nil {
		return ctx
	}
	return context.WithValue(ctx, loggerKey{}, l)
}

// GetLogger 从 context 中读取 logger
func GetLogger(ctx context.Context) *Logger {
	v := ctx.Value(loggerKey{})
	if v == nil {
		return nil
	}
	l, _ := v.(*Logger)
	return l
}

// New 新建一个结构化日志项, 即便是 nil 也是可执行的
func New() *Logger {
	return nil
}

func (l *Logger) Debug() {
	debugLog(nil, l) //nolint: staticcheck
}

func (l *Logger) DebugContext(ctx context.Context) {
	debugLog(ctx, l)
}

func (l *Logger) Info() {
	infoLog(nil, l) //nolint: staticcheck
}

func (l *Logger) InfoContext(ctx context.Context) {
	infoLog(ctx, l)
}

func (l *Logger) Warn() {
	warnLog(nil, l) //nolint: staticcheck
}

func (l *Logger) WarnContext(ctx context.Context) {
	warnLog(ctx, l)
}

func (l *Logger) Error() {
	errorLog(nil, l) //nolint: staticcheck
}

func (l *Logger) ErrorContext(ctx context.Context) {
	errorLog(ctx, l)
}

func (l *Logger) Fatal() {
	fatalLog(nil, l) //nolint: staticcheck
}

func (l *Logger) FatalContext(ctx context.Context) {
	fatalLog(ctx, l)
}

func (l *Logger) Err(err error) *Logger {
	return l.With("ERR", err.Error())
}

func (l *Logger) Text(txt string) *Logger {
	return l.With("", txt)
}

func (l *Logger) WithJSON(key string, v any) *Logger {
	return l.With(key, jsonStringer{v})
}

type jsonStringer struct {
	v any
}

func (j jsonStringer) MarshalJSON() ([]byte, error) {
	b, err := json.Marshal(j.v)
	if err != nil {
		s := fmt.Sprintf("%+v", j.v)
		return unsafe.StoB(s), nil
	}
	return b, nil
}

func (l *Logger) Format(f string, a ...any) *Logger {
	return l.With("", formatStringer{f, a})
}

type formatStringer struct {
	f string
	a []any
}

func (f formatStringer) String() string {
	return fmt.Sprintf(f.f, f.a...)
}

func (l *Logger) misc(a ...any) *Logger {
	return l.With("", miscStringer(a))
}

type miscStringer []any

func (s miscStringer) String() string {
	buff := bytes.Buffer{}
	for i, a := range s {
		if i > 0 {
			buff.WriteRune(' ')
		}
		if e, _ := a.(error); e != nil {
			buff.WriteString(e.Error())
		} else if s, _ := a.(fmt.Stringer); s != nil {
			buff.WriteString(s.String())
		} else {
			buff.WriteString(fmt.Sprintf("%+v", a))
		}
	}
	return buff.String()
}

// With 往 logger 中存入结构化字段
func (l *Logger) With(key string, value any) *Logger {
	return &Logger{
		key:   key,
		value: value,
		prev:  l,
	}
}

// WithCallerStack 往 logger 中存入调用链
func (l *Logger) WithCallerStack() *Logger {
	return l.withCallerStack(3) // 这个值要实际测出来
}

func (l *Logger) withCallerStack(skip int) *Logger {
	callers := caller.GetAllCallers()
	if len(callers) > skip {
		callers = callers[skip:]
	}
	return l.WithJSON("CALLER_STACK", callers)
}

// Logger 结构化 logger
type Logger struct {
	key   string
	value any
	prev  *Logger
}

// ----------------
// MARK: 内部实现

type logStringer struct {
	ctx    context.Context
	level  string
	logger *Logger
}

func debugLog(ctx context.Context, l *Logger) {
	s := logStringer{
		ctx:    ctx,
		level:  "DEBUG",
		logger: l,
	}
	log.Debug(s)
}

func infoLog(ctx context.Context, l *Logger) {
	s := logStringer{
		ctx:    ctx,
		level:  "INFO",
		logger: l,
	}
	log.Info(s)
}

func warnLog(ctx context.Context, l *Logger) {
	s := logStringer{
		ctx:    ctx,
		level:  "WARN",
		logger: l,
	}
	log.Warn(s)
}

func errorLog(ctx context.Context, l *Logger) {
	s := logStringer{
		ctx:    ctx,
		level:  "ERROR",
		logger: l,
	}
	log.Error(s)
}

func fatalLog(ctx context.Context, l *Logger) {
	l = l.withCallerStack(4)
	s := logStringer{
		ctx:    ctx,
		level:  "FATAL",
		logger: l,
	}
	log.Fatal(s)
}

func (l logStringer) String() string {
	j := jsonvalue.NewObject()

	// 时间
	j.MustSetString(time.Now().Local().Format("2006-01-02 15:04:05.000000")).At("TIME")

	// 级别
	j.MustSetString(l.level).At("LEVEL")

	// 调用方信息
	const skip = 10
	fillCaller(caller.GetCaller(skip), j)

	// 自定义字段
	var textFields []string

	iterateFields := func(node *Logger) {
		for ; node != nil; node = node.prev {
			if node.key == "" {
				textFields = append(textFields, fmt.Sprint(node.value))
				continue
			}
			if e, _ := node.value.(error); e != nil {
				j.MustSetString(e.Error()).At(node.key)
			} else if m, _ := node.value.(json.Marshaler); m != nil {
				j.MustSet(m).At(node.key)
			} else if s, _ := node.value.(fmt.Stringer); s != nil {
				j.MustSetString(s.String()).At(node.key)
			} else {
				j.MustSet(node.value).At(node.key)
			}
		}
	}
	if l.ctx != nil {
		if lg := GetLogger(l.ctx); lg != nil {
			iterateFields(lg)
		}
	}
	if l.logger != nil {
		iterateFields(l.logger)
	}

	// trace ID
	if ctx := l.ctx; ctx != nil {
		if traceID := trace.TraceID(ctx); traceID != "" {
			j.MustSetString(traceID).At("TRACE_ID")
		}
		if history := trace.TraceIDStack(ctx); len(history) > 0 {
			j.MustSet(history).At("TRACE_ID_STACK")
		}
	}

	// 序列化返回
	fields := j.MustMarshalString(
		jsonvalue.OptSetSequence(),
		jsonvalue.OptEscapeSlash(false),
		jsonvalue.OptEscapeHTML(false),
		jsonvalue.OptUTF8(),
	)

	if len(textFields) > 0 {
		return strings.Join(textFields, " ") + "\t" + fields
	}
	return fields
}

func fillCaller(c caller.Caller, j *jsonvalue.V) {
	parts := strings.Split(string(c.File), "/")
	if len(parts) > 3 {
		parts = parts[len(parts)-3:]
	}
	j.MustSetString(strings.Join(parts, "/")).At("FILE")
	j.MustSetInt(c.Line).At("LINE")
	j.MustSetString(c.Func.Base()).At("FUNC")
}
