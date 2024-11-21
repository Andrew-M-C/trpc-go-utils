// Package kafka 提供 trpc-database 的 Kafka 工具
package kafka

import (
	"context"

	"trpc.group/trpc-go/trpc-database/kafka"
	"trpc.group/trpc-go/trpc-go/client"
)

// ClientGetter 返回动态获取 Kafka 客户端的函数
func ClientGetter(
	name string, opts ...client.Option,
) func(context.Context) (kafka.Client, error) {
	return func(context.Context) (kafka.Client, error) {
		return kafka.NewClientProxy(name, opts...), nil
	}
}
