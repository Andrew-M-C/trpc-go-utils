// Package etcd 封装 etcd 配置 API
package etcd

import (
	"trpc.group/trpc-go/trpc-go/config"

	// etcd 配置支持
	_ "trpc.group/trpc-go/trpc-config-etcd"
)

// 实现 config.API
type API struct{}

// GetConfig 实现 config.API
func (API) GetConfig() config.KVConfig {
	return config.Get("etcd")
}
