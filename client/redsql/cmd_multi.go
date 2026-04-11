package redsql

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

// MGet 批量获取多个 key 的值，不存在或已过期的位置返回 nil
func (w wrapper) MGet(ctx context.Context, keys ...string) *redis.SliceCmd {
	cmd := redis.NewSliceCmd(ctx)
	if len(keys) == 0 {
		cmd.SetVal(nil)
		return cmd
	}

	type mkvRow struct {
		Key   string `db:"key"`
		Value string `db:"value"`
	}
	var rows []mkvRow

	args := make([]any, len(keys))
	for i, k := range keys {
		args[i] = k
	}
	query := fmt.Sprintf(
		"SELECT `key`, `value` FROM `%s` WHERE `key` IN (%s) AND %s",
		w.opts.TableName, inPlaceholders(len(keys)), notExpiredCond(),
	)
	if err := w.cli.SelectContext(ctx, &rows, query, args...); err != nil {
		cmd.SetErr(err)
		return cmd
	}

	m := make(map[string]string, len(rows))
	for _, r := range rows {
		m[r.Key] = r.Value
	}

	result := make([]any, len(keys))
	for i, k := range keys {
		if v, ok := m[k]; ok {
			result[i] = v
		}
		// 不存在或已过期：result[i] 保持 nil
	}
	cmd.SetVal(result)
	return cmd
}

// MSet 原子地设置多个 key（事务），参数支持交替 key/value、map[string]string 等多种形式
func (w wrapper) MSet(ctx context.Context, values ...any) *redis.StatusCmd {
	cmd := redis.NewStatusCmd(ctx)
	keys, vals, err := parseKVPairs(values)
	if err != nil {
		cmd.SetErr(err)
		return cmd
	}
	if len(keys) == 0 {
		cmd.SetVal("OK")
		return cmd
	}

	err = w.cli.TransactionContext(ctx, func(ctx context.Context, tx *sqlx.Tx) error {
		query := fmt.Sprintf(
			"INSERT INTO `%s` (`key`, `value`, `expire_ts_ms`) VALUES (?, ?, 0)"+
				" ON DUPLICATE KEY UPDATE `value` = VALUES(`value`), `expire_ts_ms` = 0",
			w.opts.TableName,
		)
		for i, key := range keys {
			if _, err := tx.ExecContext(ctx, query, key, vals[i]); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		cmd.SetErr(err)
		return cmd
	}
	cmd.SetVal("OK")
	return cmd
}

// MSetNX 仅当所有 key 均不存在（或已过期）时原子地设置全部 key，否则不做任何操作
func (w wrapper) MSetNX(ctx context.Context, values ...any) *redis.BoolCmd {
	cmd := redis.NewBoolCmd(ctx)
	keys, vals, err := parseKVPairs(values)
	if err != nil {
		cmd.SetErr(err)
		return cmd
	}
	if len(keys) == 0 {
		cmd.SetVal(true)
		return cmd
	}

	var success bool
	err = w.cli.TransactionContext(ctx, func(ctx context.Context, tx *sqlx.Tx) error {
		// 1. 锁住所有相关行（无论是否过期），防止并发竞争
		args := make([]any, len(keys))
		for i, k := range keys {
			args[i] = k
		}
		type lockRow struct {
			Key      string `db:"key"`
			ExpireMs int64  `db:"expire_ts_ms"`
		}
		var locked []lockRow
		if err := tx.SelectContext(ctx, &locked,
			fmt.Sprintf(
				"SELECT `key`, `expire_ts_ms` FROM `%s` WHERE `key` IN (%s) FOR UPDATE",
				w.opts.TableName, inPlaceholders(len(keys)),
			), args...); err != nil {
			return err
		}

		// 2. 检查是否有未过期的 key
		for _, r := range locked {
			if !isExpiredRow(r.ExpireMs) {
				return nil // 有 key 存在，整体失败，不设置
			}
		}

		// 3. 删除已过期的旧行（有的话）
		if len(locked) > 0 {
			if _, err := tx.ExecContext(ctx,
				fmt.Sprintf("DELETE FROM `%s` WHERE `key` IN (%s)", w.opts.TableName, inPlaceholders(len(keys))),
				args...); err != nil {
				return err
			}
		}

		// 4. 插入所有新 key
		insertQuery := fmt.Sprintf(
			"INSERT INTO `%s` (`key`, `value`, `expire_ts_ms`) VALUES (?, ?, 0)",
			w.opts.TableName,
		)
		for i, key := range keys {
			if _, err := tx.ExecContext(ctx, insertQuery, key, vals[i]); err != nil {
				return err
			}
		}
		success = true
		return nil
	})
	if err != nil {
		cmd.SetErr(err)
		return cmd
	}
	cmd.SetVal(success)
	return cmd
}
