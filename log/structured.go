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
	v *jsonvalue.V

	formatting bool
	format     string
	args       []any
}

func newLogger(level string) *logger {
	l := &logger{
		v: jsonvalue.NewObject(),
	}
	l.v.MustSetString(time.Now().Local().Format("2006-01-02 15:04:05")).At("TIME")
	l.v.MustSetString(level).At("LEVEL")
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
		l.v.MustSetString(traceID).At("TRACE_ID")
	}
	history := tracelog.TraceIDStack(ctx)
	if len(history) > 0 {
		l.v.MustSet(history).At("TRACE_ID_STACK")
	}
}

func (l *logger) setCallerStacks() {
	callers := caller.GetAllCallers()
	callers = callers[3:]
	l.v.MustSet(callers).At("STACK")
}

func (l *logger) String() string {
	if l.formatting {
		msg := fmt.Sprintf(l.format, l.args...)
		l.v.MustSetString(msg).At("TEXT")
	} else if len(l.args) > 0 {
		msg := fmt.Sprint(l.args...)
		l.v.MustSetString(msg).At("TEXT")
	}

	const skip = 9 // 这个值要跟随版本调试
	caller := caller.GetCaller(skip)
	fileParts := strings.Split(string(caller.File), "/")
	if len(fileParts) > 3 {
		f := strings.Join(fileParts[len(fileParts)-3:], "/")
		l.v.MustSetString(f).At("FILE")
	} else {
		l.v.MustSetString(string(caller.File)).At("FILE")
	}

	l.v.MustSetInt(caller.Line).At("LINE")
	l.v.MustSetString(caller.Func.Base()).At("FUNC")

	s := l.v.MustMarshalString(
		jsonvalue.OptKeySequence(internal.keySequence),
		jsonvalue.OptEscapeSlash(false),
		jsonvalue.OptUTF8(),
	)
	return s
}
