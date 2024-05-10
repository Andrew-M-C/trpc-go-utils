// Package internal 实现 client 包的一些通用逻辑
package internal

import (
	"time"

	"trpc.group/trpc-go/trpc-go/client"
)

// GetClientTarget 获取 client 的目标配置值
func GetClientTarget(name string) (string, time.Duration) {
	cnf := client.Config(name)
	if cnf == nil {
		return "", 0
	}
	timeout := time.Duration(cnf.Timeout) * time.Millisecond
	if timeout > time.Minute {
		timeout = time.Minute // 最多一分钟, 不能再多了
	}
	return cnf.Target, timeout
}
