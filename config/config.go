// Package config 提供动态配置的接口
package config

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Andrew-M-C/go.util/unsafe"
	"trpc.group/trpc-go/trpc-go/config"
	"trpc.group/trpc-go/trpc-go/log"
)

// Encoding 定义了编解码格式
type Encoding string

const (
	JSON Encoding = "json"
	YAML Encoding = "yaml"
	TEXT Encoding = "text"
)

const (
	// ErrConfig 表示 config 错误
	ErrConfig = E("amc config error")
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
		return fmt.Errorf("%w: config value holder is nil", ErrConfig)
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
		return nil, fmt.Errorf("%w: api 参数为空", ErrConfig)
	}

	unmarshaler := internal.unmarshalerByName[encoding]
	if unmarshaler == nil {
		return nil, fmt.Errorf("%w: encoding '%s' not found", ErrConfig, encoding)
	}
	return unmarshaler, nil
}

func updateValue[T any](key, value string, unmarshaler unmarshaler, holder **T) error {
	data := new(T)
	if err := unmarshaler.Unmarshal([]byte(value), &data); err != nil {
		return err
	}

	log.Debugf("%s 配置 '%s' 更新, 原始数据: '%s'", logPrefix, key, stringer{value})
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
	if conf == nil {
		return nil, nil, fmt.Errorf("%w: api 无法获取配置, 请检查是否相应的配置未注册", ErrConfig)
	}
	if res, err := conf.Get(ctx, key); err != nil {
		return nil, nil, fmt.Errorf("%w: 获取配置失败 (%v)", ErrConfig, err)
	} else if err := updateValue(key, res.Value(), unmarshaler, &holder); err != nil {
		return nil, nil, fmt.Errorf("%w: decode 数据配置失败 (%v)", ErrConfig, err)
	}

	// 监听并更新后续变化值
	notify, err := conf.Watch(ctx, key)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: watch 配置 key '%s' 失败 (%v)", ErrConfig, key, err)
	}

	ch := make(chan *T)
	go func() {
		for res := range notify {
			holder := new(T)
			if err := updateValue(key, res.Value(), unmarshaler, &holder); err != nil {
				count(fmt.Sprintf("update.%s.fail", key))
				log.Errorf("%s 更新配置 key '%s' 失败: %v", logPrefix, key, err)
				continue
			}
			count(fmt.Sprintf("update.%s.succ", key))
			ch <- holder
		}
	}()

	count(fmt.Sprintf("initialize.%s.succ", key))
	return holder, ch, nil
}

type stringer []string

func (s stringer) String() string {
	b, _ := json.Marshal(s)
	return unsafe.BtoS(b)
}
