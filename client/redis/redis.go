// Package redis 提供 trpc-database 的 go-redis 工具
package redis

import (
	"context"

	"github.com/Andrew-M-C/trpc-go-utils/client/buffer"
	redis "github.com/redis/go-redis/v9"
	"trpc.group/trpc-go/trpc-database/goredis"
	"trpc.group/trpc-go/trpc-go/client"
)

// ClientGetter 返回动态获取 Redis 客户端的函数
func ClientGetter(
	name string, opts ...client.Option,
) func(context.Context) (redis.UniversalClient, error) {
	return func(ctx context.Context) (redis.UniversalClient, error) {
		return buff.GetClient(name, opts)
	}
}

var buff *buffer.ClientBuffer[redis.UniversalClient] = buffer.NewClientBuffer(
	"amc.utils.redis.", newRedis, closeRedis,
)

func closeRedis(cli redis.UniversalClient) error {
	return cli.Close()
}

func newRedis(name string, opts ...client.Option) (redis.UniversalClient, error) {
	return goredis.New(name, opts...)
}
