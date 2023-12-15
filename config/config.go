// Package config 提供 trpc 启动配置的工具
package config

import (
	"fmt"

	"trpc.group/trpc-go/trpc-go/plugin"
)

// RegisterPlugin 注册 plugin 配置。请在 init 阶段调用或 NewServer 之前调用。
func RegisterPlugin[T any](typ, name string, receiver func(*T) error) {
	p := &pluginFactory[T]{
		typ:      typ,
		name:     name,
		receiver: receiver,
	}
	plugin.Register(name, p)
}

type pluginFactory[T any] struct {
	typ, name string
	receiver  func(*T) error
}

// Type 实现 plugin.Factory
func (p *pluginFactory[T]) Type() string {
	return p.typ
}

// Setup 实现 plugin.Factory
func (p *pluginFactory[T]) Setup(name string, decoder plugin.Decoder) error {
	c := new(T)
	if err := decoder.Decode(c); err != nil {
		return fmt.Errorf("decode config %s.%s error: '%w'", p.typ, p.name, err)
	}
	if p.receiver == nil {
		return fmt.Errorf("nil config receiver for %s.%s", p.typ, p.name)
	}
	return p.receiver(c)
}
