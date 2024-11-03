package log

import (
	"context"
	"fmt"
	"strings"
	"time"

	jsonvalue "github.com/Andrew-M-C/go.jsonvalue"
	"github.com/Andrew-M-C/go.util/runtime/caller"
	"github.com/Andrew-M-C/trpc-go-utils/tracelog"
	"trpc.group/trpc-go/trpc-go/log"
)

const (
	debugLevel = "DEBUG"
	infoLevel  = "INFO"
	warnLevel  = "WARN"
	errorLevel = "ERROR"
	fatalLevel = "FATAL"
)

// New 新建一个结构化日志项
func New(ctx ...context.Context) *Logger {
	l := newLogger("")
	if len(ctx) > 0 && ctx[0] != nil {
		l.setTracing(ctx[0])
	}
	return l
}

func (l *Logger) Any(field string, v any) *Logger {
	l.fieldKeys = append(l.fieldKeys, field)

	j, err := jsonvalue.Import(v)
	if err != nil {
		s := fmt.Sprintf("%+v", v)
		return l.Str(field, s)
	}

	l.fieldValues = append(l.fieldValues, j)
	return l
}

func (l *Logger) Debug() {
	l.level = debugLevel
	log.Debug(l)
}

func (l *Logger) Info() {
	l.level = infoLevel
	log.Info(l)
}

func (l *Logger) Warn() {
	l.level = warnLevel
	log.Warn(l)
}

func (l *Logger) Error() {
	l.level = errorLevel
	log.Error(l)
}

func (l *Logger) Fatal() {
	l.level = fatalLevel
	l.setCallerStacks()
	log.Fatal(l)
}

func (l *Logger) Text(s string) *Logger {
	l.args = []any{s}
	return l
}

func (l *Logger) Str(field string, s string) *Logger {
	l.fieldKeys = append(l.fieldKeys, field)
	l.fieldValues = append(l.fieldValues, jsonvalue.NewString(s))
	return l
}

func (l *Logger) Bool(field string, b bool) *Logger {
	l.fieldKeys = append(l.fieldKeys, field)
	l.fieldValues = append(l.fieldValues, jsonvalue.NewBool(b))
	return l
}

func (l *Logger) Int(field string, i int) *Logger {
	l.fieldKeys = append(l.fieldKeys, field)
	l.fieldValues = append(l.fieldValues, jsonvalue.NewInt(i))
	return l
}

func (l *Logger) Uint(field string, u uint) *Logger {
	l.fieldKeys = append(l.fieldKeys, field)
	l.fieldValues = append(l.fieldValues, jsonvalue.NewUint(u))
	return l
}

func (l *Logger) Int8(field string, i int8) *Logger {
	l.fieldKeys = append(l.fieldKeys, field)
	l.fieldValues = append(l.fieldValues, jsonvalue.NewInt(int(i)))
	return l
}

func (l *Logger) Uint8(field string, u uint8) *Logger {
	l.fieldKeys = append(l.fieldKeys, field)
	l.fieldValues = append(l.fieldValues, jsonvalue.NewUint(uint(u)))
	return l
}

func (l *Logger) Int16(field string, i int16) *Logger {
	l.fieldKeys = append(l.fieldKeys, field)
	l.fieldValues = append(l.fieldValues, jsonvalue.NewInt(int(i)))
	return l
}

func (l *Logger) Uint16(field string, u uint16) *Logger {
	l.fieldKeys = append(l.fieldKeys, field)
	l.fieldValues = append(l.fieldValues, jsonvalue.NewUint(uint(u)))
	return l
}

func (l *Logger) Int32(field string, i int32) *Logger {
	l.fieldKeys = append(l.fieldKeys, field)
	l.fieldValues = append(l.fieldValues, jsonvalue.NewInt32(i))
	return l
}

