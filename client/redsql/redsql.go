// Package redsql 使用 MySQL 模拟实现 Redis 的能力
package redsql

import (
	"context"
	"time"

	"github.com/Andrew-M-C/trpc-go-utils/client/sqlx"
	"github.com/redis/go-redis/v9"
	"trpc.group/trpc-go/trpc-go/client"
)

// RedSQL 表示一个 Redis 模拟实现
//
// 但请特别注意:
//   - LCS 方法不支持
//   - GetSet 方法已废弃
//   - key 长度限制为 512 字节
//   - value 大小限制为 65535 字节
type RedSQL interface {
	redis.StringCmdable
}

const (
	// DefaultTableName 为未指定 Options.TableName 时使用的默认 KV 表名。
	DefaultTableName = "t_redsql_kv"

	defaultExpiryInterval = 60 * time.Second
	defaultExpiryJitter   = 10 * time.Second
)

// Options 配置 RedSQL 客户端行为
type Options struct {
	// TableName 可选，指定存储 KV 数据的表名；为空时使用 DefaultTableName。
	TableName string

	// ExpiryInterval 指定后台清理过期 key 的轮询间隔，默认为 60s
	ExpiryInterval time.Duration

	// ExpiryJitter 指定启动时的随机抖动范围，防止多个实例同时触发（Thundering Herd），默认为 10s
	// 若 ExpiryJitter >= ExpiryInterval，自动调整：
	//   ExpiryInterval = (ExpiryInterval + ExpiryJitter) / 2
	//   ExpiryJitter   = ExpiryInterval（调整后）
	ExpiryJitter time.Duration
}

func (o *Options) applyDefaults() {
	if o.TableName == "" {
		o.TableName = DefaultTableName
	}
	if o.ExpiryInterval <= 0 {
		o.ExpiryInterval = defaultExpiryInterval
	}
	if o.ExpiryJitter < 0 {
		o.ExpiryJitter = 0
	} else if o.ExpiryJitter == 0 {
		o.ExpiryJitter = defaultExpiryJitter
	}
	if o.ExpiryJitter >= o.ExpiryInterval {
		o.ExpiryInterval = (o.ExpiryInterval + o.ExpiryJitter) / 2
		o.ExpiryJitter = o.ExpiryInterval
	}
}

// ClientGetter 返回动态获取 Redis 客户端的函数
func ClientGetter(
	name string, redSQLOpts Options, opts ...client.Option,
) func(context.Context) (RedSQL, error) {
	redSQLOpts.applyDefaults()

	gw := getterWrapper{
		name:       name,
		opts:       opts,
		redSQLOpts: redSQLOpts,
	}

	startExpiryWorker(name, opts, redSQLOpts)

	return gw.getter
}

type getterWrapper struct {
	name string
	opts []client.Option

	redSQLOpts Options
}

func (gw getterWrapper) getter(ctx context.Context) (RedSQL, error) {
	cli, err := sqlx.ClientGetter(gw.name, gw.opts...)(ctx)
	if err != nil {
		return nil, err
	}
	return wrapper{cli: cli, opts: gw.redSQLOpts}, nil
}

type wrapper struct {
	cli  sqlx.Client
	opts Options
}

// New 使用已有的 sqlx.Client 创建 RedSQL 实例，适用于不启动 trpc 框架的场景（如单元测试）
func New(cli sqlx.Client, opts Options) RedSQL {
	opts.applyDefaults()
	return wrapper{cli: cli, opts: opts}
}
