package redcache

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache 泛型化封装一个基于 Redis string 类型的任意类型存储
type Cache[T any] interface {
	Get(key string) (T, bool)
	Set(key string, value T) bool
	Del(key string)

	SetWithExpire(key string, value T, ttl time.Duration) bool
	GetWithStatus(key string) (T, CachedStatus)

	GetWithLoad(ctx context.Context, key string) (T, error)
	GetWithCustomLoad(ctx context.Context, key string, customLoad LoadFunc[T], ttl time.Duration) (T, error)

	MGetWithLoad(ctx context.Context, keys []string) (map[string]T, error)
	MGetWithCustomLoad(ctx context.Context, keys []string, customLoad MLoadFunc[T], ttl time.Duration) (map[string]T, error)
}

// RedisGetter 返回动态获取 Redis 客户端的函数
type RedisGetter func(ctx context.Context) (redis.StringCmdable, error)

// New 新建一个基于 Redis string 类型的任意类型存储
func New[T any](redisGetter RedisGetter, opts ...Option[T]) Cache[T] {
	if redisGetter == nil {
		redisGetter = func(ctx context.Context) (redis.StringCmdable, error) {
			return nil, errors.New("redis getter is nil")
		}
	}
	return &impl[T]{
		cli:  redisGetter,
		opts: mergeOptions[T](opts),
	}
}

// CachedStatus cache status
type CachedStatus int

const (
	// CacheNotExist cache data does not exist
	CacheNotExist CachedStatus = 1
	// CacheExist cache data exists
	CacheExist CachedStatus = 2
	// CacheExpire cache data exists but has expired. You can choose whether to use it
	CacheExpire CachedStatus = 3
)

// LoadFunc loads the value data corresponding to the key and is used to fill the cache
type LoadFunc[T any] func(ctx context.Context, key string) (T, error)

// MLoadFunc loads the value data of multiple keys in batches to fill the cache
type MLoadFunc[T any] func(ctx context.Context, keys []string) (map[string]T, error)

// ItemCallBackFunc callback function triggered when the element expires/deletes
type ItemCallBackFunc[T any] func(Item[T])

// Item is the element that triggered the callback event
type Item[T any] struct {
	Key   string
	Value T
}
