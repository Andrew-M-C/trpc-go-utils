// Package internal 实现 client 包的一些通用逻辑
package internal

import (
	"fmt"
	"time"

	syncutil "github.com/Andrew-M-C/go.util/sync"
	"github.com/Andrew-M-C/trpc-go-utils/metrics"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/log"
)

// ClientBuffer client 缓存池
type ClientBuffer[T any] struct {
	clients syncutil.Map[string, *clientContext[T]]
	mPrefix string
	newer   func(string, ...client.Option) (T, error)
	closer  func(T) error
}

// NewClientBuffer 新建一个 client 缓存池
func NewClientBuffer[T any](
	metricsPrefix string,
	newer func(string, ...client.Option) (T, error),
	closer func(T) error,
) *ClientBuffer[T] {
	b := &ClientBuffer[T]{
		clients: syncutil.NewMap[string, *clientContext[T]](),
		mPrefix: metricsPrefix,
		newer:   newer,
		closer:  closer,
	}
	return b
}

type clientContext[T any] struct {
	client  T
	target  string
	timeout time.Duration
}

func (b *ClientBuffer[T]) count(name string) {
	metrics.IncrCounter(b.mPrefix+name, 1)
}

func (b *ClientBuffer[T]) GetClient(
	name string, opts []client.Option,
) (client T, err error) {
	target, newTimeout := getClientTarget(name)
	if target == "" {
		// 没有配置, 如果历史 client 存在的话返回历史 client, 如果没有的话就只能返回错误了
		cli, exist := b.clients.Load(name)
		if !exist {
			err = fmt.Errorf("client with name '%s' not configured", name)
			return
		}
		return cli.client, nil
	}

	// 判断一下配置是否发生了变化
	if prev, exist := b.clients.Load(name); !exist {
		// should refresh
	} else if prev.target != target {
		// should refresh
	} else {
		return prev.client, nil
	}

	b.count("clientUpdate.cnt")

	newClient, err := b.newer(name, opts...)
	if err != nil {
		return
	}
	newClientCtx := &clientContext[T]{
		client:  newClient,
		target:  target,
		timeout: newTimeout,
	}
	prevClientCtx, loaded := b.clients.Swap(name, newClientCtx)
	if !loaded {
		return newClient, nil
	}

	go func() {
		time.Sleep(prevClientCtx.timeout)
		if err := b.closer(prevClientCtx.client); err != nil {
			b.count("clientDestroy.fail")
			log.Errorf("close client %v (%+v) failed: '%v'", name, prevClientCtx.target, err)
			return
		}
		b.count("clientDestroy.succ")
		log.Errorf("close previous redis %v (%+v) success", name, prevClientCtx.target)
	}()

	return newClient, nil
}

// getClientTarget 获取 client 的目标配置值
func getClientTarget(name string) (string, time.Duration) {
	cnf := client.Config(name)
	if cnf == nil {
		return "", 0
	}
	timeout := time.Duration(cnf.Timeout) * time.Millisecond
	if timeout > time.Minute {
		timeout = time.Minute // 最多一分钟, 不能再多了
	}
	return cnf.Target, timeout
}
