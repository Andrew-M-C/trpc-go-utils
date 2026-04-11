package redsql

import (
	"fmt"
	"strings"
	"time"
)

// nowMs 返回当前 Unix 毫秒时间戳
func nowMs() int64 {
	return time.Now().UnixMilli()
}

// expiresAtMs 将 TTL 转换为绝对过期时间戳（毫秒）
// d <= 0 表示永不过期，返回 0
func expiresAtMs(d time.Duration) int64 {
	if d <= 0 {
		return 0
	}
	return time.Now().Add(d).UnixMilli()
}

// isExpiredRow 判断一行数据是否已过期（0 表示永不过期）
func isExpiredRow(expireMs int64) bool {
	return expireMs != 0 && expireMs <= nowMs()
}

// notExpiredCond 返回用于 SELECT 的未过期过滤 SQL 片段
func notExpiredCond() string {
	return "(`expire_ts_ms` = 0 OR `expire_ts_ms` > UNIX_TIMESTAMP(NOW(3)) * 1000)"
}

// formatValue 将任意值转换为 Redis 兼容的字符串
func formatValue(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return string(val)
	default:
		return fmt.Sprint(v)
	}
}

// inPlaceholders 生成 n 个 ? 占位符（逗号分隔）
func inPlaceholders(n int) string {
	if n <= 0 {
		return ""
	}
	return "?" + strings.Repeat(",?", n-1)
}

// flattenArgs 将 MSet/MSetNX 的可变参数展开为扁平 []any
// 支持：交替的 key/value 对、[]string、[]any、map[string]string、map[string]any
func flattenArgs(values []any) []any {
	if len(values) == 1 {
		return flattenSingle(nil, values[0])
	}
	return values
}

func flattenSingle(dst []any, arg any) []any {
	switch v := arg.(type) {
	case []string:
		for _, s := range v {
			dst = append(dst, s)
		}
	case []any:
		dst = append(dst, v...)
	case map[string]any:
		for k, val := range v {
			dst = append(dst, k, val)
		}
	case map[string]string:
		for k, val := range v {
			dst = append(dst, k, val)
		}
	default:
		dst = append(dst, arg)
	}
	return dst
}

// parseKVPairs 将 MSet/MSetNX 参数解析为 (keys, values) 切片
func parseKVPairs(values []any) (keys, vals []string, err error) {
	flat := flattenArgs(values)
	if len(flat)%2 != 0 {
		return nil, nil, fmt.Errorf("redsql: requires even number of key-value arguments, got %d", len(flat))
	}
	for i := 0; i < len(flat); i += 2 {
		keys = append(keys, formatValue(flat[i]))
		vals = append(vals, formatValue(flat[i+1]))
	}
	return
}
