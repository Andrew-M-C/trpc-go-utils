// Package sqlx 提供 trpc-database 的 sqlx 工具
package sqlx

import (
	"context"

	"trpc.group/trpc-go/trpc-database/mysql"
	"trpc.group/trpc-go/trpc-go/client"
)

// MySQLGetterFunc 动态获取 MySQL 客户端的函数
type MySQLGetterFunc func(context.Context) (mysql.Client, error)

// MySQLGetter 返回动态获取 MySQL 客户端的函数
func MySQLGetter(name string, opts ...client.Option) MySQLGetterFunc {
	return func(ctx context.Context) (mysql.Client, error) {
		// trpc mysql 的实现本身就自带了动态 client, 因此可以直接返回
		return mysql.NewUnsafeClient(name, opts...), nil
	}
}
