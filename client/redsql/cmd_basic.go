package redsql

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

// kvRow 是 SELECT value, expire_ts_ms 的结果行（expire_ts_ms=0 表示永不过期）
type kvRow struct {
	Value    string `db:"value"`
	ExpireMs int64  `db:"expire_ts_ms"`
}

// Get 获取 key 的值，不存在或已过期时返回 redis.Nil
func (w wrapper) Get(ctx context.Context, key string) *redis.StringCmd {
	cmd := redis.NewStringCmd(ctx)
	var rows []kvRow
	query := fmt.Sprintf(
		"SELECT `value`, `expire_ts_ms` FROM `%s` WHERE `key` = ? AND %s LIMIT 1",
		w.opts.TableName, notExpiredCond(),
	)
	if err := w.cli.SelectContext(ctx, &rows, query, key); err != nil {
		cmd.SetErr(err)
		return cmd
	}
	if len(rows) == 0 {
		cmd.SetErr(redis.Nil)
		return cmd
	}
	cmd.SetVal(rows[0].Value)
	return cmd
}

// Set 设置 key 的值，expiration=0 表示永不过期
func (w wrapper) Set(ctx context.Context, key string, value any, expiration time.Duration) *redis.StatusCmd {
	cmd := redis.NewStatusCmd(ctx)
	query := fmt.Sprintf(
		"INSERT INTO `%s` (`key`, `value`, `expire_ts_ms`) VALUES (?, ?, ?)"+
			" ON DUPLICATE KEY UPDATE `value` = VALUES(`value`), `expire_ts_ms` = VALUES(`expire_ts_ms`)",
		w.opts.TableName,
	)
	if _, err := w.cli.ExecContext(ctx, query, key, formatValue(value), expiresAtMs(expiration)); err != nil {
		cmd.SetErr(err)
		return cmd
	}
	cmd.SetVal("OK")
	return cmd
}

// SetEx 设置 key 的值及过期时间
func (w wrapper) SetEx(ctx context.Context, key string, value any, expiration time.Duration) *redis.StatusCmd {
	return w.Set(ctx, key, value, expiration)
}

// SetArgs 支持 NX/XX/KeepTTL/ExpireAt 等选项的完整 SET 实现
func (w wrapper) SetArgs(ctx context.Context, key string, value any, a redis.SetArgs) *redis.StatusCmd {
	cmd := redis.NewStatusCmd(ctx)

	var ttl time.Duration
	if !a.ExpireAt.IsZero() {
		ttl = time.Until(a.ExpireAt)
		if ttl <= 0 {
			ttl = time.Nanosecond // 立即过期
		}
	} else {
		ttl = a.TTL
	}

	switch a.Mode {
	case "NX":
		r := w.SetNX(ctx, key, value, ttl)
		if r.Err() != nil {
			cmd.SetErr(r.Err())
		} else if !r.Val() {
			cmd.SetErr(redis.Nil)
		} else {
			cmd.SetVal("OK")
		}
		return cmd
	case "XX":
		r := w.SetXX(ctx, key, value, ttl)
		if r.Err() != nil {
			cmd.SetErr(r.Err())
		} else if !r.Val() {
			cmd.SetErr(redis.Nil)
		} else {
			cmd.SetVal("OK")
		}
		return cmd
	}

	if a.KeepTTL {
		// 保留原有 TTL：只更新 value，expire_ts_ms 不变
		// 若 key 不存在则插入（无 TTL，expire_ts_ms=0）
		query := fmt.Sprintf(
			"INSERT INTO `%s` (`key`, `value`, `expire_ts_ms`) VALUES (?, ?, 0)"+
				" ON DUPLICATE KEY UPDATE `value` = VALUES(`value`)",
			w.opts.TableName,
		)
		if _, err := w.cli.ExecContext(ctx, query, key, formatValue(value)); err != nil {
			cmd.SetErr(err)
			return cmd
		}
		cmd.SetVal("OK")
		return cmd
	}

	return w.Set(ctx, key, value, ttl)
}

// SetNX 仅当 key 不存在（或已过期）时设置，成功返回 true
func (w wrapper) SetNX(ctx context.Context, key string, value any, expiration time.Duration) *redis.BoolCmd {
	cmd := redis.NewBoolCmd(ctx)
	var affected int64

	err := w.cli.TransactionContext(ctx, func(ctx context.Context, tx *sqlx.Tx) error {
		// 先删除同名的已过期 key，使 INSERT IGNORE 能插入
		_, err := tx.ExecContext(ctx,
			fmt.Sprintf(
				"DELETE FROM `%s` WHERE `key` = ? AND `expire_ts_ms` != 0 AND `expire_ts_ms` <= UNIX_TIMESTAMP(NOW(3)) * 1000",
				w.opts.TableName,
			), key)
		if err != nil {
			return err
		}
		result, err := tx.ExecContext(ctx,
			fmt.Sprintf(
				"INSERT IGNORE INTO `%s` (`key`, `value`, `expire_ts_ms`) VALUES (?, ?, ?)",
				w.opts.TableName,
			), key, formatValue(value), expiresAtMs(expiration))
		if err != nil {
			return err
		}
		affected, err = result.RowsAffected()
		return err
	})
	if err != nil {
		cmd.SetErr(err)
		return cmd
	}
	cmd.SetVal(affected > 0)
	return cmd
}

