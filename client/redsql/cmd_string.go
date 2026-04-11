package redsql

import (
	"context"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

func errNotSupported(cmd string) error {
	return errors.New("redsql: " + cmd + " is not supported")
}

// Append 在 key 的值末尾追加字符串，返回追加后的字节长度；key 不存在时相当于 SET
func (w wrapper) Append(ctx context.Context, key, value string) *redis.IntCmd {
	cmd := redis.NewIntCmd(ctx)
	var newLen int64

	err := w.cli.TransactionContext(ctx, func(ctx context.Context, tx *sqlx.Tx) error {
		var rows []kvRow
		if err := tx.SelectContext(ctx, &rows,
			fmt.Sprintf("SELECT `value`, `expire_ts_ms` FROM `%s` WHERE `key` = ? LIMIT 1 FOR UPDATE",
				w.opts.TableName), key); err != nil {
			return err
		}

		var newVal string
		var keepExpire int64

		if len(rows) > 0 && !isExpiredRow(rows[0].ExpireMs) {
			newVal = rows[0].Value + value
			keepExpire = rows[0].ExpireMs
		} else {
			newVal = value
			// 已过期或不存在：创建新 key，无 TTL（keepExpire=0）
		}
		newLen = int64(len(newVal))

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
	cmd.SetVal(newLen)
	return cmd
}

// StrLen 返回 key 对应字符串的字节长度；key 不存在时返回 0
func (w wrapper) StrLen(ctx context.Context, key string) *redis.IntCmd {
	cmd := redis.NewIntCmd(ctx)
	var rows []kvRow
	if err := w.cli.SelectContext(ctx, &rows,
		fmt.Sprintf("SELECT `value`, `expire_ts_ms` FROM `%s` WHERE `key` = ? AND %s LIMIT 1",
			w.opts.TableName, notExpiredCond()), key); err != nil {
		cmd.SetErr(err)
		return cmd
	}
	if len(rows) == 0 {
		cmd.SetVal(0)
		return cmd
	}
	cmd.SetVal(int64(len(rows[0].Value)))
	return cmd
}

// GetRange 返回 key 对应字符串的子串 [start, end]（两端包含，支持负索引）
// 与 Redis GETRANGE 语义完全一致；key 不存在时返回 ""
func (w wrapper) GetRange(ctx context.Context, key string, start, end int64) *redis.StringCmd {
	cmd := redis.NewStringCmd(ctx)
	var rows []kvRow
	if err := w.cli.SelectContext(ctx, &rows,
		fmt.Sprintf("SELECT `value`, `expire_ts_ms` FROM `%s` WHERE `key` = ? AND %s LIMIT 1",
			w.opts.TableName, notExpiredCond()), key); err != nil {
		cmd.SetErr(err)
		return cmd
	}
	if len(rows) == 0 {
		cmd.SetVal("")
		return cmd
	}

	s := rows[0].Value
	n := int64(len(s))

	// 将负索引转换为正索引
	if start < 0 {
		start = n + start
	}
	if end < 0 {
		end = n + end
	}
	// 边界裁剪
	if start < 0 {
		start = 0
	}
	if end >= n {
		end = n - 1
	}
	if start > end || n == 0 {
		cmd.SetVal("")
		return cmd
	}
	cmd.SetVal(s[start : end+1])
	return cmd
}

// SetRange 从 offset 开始覆写 key 的字符串，不足长度时用 \x00 填充；返回写入后的字节长度
// 与 Redis SETRANGE 语义一致；key 不存在时相当于创建
func (w wrapper) SetRange(ctx context.Context, key string, offset int64, value string) *redis.IntCmd {
	cmd := redis.NewIntCmd(ctx)
	var newLen int64

	err := w.cli.TransactionContext(ctx, func(ctx context.Context, tx *sqlx.Tx) error {
		var rows []kvRow
		if err := tx.SelectContext(ctx, &rows,
			fmt.Sprintf("SELECT `value`, `expire_ts_ms` FROM `%s` WHERE `key` = ? LIMIT 1 FOR UPDATE",
				w.opts.TableName), key); err != nil {
			return err
		}

		var existing string
		var keepExpire int64

		if len(rows) > 0 && !isExpiredRow(rows[0].ExpireMs) {
			existing = rows[0].Value
			keepExpire = rows[0].ExpireMs
		}

		// 计算覆写后的新值
		end := offset + int64(len(value))
		buf := []byte(existing)
		if end > int64(len(buf)) {
			// 不足长度时用 \x00 填充
			padded := make([]byte, end)
			copy(padded, buf)
			buf = padded
		}
		copy(buf[offset:], value)
		newVal := string(buf)
		newLen = int64(len(newVal))

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
	cmd.SetVal(newLen)
	return cmd
}

// LCS 暂不支持（MySQL 无内置 LCS 函数，Go 侧实现性能风险较高）
func (w wrapper) LCS(ctx context.Context, q *redis.LCSQuery) *redis.LCSCmd {
	cmd := redis.NewLCSCmd(ctx, q)
	cmd.SetErr(errNotSupported("LCS"))
	return cmd
}
