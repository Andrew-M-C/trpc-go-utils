package tracelog

import (
	"context"
	"fmt"
	"os"

	"github.com/Andrew-M-C/go.util/runtime/caller"
	"github.com/Andrew-M-C/trpc-go-utils/log"
)

var (
	DebugContextf = log.DebugContextf
	InfoContextf  = log.InfoContextf
	WarnContextf  = log.WarnContextf
	ErrorContextf = log.ErrorContextf

	DebugContext = log.DebugContext
	InfoContext  = log.InfoContext
	WarnContext  = log.WarnContext
	ErrorContext = log.ErrorContext

	Debugf = log.Debugf
	Infof  = log.Infof
	Warnf  = log.Warnf
	Errorf = log.Errorf

	Debug = log.Debug
	Info  = log.Info
	Warn  = log.Warn
	Error = log.Error
)

func getStackInfo() fmt.Stringer {
	callers := caller.GetAllCallers()
	if len(callers) > 2 {
		callers = callers[2:]
	}
	return ToJSON(callers)
}

// FatalContextf 打 fatal 级别日志并退出应用程序
func FatalContextf(ctx context.Context, f string, a ...any) {
	a = append(a, getStackInfo())
	log.FatalContextf(ctx, f+", caller stacks: '%v'", a...)
	os.Exit(-1)
}

// FatalContext 打 fatal 级别日志并退出应用程序
func FatalContext(ctx context.Context, a ...any) {
	a = append(a, "caller stacks: ", getStackInfo())
	log.FatalContext(ctx, a...)
	os.Exit(-1)
}

// Fatalf 打 fatal 级别日志并退出应用程序
func Fatalf(f string, a ...any) {
	a = append(a, getStackInfo())
	log.Fatalf(f+", caller stacks: '%v'", a...)
	os.Exit(-1)
}

// Fatal 打 fatal 级别日志并退出应用程序
func Fatal(a ...any) {
	a = append(a, "caller stacks: ", getStackInfo())
	log.Fatal(a...)
	os.Exit(-1)
}