func (l *Logger) Uint32(field string, u uint32) *Logger {
	l.fieldKeys = append(l.fieldKeys, field)
	l.fieldValues = append(l.fieldValues, jsonvalue.NewUint32(u))
	return l
}

func (l *Logger) Int64(field string, i int64) *Logger {
	l.fieldKeys = append(l.fieldKeys, field)
	l.fieldValues = append(l.fieldValues, jsonvalue.NewInt64(i))
	return l
}

func (l *Logger) Uint64(field string, u uint64) *Logger {
	l.fieldKeys = append(l.fieldKeys, field)
	l.fieldValues = append(l.fieldValues, jsonvalue.NewUint64(u))
	return l
}

func (l *Logger) Float32(field string, f float32) *Logger {
	l.fieldKeys = append(l.fieldKeys, field)
	l.fieldValues = append(l.fieldValues, jsonvalue.NewFloat32(f))
	return l
}

func (l *Logger) Float64(field string, f float64) *Logger {
	l.fieldKeys = append(l.fieldKeys, field)
	l.fieldValues = append(l.fieldValues, jsonvalue.NewFloat64(f))
	return l
}

func (l *Logger) Err(err error) *Logger {
	l.err = err
	return l
}

type Logger struct {
	level   string
	callers []caller.Caller

	err error

	traceID      string
	traceIDStack []string

	fieldKeys   []string
	fieldValues []*jsonvalue.V

	formatting bool
	format     string
	args       []any

	fullCaller bool
}

// ----------------
// MARK: 内部实现

func newLogger(level string) *Logger {
	l := &Logger{
		level: level,
	}
	callers := caller.GetAllCallers()
	l.callers = callers[3:]
	return l
}

func (l *Logger) setFormatting(f string, args []any) {
	l.formatting = true
	l.format = f
	l.args = args
}

func (l *Logger) setTracing(ctx context.Context) {
	traceID := tracelog.TraceID(ctx)
	if traceID != "" {
		l.traceID = traceID
	}
	history := tracelog.TraceIDStack(ctx)
	if len(history) > 0 {
		l.traceIDStack = history
	}
}

func (l *Logger) setCallerStacks() {
	l.fullCaller = true
}

func (l *Logger) String() string {
	v := jsonvalue.NewObject()
	v.MustSetString(time.Now().Local().Format("2006-01-02 15:04:05.000000")).At("TIME")

	if l.level != "" {
		v.MustSetString(l.level).At("LEVEL")
	}

	if l.err != nil {
		v.MustSetString(l.err.Error()).At("ERROR")
	}

	if l.formatting {
		msg := fmt.Sprintf(l.format, l.args...)
		if msg != "" {
			v.MustSetString(msg).At("TEXT")
		}
	} else if len(l.args) > 0 {
		msg := fmt.Sprint(l.args...)
		if msg != "" {
			v.MustSetString(msg).At("TEXT")
		}
	}

	for i, k := range l.fieldKeys {
		v.MustSet(l.fieldValues[i]).At(k)
	}

	if l.fullCaller {
		v.MustSet(l.callers).At("STACK")
	}

	if len(l.callers) > 0 {
		caller := l.callers[0]
		v.MustSetString(getFile(caller.File)).At("FILE")
		v.MustSetInt(caller.Line).At("LINE")
		v.MustSetString(caller.Func.Name()).At("FUNC")
	}

	if l.traceID != "" {
		v.MustSetString(l.traceID).At("TRACE_ID")
	}
	if len(l.traceIDStack) > 0 {
		v.MustSet(l.traceIDStack).At("TRACE_ID_STACK")
	}

	s := v.MustMarshalString(
		jsonvalue.OptSetSequence(),
		jsonvalue.OptEscapeSlash(false),
		jsonvalue.OptUTF8(),
	)
	return s
}

func getFile(f caller.File) string {
	parts := strings.Split(string(f), "/")
	if len(parts) > 3 {
		parts = parts[len(parts)-3:]
	}
	return strings.Join(parts, "/")
}
