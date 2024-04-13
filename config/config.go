// Package config 提供动态配置的接口
package config

import (
	"context"
	"errors"
	"fmt"

	"trpc.group/trpc-go/trpc-go/config"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/metrics"
)

// Encoding 定义了编解码格式
type Encoding string

const (
	JSON Encoding = "json"
	YAML Encoding = "yaml"
)

// API 定义配置配置 API
type API interface {
	GetConfig() config.KVConfig
}

// Bind 绑定一个远端配置和本地存储
func Bind[T any](
	ctx context.Context, api API, encoding Encoding, key string, holder **T,
) error {
	if holder == nil {
		return errors.New("config value holder is nil")
	}

	firstValue, watcher, err := Watch[T](ctx, api, encoding, key)
	if err != nil {
		return err
	}

	*holder = firstValue
	go func() {
		for v := range watcher {
			*holder = v
		}
	}()

	return nil
}

func validateAndGetUnmarshaler(api API, encoding Encoding) (unmarshaler, error) {
	if api == nil {
		return nil, errors.New("api is nil")
	}

	unmarshaler := internal.unmarshalerByName[encoding]
	if unmarshaler == nil {
		return nil, fmt.Errorf("encoding '%s' not found", encoding)
	}
	return unmarshaler, nil
}

func updateValue[T any](key, value string, unmarshaler unmarshaler, holder **T) error {
	data := new(T)
	if err := unmarshaler.Unmarshal([]byte(value), &data); err != nil {
		return err
	}

	log.Debugf("配置 '%s' 更新, 原始数据: '%s'", key, value)
	*holder = data
	return nil
}

// Watch 获取第一个配置并持续监听变化
func Watch[T any](
	ctx context.Context, api API, encoding Encoding, key string,
) (firstValue *T, watcher <-chan *T, err error) {
	unmarshaler, err := validateAndGetUnmarshaler(api, encoding)
	if err != nil {
		return nil, nil, err
	}

	// 获取一个最新值
	holder := new(T)
	conf := api.GetConfig()
	if res, err := conf.Get(ctx, key); err != nil {
		return nil, nil, fmt.Errorf("获取配置失败 (%w)", err)
	} else if err := updateValue(key, res.Value(), unmarshaler, &holder); err != nil {
		return nil, nil, fmt.Errorf("decode 数据配置失败 (%w)", err)
	}

	// 监听并更新后续变化值
	notify, err := conf.Watch(ctx, key)
	if err != nil {
		return nil, nil, fmt.Errorf("watch 配置 key '%s' 失败 (%w)", key, err)
	}

	ch := make(chan *T)
	go func() {
		for res := range notify {
			holder := new(T)
			if err := updateValue(key, res.Value(), unmarshaler, &holder); err != nil {
				metrics.Counter("utils.config.update.fail").Incr()
				log.Errorf("更新配置 key '%s' 失败: %v", key, err)
				continue
			}
			metrics.Counter("utils.config.update.succ").Incr()
			ch <- holder
		}
	}()

	metrics.Counter("utils.config.initialize.succ").Incr()
	return holder, ch, nil
}
