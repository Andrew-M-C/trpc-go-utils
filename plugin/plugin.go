// Package plugin 提供 trpc 启动配置中的 plugin 工具
package plugin

import (
	"fmt"

	"trpc.group/trpc-go/trpc-go/plugin"
)

// Register 注册 plugin 配置。请在 init 阶段调用或 NewServer 之前调用。
func Register[T any](typ, name string, receiver func(*T) error, opts ...Option) {
	p := &pluginFactory[T]{
		typ:      typ,
		name:     name,
		receiver: receiver,
		options:  mergeOptions(opts),
	}
	plugin.Register(name, p)
}

// Bind 将 plugins 配置与本地存储绑定
func Bind[T any](typ, name string, target *T) {
	Register(typ, name, func(t *T) error {
		*target = *t
		return nil
	})
}

type pluginFactory[T any] struct {
	typ, name string
	receiver  func(*T) error
	options   options
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

// DependsOn 实现 plugin.Depender
func (p *pluginFactory[T]) DependsOn() []string {
	return p.options.dependsOn
}

// FlexDependsOn 实现 plugin.FlexDepender
func (p *pluginFactory[T]) FlexDependsOn() []string {
	return p.options.flexDependsOn
}
