package log

import (
	"context"
	"fmt"
	"strings"
	"time"

	jsonvalue "github.com/Andrew-M-C/go.jsonvalue"
	"github.com/Andrew-M-C/go.util/runtime/caller"
	"github.com/Andrew-M-C/trpc-go-utils/tracelog"
)

const (
	debugLevel = "DEBUG"
	infoLevel  = "INFO"
	warnLevel  = "WARN"
	errorLevel = "ERROR"
	fatalLevel = "FATAL"
)

type logger struct {
	level   string
	callers []caller.Caller

	traceID      string
	traceIDStack []string

	fieldKeys   []string
	fieldValues []*jsonvalue.V

	formatting bool
	format     string
	args       []any

	fullCaller bool
}

func newLogger(level string) *logger {
	l := &logger{
		level: level,
	}
	callers := caller.GetAllCallers()
	l.callers = callers[3:]
	return l
}

func (l *logger) setFormatting(f string, args []any) {
	l.formatting = true
	l.format = f
	l.args = args
}

func (l *logger) setTracing(ctx context.Context) {
	traceID := tracelog.TraceID(ctx)
	if traceID != "" {
		l.traceID = traceID
	}
	history := tracelog.TraceIDStack(ctx)
	if len(history) > 0 {
		l.traceIDStack = history
	}
}

func (l *logger) setCallerStacks() {
	l.fullCaller = true
}

func (l *logger) String() string {
	v := jsonvalue.NewObject()
	v.MustSetString(time.Now().Local().Format("2006-01-02 15:04:05.000000")).At("TIME")

	if l.level != "" {
		v.MustSetString(l.level).At("LEVEL")
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
