package localcache

import (
	"context"
	"time"

	"trpc.group/trpc-go/trpc-database/localcache"
)

type cacheImpl[T any] struct {
	cache localcache.Cache
}

func (c *cacheImpl[T]) Get(key string) (res T, ok bool) {
	v, ok := c.cache.Get(key)
	if !ok {
		return res, false
	}
	res, ok = v.(T)
	return res, ok
}

func (c *cacheImpl[T]) Set(key string, value T) bool {
	return c.cache.Set(key, value)
}

func (c *cacheImpl[T]) Del(key string) {
	c.cache.Del(key)
}

func (c *cacheImpl[T]) Len() int {
	return c.cache.Len()
}

func (c *cacheImpl[T]) Clear() {
	c.cache.Clear()
}

func (c *cacheImpl[T]) Close() {
	c.cache.Close()
}

func (c *cacheImpl[T]) SetWithExpire(
	key string, value T, ttl time.Duration,
) bool {
	sec := durationToSec(ttl)
	return c.cache.SetWithExpire(key, value, sec)
}

func (c *cacheImpl[T]) GetWithStatus(
	key string,
) (res T, status localcache.CachedStatus) {
	v, status := c.cache.GetWithStatus(key)
	res, ok := v.(T)
	if !ok {
		status = CacheNotExist
	}
	return res, status
}

func (c *cacheImpl[T]) GetWithLoad(
	ctx context.Context, key string,
) (res T, err error) {
	v, err := c.cache.GetWithLoad(ctx, key)
	if err != nil {
		return res, err
	}
	res, _ = v.(T)
	return res, nil
}

func (c *cacheImpl[T]) GetWithCustomLoad(
	ctx context.Context, key string, customLoad LoadFunc[T], ttl time.Duration,
) (res T, err error) {
	fu := func(ctx context.Context, key string) (interface{}, error) {
		return customLoad(ctx, key)
	}
	v, err := c.cache.GetWithCustomLoad(ctx, key, fu, durationToSec(ttl))
	if err != nil {
		return res, err
	}
	res, _ = v.(T)
	return res, nil
}

func (c *cacheImpl[T]) MGetWithLoad(
	ctx context.Context, keys []string,
) (map[string]T, error) {
	m, err := c.cache.MGetWithLoad(ctx, keys)
	if err != nil {
		return nil, err
	}
	return anyMapToT[T](m), nil
}

func (c *cacheImpl[T]) MGetWithCustomLoad(
	ctx context.Context, keys []string, customLoad MLoadFunc[T], ttl time.Duration,
) (map[string]T, error) {
	fu := func(ctx context.Context, keys []string) (map[string]any, error) {
		m, err := customLoad(ctx, keys)
		if err != nil {
			return nil, err
		}
		return tMapToAny(m), nil
	}
	m, err := c.cache.MGetWithCustomLoad(ctx, keys, fu, durationToSec(ttl))
	if err != nil {
		return nil, err
	}
	return anyMapToT[T](m), nil
}

func durationToSec(d time.Duration) int64 {
	if d <= 0 {
		return 0
	}
	if d < time.Second {
		return 1
	}
	// 四舍五入
	return int64((d + 500*time.Millisecond) / time.Second)
}

func anyMapToT[T any](m map[string]any) map[string]T {
	res := make(map[string]T, len(m))
	for k, v := range m {
		if vt, ok := v.(T); ok {
			res[k] = vt
		}
	}
	return res
}

func tMapToAny[T any](m map[string]T) map[string]any {
	res := make(map[string]any, len(m))
	for k, v := range m {
		res[k] = v
	}
	return res
}
