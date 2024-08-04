package localcache

import (
	"context"
	"time"

	"trpc.group/trpc-go/trpc-database/localcache"
)

// Cache 泛型化封装 trpc 官方的 localcache.Cache 类型
type Cache[T any] interface {
	Get(key string) (T, bool)
	Set(key string, value T) bool
	Del(key string)
	Len() int
	Clear()
	Close()

	SetWithExpire(key string, value T, ttl time.Duration) bool
	GetWithStatus(key string) (T, localcache.CachedStatus)

	GetWithLoad(ctx context.Context, key string) (T, error)
	GetWithCustomLoad(ctx context.Context, key string, customLoad LoadFunc[T], ttl time.Duration) (T, error)

	MGetWithLoad(ctx context.Context, keys []string) (map[string]T, error)
	MGetWithCustomLoad(ctx context.Context, keys []string, customLoad MLoadFunc[T], ttl time.Duration) (map[string]T, error)
}

// New generate a cache object
func New[T any](opts ...Option) Cache[T] {
	cache := localcache.New(opts...)
	c := &cacheImpl[T]{
		cache: cache,
	}
	return c
}

const (
	// CacheNotExist cache data does not exist
	CacheNotExist = localcache.CacheNotExist
	// CacheExist cache data exists
	CacheExist = localcache.CacheExist
	// CacheExpire cache data exists but has expired. You can choose whether to use it
	CacheExpire = localcache.CacheExpire
)

const (
	// ItemDelete Triggered when actively deleted/expired
	ItemDelete = localcache.ItemDelete
	// ItemLruDel LRU triggered deletion
	ItemLruDel = localcache.ItemLruDel
)

// LoadFunc loads the value data corresponding to the key and is used to fill the cache
type LoadFunc[T any] func(ctx context.Context, key string) (T, error)

// MLoadFunc loads the value data of multiple keys in batches to fill the cache
type MLoadFunc[T any] func(ctx context.Context, keys []string) (map[string]T, error)

// ItemCallBackFunc callback function triggered when the element expires/deletes
type ItemCallBackFunc[T any] func(Item[T])

// Item is the element that triggered the callback event
type Item[T any] struct {
	Flag  localcache.ItemFlag
	Key   string
	Value T
}

// Option parameter tool function
type Option = localcache.Option

// WithCapacity sets the maximum number of keys
func WithCapacity(capacity int) Option {
	return localcache.WithCapacity(capacity)
}

func WithDelay(duration time.Duration) Option {
	return localcache.WithDelay(durationToSec(duration))
}

func WithExpiration(ttl time.Duration) Option {
	return localcache.WithExpiration(durationToSec(ttl))
}

func WithLoad[T any](f LoadFunc[T]) Option {
	return localcache.WithLoad(func(ctx context.Context, key string) (any, error) {
		return f(ctx, key)
	})
}

func WithMLoad[T any](f MLoadFunc[T]) Option {
	return localcache.WithMLoad(func(ctx context.Context, keys []string) (map[string]any, error) {
		m, err := f(ctx, keys)
		if err != nil {
			return nil, err
		}
		res := make(map[string]any, len(m))
		for k, v := range m {
			res[k] = v
		}
		return res, nil
	})
}

func WithOnDel[T any](delCallBack ItemCallBackFunc[T]) Option {
	return localcache.WithOnDel(func(i *localcache.Item) {
		if i == nil {
			return
		}
		v, ok := i.Value.(T)
		if !ok {
			return
		}
		delCallBack(Item[T]{
			Flag:  i.Flag,
			Key:   i.Key,
			Value: v,
		})
	})
}

func WithOnExpire[T any](expireCallback ItemCallBackFunc[T]) Option {
	return localcache.WithOnExpire(func(i *localcache.Item) {
		if i == nil {
			return
		}
		v, ok := i.Value.(T)
		if !ok {
			return
		}
		expireCallback(Item[T]{
			Flag:  i.Flag,
			Key:   i.Key,
			Value: v,
		})
	})
}

func WithSettingTimeout(t time.Duration) Option {
	return localcache.WithSettingTimeout(t)
}

func WithSyncDelFlag(flag bool) Option {
	return localcache.WithSyncDelFlag(flag)
}
