// Package sqlx 提供 trpc-database 的 sqlx 工具
package sqlx

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
	"trpc.group/trpc-go/trpc-database/mysql"
	"trpc.group/trpc-go/trpc-go/client"
)

// ClientGetter 返回动态获取 MySQL 客户端的函数
func ClientGetter(
	name string, opts ...client.Option,
) func(context.Context) (Client, error) {
	return func(ctx context.Context) (Client, error) {
		// trpc mysql 的实现本身就自带了动态 client, 因此可以直接返回
		cli := mysql.NewUnsafeClient(name, opts...)
		return &clientWrapper{db: cli}, nil
	}
}

// TxFunc 表示一次 transaction 的执行, 接收 context 和 sqlx.Tx 作为参数
type TxFunc func(context.Context, *sqlx.Tx) error

// Client 简化 sqlx 接口, 尽量
type Client interface {
	QueryContext(ctx context.Context, next mysql.NextFunc, query string, args ...any) error
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	SelectContext(ctx context.Context, dest any, query string, args ...any) error
	TransactionContext(ctx context.Context, fn TxFunc) error
}

type clientWrapper struct {
	db mysql.Client
}

func (c *clientWrapper) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return c.db.Exec(ctx, query, args...)
}

func (c *clientWrapper) SelectContext(ctx context.Context, dest any, query string, args ...any) error {
	return c.db.Select(ctx, dest, query, args...)
}

func (c *clientWrapper) TransactionContext(ctx context.Context, fn TxFunc) error {
	return c.db.Transactionx(ctx, func(tx *sqlx.Tx) error {
		return fn(ctx, tx)
	})
}

func (c *clientWrapper) QueryContext(ctx context.Context, next mysql.NextFunc, query string, args ...any) error {
	return c.db.Query(ctx, next, query, args...)
}

// SqlxWrapper 包装 sqlx.DB 并实现 Client 接口, 这主要是为了方便不需要启动 trpc 框架、
// 直接传入 sqlx.DB 的场景, 比如单元测试。
type SqlxWrapper struct {
	*sqlx.DB
}

var _ Client = (*SqlxWrapper)(nil)

func (db *SqlxWrapper) QueryContext(ctx context.Context, next mysql.NextFunc, query string, args ...any) error {
	rows, err := db.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		if err := next(rows); err != nil {
			return err
		}
	}
	return nil
}

func (db *SqlxWrapper) TransactionContext(ctx context.Context, fn TxFunc) error {
	tx := db.DB.MustBegin().Unsafe()

	if err := fn(ctx, tx); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}
