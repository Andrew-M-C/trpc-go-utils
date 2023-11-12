// Package mysql 简单使用 MySQL 轮询来维护服务的可用性
package mysql

import "time"

const (
	// 注册器名称
	SelectorName = "mysql"
	// trpc_go.yaml plugins 下的第一层 key
	PluginType = "registry"
	// trpc_go.yaml plugins 下的第二层 key
	PluginName = "mysql"
)

// 内部配置
const (
	// 心跳间隔
	heartbeatInterval = 2 * time.Second
	// 活跃检查限时
	liveTimeInterval = 5 * time.Second
)

func init() {
	initRegistry()
	initPlugin()
	initSelector()
}
