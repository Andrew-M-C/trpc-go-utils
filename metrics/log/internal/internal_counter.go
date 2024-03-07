// Package internal log 内部工具
package internal

import "sync"

// Counter 实现计数类型值, 实际上也可以实现 gauge 值
type Counter struct {
	lck sync.Mutex

	min, max, sum float64
	count         int
}

func NewCounter() *Counter {
	return &Counter{}
}

// IncrBy 自增
func (c *Counter) IncrBy(val float64) {
	c.lck.Lock()
	defer c.lck.Unlock()

	if c.count == 0 {
		c.min, c.max, c.sum, c.count = val, val, val, 1
		return
	}

	if val < c.min {
		c.min = val
	}
	if val > c.max {
		c.max = val
	}
	c.sum += val
	c.count++
}

// Statistic 统计结果
func (c *Counter) Statistic() (min, max, sum, avg float64, count int) {
	c.lck.Lock()
	defer c.lck.Unlock()

	if c.count == 0 {
		return
	}

	min = c.min
	max = c.max
	sum = c.sum
	avg = c.sum / float64(c.count)
	count = c.count
	return
}
