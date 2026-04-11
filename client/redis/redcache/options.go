package redcache

import (
	"context"
	"encoding/json"

	"github.com/Andrew-M-C/go.util/unsafe"
)

// Option 表示可选选项
type Option[T any] func(*options[T])

type options[T any] struct {
	load  LoadFunc[T]
	mLoad MLoadFunc[T]

	marshaler   func(T) (string, error)
	unmarshaler func(string) (T, error)
}

// WithLoad 设置加载函数
func WithLoad[T any](load LoadFunc[T]) Option[T] {
	return func(o *options[T]) {
		o.load = load
	}
}

// WithMLoad 设置批量加载函数
func WithMLoad[T any](mLoad MLoadFunc[T]) Option[T] {
	return func(o *options[T]) {
		o.mLoad = mLoad
	}
}

// WithMarshaler 设置序列化函数
func WithMarshaler[T any](marshaler func(T) (string, error)) Option[T] {
	return func(o *options[T]) {
		o.marshaler = marshaler
	}
}

// WithUnmarshaler 设置反序列化函数
func WithUnmarshaler[T any](unmarshaler func(string) (T, error)) Option[T] {
	return func(o *options[T]) {
		o.unmarshaler = unmarshaler
	}
}

func mergeOptions[T any](opts []Option[T]) *options[T] {
	o := &options[T]{}
	for _, f := range opts {
		if f != nil {
			f(o)
		}
	}

	o.checkLoadFunc()
	o.checkMLoadFunc()
	o.checkMarshaler()
	o.checkUnmarshaler()

	return o
}

func (o *options[T]) checkLoadFunc() {
	if o.load != nil {
		return
	}
	o.load = func(ctx context.Context, key string) (T, error) {
		var res T
		return res, nil
	}
}

func (o *options[T]) checkMLoadFunc() {
	if o.mLoad != nil {
		return
	}
	o.mLoad = func(ctx context.Context, keys []string) (map[string]T, error) {
		res := make(map[string]T, len(keys))
		for _, k := range keys {
			var v T
			res[k] = v
		}
		return res, nil
	}
}

func (o *options[T]) checkMarshaler() {
	if o.marshaler != nil {
		return
	}
	o.marshaler = func(v T) (string, error) {
		b, err := json.Marshal(v)
		if err != nil {
			return "", err
		}
		return unsafe.BtoS(b), nil
	}
}

func (o *options[T]) checkUnmarshaler() {
	if o.unmarshaler != nil {
		return
	}
	o.unmarshaler = func(s string) (T, error) {
		var res T
		b := unsafe.StoB(s)
		err := json.Unmarshal(b, &res)
		return res, err
	}
}
