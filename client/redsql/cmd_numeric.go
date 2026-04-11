package redsql

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

// incrByInt 是 Incr/IncrBy/Decr/DecrBy 的公共实现
// SELECT FOR UPDATE 锁住行后在 Go 侧计算新值再 UPSERT，保证原子性
func (w wrapper) incrByInt(ctx context.Context, key string, delta int64) *redis.IntCmd {
	cmd := redis.NewIntCmd(ctx)
	var result int64

	err := w.cli.TransactionContext(ctx, func(ctx context.Context, tx *sqlx.Tx) error {
		// 锁住行（无论是否过期），防止幻读
		var rows []kvRow
		if err := tx.SelectContext(ctx, &rows,
			fmt.Sprintf("SELECT `value`, `expire_ts_ms` FROM `%s` WHERE `key` = ? LIMIT 1 FOR UPDATE",
				w.opts.TableName), key); err != nil {
			return err
		}

		var current int64
		var keepExpire int64

		if len(rows) > 0 && !isExpiredRow(rows[0].ExpireMs) {
			// key 存在且未过期：解析当前值，保留 TTL
			n, err := strconv.ParseInt(strings.TrimSpace(rows[0].Value), 10, 64)
			if err != nil {
				return fmt.Errorf("redsql: value is not an integer or out of range")
			}
			current = n
			keepExpire = rows[0].ExpireMs
		}
		// 已过期或不存在：current=0，keepExpire=0（不保留 TTL）

		result = current + delta

		// UPSERT：如果行存在（可能是过期行），用 ON DUPLICATE KEY UPDATE 覆盖
		_, err := tx.ExecContext(ctx,
			fmt.Sprintf(
				"INSERT INTO `%s` (`key`, `value`, `expire_ts_ms`) VALUES (?, ?, ?)"+
					" ON DUPLICATE KEY UPDATE `value` = VALUES(`value`), `expire_ts_ms` = VALUES(`expire_ts_ms`)",
				w.opts.TableName),
			key, strconv.FormatInt(result, 10), keepExpire)
		return err
	})
	if err != nil {
		cmd.SetErr(err)
		return cmd
	}
	cmd.SetVal(result)
	return cmd
}

func (w wrapper) Incr(ctx context.Context, key string) *redis.IntCmd {
	return w.incrByInt(ctx, key, 1)
}

func (w wrapper) IncrBy(ctx context.Context, key string, value int64) *redis.IntCmd {
	return w.incrByInt(ctx, key, value)
}

func (w wrapper) Decr(ctx context.Context, key string) *redis.IntCmd {
	return w.incrByInt(ctx, key, -1)
}

func (w wrapper) DecrBy(ctx context.Context, key string, decrement int64) *redis.IntCmd {
	return w.incrByInt(ctx, key, -decrement)
}

// IncrByFloat 将 key 的值加上浮点数 delta，与 Redis 行为一致（不允许 NaN / Inf）
func (w wrapper) IncrByFloat(ctx context.Context, key string, value float64) *redis.FloatCmd {
	cmd := redis.NewFloatCmd(ctx)
	var result float64

	err := w.cli.TransactionContext(ctx, func(ctx context.Context, tx *sqlx.Tx) error {
		var rows []kvRow
		if err := tx.SelectContext(ctx, &rows,
			fmt.Sprintf("SELECT `value`, `expire_ts_ms` FROM `%s` WHERE `key` = ? LIMIT 1 FOR UPDATE",
				w.opts.TableName), key); err != nil {
			return err
		}

		var current float64
		var keepExpire int64

		if len(rows) > 0 && !isExpiredRow(rows[0].ExpireMs) {
			f, err := strconv.ParseFloat(strings.TrimSpace(rows[0].Value), 64)
			if err != nil {
				return fmt.Errorf("redsql: value is not a valid float")
			}
			current = f
			keepExpire = rows[0].ExpireMs
		}

		result = current + value
		if math.IsNaN(result) || math.IsInf(result, 0) {
			return fmt.Errorf("redsql: increment would produce NaN or Infinity")
		}

		// 与 Redis 格式保持一致：用 'g' 格式，最多 17 位有效数字
		newVal := strconv.FormatFloat(result, 'g', 17, 64)

		_, err := tx.ExecContext(ctx,
			fmt.Sprintf(
				"INSERT INTO `%s` (`key`, `value`, `expire_ts_ms`) VALUES (?, ?, ?)"+
					" ON DUPLICATE KEY UPDATE `value` = VALUES(`value`), `expire_ts_ms` = VALUES(`expire_ts_ms`)",
				w.opts.TableName),
			key, newVal, keepExpire)
		return err
	})
	if err != nil {
		cmd.SetErr(err)
		return cmd
	}
	cmd.SetVal(result)
	return cmd
}
