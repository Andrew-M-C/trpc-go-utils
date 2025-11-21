// Package file 封装本地文件配置 API, 支持热更新
package file

import (
	"context"
	"errors"
	"sync"

	syncutil "github.com/Andrew-M-C/go.util/sync"
	"trpc.group/trpc-go/trpc-go/config"
)

func init() {
	registerPlugin()
}

var internal = struct {
	lck      sync.Mutex
	configs  []pluginConfigItem
	watchers map[string]*fileListener
	cache    syncutil.Map[string, string]
}{
	watchers: make(map[string]*fileListener),
	cache:    syncutil.NewMap[string, string](),
}

// 实现 config.API
type API struct{}

// GetConfig 实现 config.API
func (a API) GetConfig() config.KVConfig {
	return a
}

// Put 实现 config.KV 接口
func (API) Put(
	ctx context.Context, key, val string, opts ...config.Option,
) error {
	return errors.New("file config does not support put")
}

// Get 实现 config.KV 接口
func (API) Get(
	ctx context.Context, key string, opts ...config.Option,
) (config.Response, error) {
	res, err := readFile(key)
	if err == nil { // 注意这里是 == nil
		return res, nil
	}

	v, exist := internal.cache.Load(key)
	if !exist {
		return nil, err
	}
	res = &content{
		value: v,
		event: config.EventTypeNull,
	}
	return res, nil
}

func (API) Del(
	ctx context.Context, key string, opts ...config.Option,
) error {
	return errors.New("file config does not support del")
}

// Watch 实现 config.Watcher 接口
func (API) Watch(
	ctx context.Context, key string, opts ...config.Option,
) (<-chan config.Response, error) {
	internal.lck.Lock()
	defer internal.lck.Unlock()

	if w, ok := internal.watchers[key]; ok {
		return w.ch, nil
	}

	w, err := newWatcher(key)
	if err != nil {
		return nil, err
	}
	internal.watchers[key] = w
	return w.ch, nil
}

// NAme 实现 config.KVConfig 接口
func (API) Name() string {
	return "file"
}
