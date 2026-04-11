package redcache

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

type impl[T any] struct {
	cli  RedisGetter
	opts *options[T]
}

func (i *impl[T]) Get(key string) (T, bool) {
	ctx := context.Background()
	cli, err := i.cli(ctx)
	if err != nil {
		var zero T
		return zero, false
	}
	s, err := cli.Get(ctx, key).Result()
	if err != nil {
		var zero T
		return zero, false
	}
	v, err := i.opts.unmarshaler(s)
	if err != nil {
		var zero T
		return zero, false
	}
	return v, true
}

func (i *impl[T]) Set(key string, value T) bool {
	return i.setWithCtx(context.Background(), key, value, 0)
}

func (i *impl[T]) Del(key string) {
	ctx := context.Background()
	cli, err := i.cli(ctx)
	if err != nil {
		return
	}
	cli.GetDel(ctx, key)
}

func (i *impl[T]) SetWithExpire(key string, value T, ttl time.Duration) bool {
	return i.setWithCtx(context.Background(), key, value, ttl)
}

func (i *impl[T]) GetWithStatus(key string) (T, CachedStatus) {
	ctx := context.Background()
	cli, err := i.cli(ctx)
	if err != nil {
		var zero T
		return zero, CacheNotExist
	}
	s, err := cli.Get(ctx, key).Result()
	if err != nil {
		var zero T
		return zero, CacheNotExist
	}
	v, err := i.opts.unmarshaler(s)
	if err != nil {
		var zero T
		return zero, CacheNotExist
	}
	return v, CacheExist
}

func (i *impl[T]) GetWithLoad(ctx context.Context, key string) (T, error) {
	return i.GetWithCustomLoad(ctx, key, i.opts.load, 0)
}

func (i *impl[T]) GetWithCustomLoad(ctx context.Context, key string, customLoad LoadFunc[T], ttl time.Duration) (T, error) {
	cli, err := i.cli(ctx)
	if err == nil {
		s, getErr := cli.Get(ctx, key).Result()
		if getErr == nil {
			if v, unmarshalErr := i.opts.unmarshaler(s); unmarshalErr == nil {
				return v, nil
			}
		} else if !errors.Is(getErr, redis.Nil) {
			// non-nil Redis error: fall through to load
			_ = getErr
		}
	}
	return i.loadAndSet(ctx, key, customLoad, ttl)
}

func (i *impl[T]) MGetWithLoad(ctx context.Context, keys []string) (map[string]T, error) {
	return i.MGetWithCustomLoad(ctx, keys, i.opts.mLoad, 0)
}

func (i *impl[T]) MGetWithCustomLoad(ctx context.Context, keys []string, customLoad MLoadFunc[T], ttl time.Duration) (map[string]T, error) {
	if len(keys) == 0 {
		return map[string]T{}, nil
	}

	result := make(map[string]T, len(keys))
	missing := make([]string, 0, len(keys))

	cli, err := i.cli(ctx)
	if err != nil {
		missing = keys
	} else {
		vals, mgetErr := cli.MGet(ctx, keys...).Result()
		if mgetErr != nil {
			missing = keys
		} else {
			for idx, val := range vals {
				s, ok := val.(string)
				if !ok {
					missing = append(missing, keys[idx])
					continue
				}
				v, unmarshalErr := i.opts.unmarshaler(s)
				if unmarshalErr != nil {
					missing = append(missing, keys[idx])
					continue
				}
				result[keys[idx]] = v
			}
		}
	}

	if len(missing) == 0 {
		return result, nil
	}

	loaded, loadErr := customLoad(ctx, missing)
	if loadErr != nil {
		return result, loadErr
	}

	for k, v := range loaded {
		result[k] = v
		i.setWithCtx(ctx, k, v, ttl)
	}

	return result, nil
}

func (i *impl[T]) loadAndSet(ctx context.Context, key string, load LoadFunc[T], ttl time.Duration) (T, error) {
	v, err := load(ctx, key)
	if err != nil {
		var zero T
		return zero, err
	}
	i.setWithCtx(ctx, key, v, ttl)
	return v, nil
}

func (i *impl[T]) setWithCtx(ctx context.Context, key string, value T, ttl time.Duration) bool {
	cli, err := i.cli(ctx)
	if err != nil {
		return false
	}
	s, marshalErr := i.opts.marshaler(value)
	if marshalErr != nil {
		return false
	}
	return cli.Set(ctx, key, s, ttl).Err() == nil
}
