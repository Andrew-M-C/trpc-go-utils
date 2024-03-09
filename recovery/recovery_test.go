package recovery_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/Andrew-M-C/go.util/runtime/caller"
	"github.com/Andrew-M-C/trpc-go-utils/recovery"
	"github.com/smartystreets/goconvey/convey"
	"trpc.group/trpc-go/trpc-go/metrics"
)

var (
	cv = convey.Convey
	so = convey.So
	eq = convey.ShouldEqual

	notNil   = convey.ShouldNotBeNil
	notPanic = convey.ShouldNotPanic
)

func TestMain(m *testing.M) {
	_ = m.Run()
}

func TestCatchPanic(t *testing.T) {
	cv("WithContext", t, func() {
		var outerCtx context.Context
		type key struct{}
		const value = "hhhh"
		f := func() {
			defer recovery.CatchPanic(
				recovery.WithContext(context.WithValue(context.Background(), key{}, value)),
				recovery.WithCallback(func(ctx context.Context, _ any, _ []caller.Caller) {
					outerCtx = ctx
				}),
			)
			panic("some panic")
		}

		so(f, notPanic)
		so(outerCtx, notNil)
		so(outerCtx.Value(key{}), notNil)
		so(fmt.Sprint(outerCtx.Value(key{})), eq, value)
	})

	cv("WithErrorLog", t, func() {
		f := func() {
			defer recovery.CatchPanic(recovery.WithErrorLog())
			panic(errors.New("panic with error type"))
		}
		so(f, notPanic)
	})

	cv("WithMetrics", t, func() {
		const name = "test.recovery.panic"
		m := &testMetrics{}
		metrics.RegisterMetricsSink(m)
		f := func() {
			defer recovery.CatchPanic(recovery.WithMetrics(name))
			panic("some panic with metrics")
		}
		so(f, notPanic)
		so(len(m.metrics), eq, 1)
		so(m.metrics[0].Name(), eq, name)
		so(m.metrics[0].Value(), eq, float64(1))
	})
}

type testMetrics struct {
	metrics []*metrics.Metrics
}

func (*testMetrics) Name() string {
	return "test"
}

func (m *testMetrics) Report(rec metrics.Record, _ ...metrics.Option) error {
	m.metrics = append(m.metrics, rec.GetMetrics()...)
	return nil
}
