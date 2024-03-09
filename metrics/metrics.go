// Package metrics 提供 trpc metrics 功能的封装
package metrics

import (
	"golang.org/x/exp/constraints"
	"trpc.group/trpc-go/trpc-go/metrics"
)

// RealNumber 泛指所有实数
type RealNumber interface {
	constraints.Integer | constraints.Float
}

// IncrCounter increases counter key by value. Counters should accumulate values
func IncrCounter[N RealNumber](key string, value N) {
	metrics.IncrCounter(key, float64(value))
}

// SetGauge sets gauge key to value. An IGauge retains the last set value.
func SetGauge[N RealNumber](key string, value N) {
	metrics.SetGauge(key, float64(value))
}