// SetXX 仅当 key 存在且未过期时更新，成功返回 true
func (w wrapper) SetXX(ctx context.Context, key string, value any, expiration time.Duration) *redis.BoolCmd {
	cmd := redis.NewBoolCmd(ctx)
	query := fmt.Sprintf(
		"UPDATE `%s` SET `value` = ?, `expire_ts_ms` = ? WHERE `key` = ? AND %s",
		w.opts.TableName, notExpiredCond(),
	)
	result, err := w.cli.ExecContext(ctx, query, formatValue(value), expiresAtMs(expiration), key)
	if err != nil {
		cmd.SetErr(err)
		return cmd
	}
	affected, err := result.RowsAffected()
	if err != nil {
		cmd.SetErr(err)
		return cmd
	}
	cmd.SetVal(affected > 0)
	return cmd
}

// GetDel 获取并删除 key；key 不存在或已过期时返回 redis.Nil
func (w wrapper) GetDel(ctx context.Context, key string) *redis.StringCmd {
	cmd := redis.NewStringCmd(ctx)
	var val string
	var found bool

	err := w.cli.TransactionContext(ctx, func(ctx context.Context, tx *sqlx.Tx) error {
		// SELECT FOR UPDATE 锁住行（无论是否过期）
		var rows []kvRow
		if err := tx.SelectContext(ctx, &rows,
			fmt.Sprintf("SELECT `value`, `expire_ts_ms` FROM `%s` WHERE `key` = ? LIMIT 1 FOR UPDATE",
				w.opts.TableName), key); err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		// 无论是否过期都删除（顺带清理）
		_, err := tx.ExecContext(ctx,
			fmt.Sprintf("DELETE FROM `%s` WHERE `key` = ?", w.opts.TableName), key)
		if err != nil {
			return err
		}
		if !isExpiredRow(rows[0].ExpireMs) {
			found = true
			val = rows[0].Value
		}
		return nil
	})
	if err != nil {
		cmd.SetErr(err)
		return cmd
	}
	if !found {
		cmd.SetErr(redis.Nil)
		return cmd
	}
	cmd.SetVal(val)
	return cmd
}

// GetEx 获取 key 的值并更新其 TTL；expiration=-1 表示移除过期时间（持久化）
func (w wrapper) GetEx(ctx context.Context, key string, expiration time.Duration) *redis.StringCmd {
	cmd := redis.NewStringCmd(ctx)
	var val string
	var found bool

	err := w.cli.TransactionContext(ctx, func(ctx context.Context, tx *sqlx.Tx) error {
		var rows []kvRow
		if err := tx.SelectContext(ctx, &rows,
			fmt.Sprintf("SELECT `value`, `expire_ts_ms` FROM `%s` WHERE `key` = ? AND %s LIMIT 1 FOR UPDATE",
				w.opts.TableName, notExpiredCond()), key); err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		found = true
		val = rows[0].Value

		var newExpireMs int64
		if expiration != -1 {
			newExpireMs = expiresAtMs(expiration) // 0 means no expiry
		}
		_, err := tx.ExecContext(ctx,
			fmt.Sprintf("UPDATE `%s` SET `expire_ts_ms` = ? WHERE `key` = ?", w.opts.TableName),
			newExpireMs, key)
		return err
	})
	if err != nil {
		cmd.SetErr(err)
		return cmd
	}
	if !found {
		cmd.SetErr(redis.Nil)
		return cmd
	}
	cmd.SetVal(val)
	return cmd
}

// GetSet 获取旧值并设置新值（原子）；key 不存在时仍然设置，但返回 redis.Nil
// 注：该命令在 Redis 6.2 中已废弃，建议使用 GetEx
func (w wrapper) GetSet(ctx context.Context, key string, value any) *redis.StringCmd {
	cmd := redis.NewStringCmd(ctx)
	newVal := formatValue(value)
	var oldVal string
	var found bool

	err := w.cli.TransactionContext(ctx, func(ctx context.Context, tx *sqlx.Tx) error {
		var rows []kvRow
		if err := tx.SelectContext(ctx, &rows,
			fmt.Sprintf("SELECT `value`, `expire_ts_ms` FROM `%s` WHERE `key` = ? LIMIT 1 FOR UPDATE",
				w.opts.TableName), key); err != nil {
			return err
		}
		if len(rows) > 0 && !isExpiredRow(rows[0].ExpireMs) {
			found = true
			oldVal = rows[0].Value
		}
		// 无论 key 是否存在/过期，都用新值覆盖（清除 TTL，与 Redis 行为一致）
		_, err := tx.ExecContext(ctx,
			fmt.Sprintf(
				"INSERT INTO `%s` (`key`, `value`, `expire_ts_ms`) VALUES (?, ?, 0)"+
					" ON DUPLICATE KEY UPDATE `value` = VALUES(`value`), `expire_ts_ms` = 0",
				w.opts.TableName),
			key, newVal)
		return err
	})
	if err != nil {
		cmd.SetErr(err)
		return cmd
	}
	if !found {
		cmd.SetErr(redis.Nil)
		return cmd
	}
	cmd.SetVal(oldVal)
	return cmd
}
