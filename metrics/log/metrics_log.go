// Package log 简单统计每分钟的指标数据, 然后写入日志
package log

import (
	"strings"
	"time"

	"github.com/Andrew-M-C/go.util/sync"
	"github.com/Andrew-M-C/trpc-go-utils/metrics/log/internal"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/metrics"
)

const (
	// MetricsSinkName 表示上报器名称
	MetricsSinkName = "log"
	// 在 plugin 配置中的 type
	PluginType = "metrics"
	// 在 plugin 配置中的 name
	PluginName = "log"
)

// 注册 MySQL metrics 功能
func RegisterMetricsMySQL() {
	impl := &sinkImpl{
		logger: log.Infof,
		notify: make(chan metrics.Metrics, 1024),
	}
	impl.resetMetrics()
	impl.registerPlugin()
	go impl.routine()

	metrics.RegisterMetricsSink(impl)
}

type sinkImpl struct {
	server  string // 服务名
	logger  func(string, ...any)
	metrics sync.Map[string, *internal.Counter]
	notify  chan metrics.Metrics
}

func (*sinkImpl) Name() string {
	return MetricsSinkName
}

func (impl *sinkImpl) Report(rec metrics.Record, _ ...metrics.Option) error {
	if len(rec.GetDimensions()) == 0 {
		// 单纬度指标
		for _, m := range rec.GetMetrics() {
			impl.reportSingleDimensionMetrics(m)
		}
		return nil
	}
	// TODO: 多维度
	return nil
}

func (impl *sinkImpl) resetMetrics() {
	impl.metrics = sync.NewMap[string, *internal.Counter]()
}

func (impl *sinkImpl) reportSingleDimensionMetrics(m *metrics.Metrics) {
	if strings.HasPrefix(m.Name(), "trpc.Log") {
		return
	}
	impl.notify <- *m
}

func (impl *sinkImpl) routine() {
	sec := time.Now().Second()
	tick := time.NewTicker(time.Duration(60-sec) * time.Second)
	firstTick := true

	for {
		select {
		case <-tick.C:
			if firstTick {
				firstTick = false
				tick.Reset(time.Minute)
			}
			impl.statisticAndLog()

		case m := <-impl.notify:
			counter, _ := impl.metrics.LoadOrStore(m.Name(), internal.NewCounter())
			counter.IncrBy(m.Value())
		}
	}
}

func (impl *sinkImpl) statisticAndLog() {
	m := impl.metrics
	impl.resetMetrics()

	m.Range(func(name string, value *internal.Counter) bool {
		min, max, sum, avg, count := value.Statistic()
		impl.logger(
			"metrics | %s | %s | min %g, max %g, sum %g, avg %g, count %d",
			impl.server, name, min, max, sum, avg, count,
		)
		return true
	})
}
