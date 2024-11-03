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
	if len(ctx) > 0 {
		l.ctx = ctx[0]
	}
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

func (l *Logger) Any(field string, v any) *Logger {
	l.fields = append(l.fields, func(v *jsonvalue.V) {
		j, err := jsonvalue.Import(v)
		if err != nil {
			s := fmt.Sprintf("%+v", v)
			v.MustSetString(s).At(field)
		}
		v.MustSet(j).At(field)
	})
	return l
}

func (l *Logger) Text(s string) *Logger {
	l.args = []any{s}
	return l
}

func (l *Logger) Str(field string, s string) *Logger {
	l.fields = append(l.fields, func(v *jsonvalue.V) {
		v.MustSetString(s).At(field)
	})
	return l
}

func (l *Logger) Stringer(field string, s fmt.Stringer) *Logger {
	l.fields = append(l.fields, func(v *jsonvalue.V) {
		v.MustSetString(s.String()).At(field)
	})
	return l
}

func (l *Logger) Bool(field string, b bool) *Logger {
	l.fields = append(l.fields, func(v *jsonvalue.V) {
		v.MustSetBool(b).At(field)
	})
	return l
}

func (l *Logger) Int(field string, i int) *Logger {
	l.fields = append(l.fields, func(v *jsonvalue.V) {
		v.MustSetInt(i).At(field)
	})
	return l
}

func (l *Logger) Uint(field string, u uint) *Logger {
	l.fields = append(l.fields, func(v *jsonvalue.V) {
		v.MustSetUint(u).At(field)
	})
	return l
}

func (l *Logger) Int64(field string, i int64) *Logger {
	l.fields = append(l.fields, func(v *jsonvalue.V) {
		v.MustSetInt64(i).At(field)
	})
	return l
}

func (l *Logger) Uint64(field string, u uint64) *Logger {
	l.fields = append(l.fields, func(v *jsonvalue.V) {
		v.MustSetUint64(u).At(field)
	})
	return l
}

func (l *Logger) Float32(field string, f float32) *Logger {
	l.fields = append(l.fields, func(v *jsonvalue.V) {
		v.MustSetFloat32(f).At(field)
	})
	return l
}

func (l *Logger) Float64(field string, f float64) *Logger {
	l.fields = append(l.fields, func(v *jsonvalue.V) {
		v.MustSetFloat64(f).At(field)
	})
	return l
}

func (l *Logger) Err(err error) *Logger {
	l.err = err
	return l
}

type Logger struct {
	ctx context.Context

	level   string
	callers []caller.Caller

	err error

	fields []func(*jsonvalue.V)

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

func (l *Logger) setCallerStacks() {
	l.fullCaller = true
}

func (l *Logger) String() string {
	v := jsonvalue.NewObject()

	// time
	v.MustSetString(time.Now().Local().Format("2006-01-02 15:04:05.000000")).At("TIME")

	// level
	if l.level != "" {
		v.MustSetString(l.level).At("LEVEL")
	}

	// error
	if l.err != nil {
		v.MustSetString(l.err.Error()).At("ERROR")
	}

	// message
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

	// fields
	for _, fu := range l.fields {
		fu(v)
	}

	// fields in ctx
	extractCtxKVs(l.ctx, v)

	// stack for fatal
	if l.fullCaller {
		v.MustSet(l.callers).At("STACK")
	}

	// callers
	if len(l.callers) > 0 {
		caller := l.callers[0]
		v.MustSetString(getFile(caller.File)).At("FILE")
		v.MustSetInt(caller.Line).At("LINE")
		v.MustSetString(caller.Func.Name()).At("FUNC")
	}

	// tracing
	if l.ctx != nil {
		if traceID := tracelog.TraceID(l.ctx); traceID != "" {
			v.MustSetString(traceID).At("TRACE_ID")
		}
		if history := tracelog.TraceIDStack(l.ctx); len(history) > 0 {
			v.MustSet(history).At("TRACE_ID_STACK")
		}
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
