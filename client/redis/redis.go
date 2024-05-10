// Package redis 提供 trpc-database 的 goredis 工具
package redis

import (
	"context"
	"fmt"
	"time"

	syncutil "github.com/Andrew-M-C/go.util/sync"
	util "github.com/Andrew-M-C/trpc-go-utils/client/internal"
	redis "github.com/redis/go-redis/v9"
	"trpc.group/trpc-go/trpc-database/goredis"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/metrics"
)

// RedisGetterFunc 动态获取 go-redis 客户端的函数
type RedisGetterFunc func(context.Context) (redis.UniversalClient, error)

// RedisGetter 返回动态获取 Redis 客户端的函数
func RedisGetter(name string, opts ...client.Option) RedisGetterFunc {
	return func(ctx context.Context) (redis.UniversalClient, error) {
		return getClient(ctx, name)
	}
}

func getClient(_ context.Context, name string, opts ...client.Option) (redis.UniversalClient, error) {
	target, newTimeout := util.GetClientTarget(name)
	if target == "" {
		// 没有配置, 如果历史 client 存在的话返回历史 client, 如果没有的话就只能返回错误了
		cli, exist := internal.clients.Load(name)
		if !exist {
			return nil, fmt.Errorf("client with name '%s' not configured", name)
		}
		return cli.client, nil
	}

	// 判断一下配置是否发生了变化
	if prev, exist := internal.clients.Load(name); !exist {
		// should refresh
	} else if prev.target != target {
		// should refresh
	} else {
		return prev.client, nil
	}

	// 需要刷新配置, 返回
	newClient, err := goredis.New(name, opts...)
	if err != nil {
		return nil, err
	}

	count("clientUpdate.cnt")

	newClientCtx := &clientContext{
		client:  newClient,
		target:  target,
		timeout: newTimeout,
	}
	prevClientCtx, loaded := internal.clients.Swap(name, newClientCtx)
	if loaded {
		go func() {
			time.Sleep(prevClientCtx.timeout)
			if err := prevClientCtx.client.Close(); err != nil {
				count("clientDestroy.fail")
				log.Errorf("close redis %v (%+v) failed: '%v'", name, prevClientCtx.target, err)
				return
			}
			count("clientDestroy.succ")
			log.Errorf("close previous redis %v (%+v) failed: '%v'", name, prevClientCtx.target, err)
		}()
	}

	return newClient, nil
}

type clientContext struct {
	client  redis.UniversalClient
	target  string
	timeout time.Duration
}

var internal = struct {
	clients syncutil.Map[string, *clientContext]
}{
	clients: syncutil.NewMap[string, *clientContext](),
}

func count(name string) {
	metrics.IncrCounter("amc.utils.redis."+name, 1)
}
