// Package redis 提供 trpc-database 的 goredis 工具
package redis

import (
	"context"

	"github.com/Andrew-M-C/trpc-go-utils/client/internal"
	redis "github.com/redis/go-redis/v9"
	"trpc.group/trpc-go/trpc-database/goredis"
	"trpc.group/trpc-go/trpc-go/client"
)

// RedisGetterFunc 动态获取 go-redis 客户端的函数
type RedisGetterFunc func(context.Context) (redis.UniversalClient, error)

// RedisGetter 返回动态获取 Redis 客户端的函数
func RedisGetter(name string, opts ...client.Option) RedisGetterFunc {
	return func(ctx context.Context) (redis.UniversalClient, error) {
		return buffer.GetClient(name, opts)
	}
}

var buffer *internal.ClientBuffer[redis.UniversalClient] = internal.NewClientBuffer(
	"amc.utils.redis.", newRedis, closeRedis,
)

func closeRedis(cli redis.UniversalClient) error {
	return cli.Close()
}

func newRedis(name string, opts ...client.Option) (redis.UniversalClient, error) {
	return goredis.New(name, opts...)
}
