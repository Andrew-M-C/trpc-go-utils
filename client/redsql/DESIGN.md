# redsql 设计文档

## 概述

`redsql` 使用 MySQL 作为存储后端，实现 `redis.StringCmdable` 接口，让业务代码在不依赖 Redis 的情况下，以相同的 API 使用 MySQL 模拟 Redis String 操作。

典型使用场景：开发/测试环境无 Redis、或对可靠性要求高于性能的场景。

---

## 对外接口

```go
// RedSQL 表示 Redis 模拟实现，当前实现 redis.StringCmdable
type RedSQL interface {
    redis.StringCmdable
}

// ClientGetter 返回动态获取 RedSQL 客户端的函数，遵循 trpc-go 的 client 惯例
func ClientGetter(
    name string,
    redSQLOpts RedSQLOptions,
    opts ...client.Option,
) func(context.Context) (RedSQL, error)
```

### RedSQLOptions

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `TableName` | `string` | `t_redsql_kv` | 存储 KV 数据的 MySQL 表名 |
| `ExpiryInterval` | `time.Duration` | `60s` | 后台过期清理协程的轮询间隔 |
| `ExpiryJitter` | `time.Duration` | `10s` | 启动抖动范围，防止多实例 Thundering Herd |

**ExpiryJitter 自动调整规则**（在 `applyDefaults()` 中执行）：
- `ExpiryJitter < 0` → 修正为 `0`（不抖动）
- `ExpiryJitter >= ExpiryInterval` → `ExpiryInterval = (ExpiryInterval + ExpiryJitter) / 2`，`ExpiryJitter = ExpiryInterval`（保证抖动始终小于一个周期）

---

## 数据库表结构

```sql
CREATE TABLE IF NOT EXISTS `t_redsql_kv` (
  `id`           BIGINT        NOT NULL AUTO_INCREMENT COMMENT '自增主键',
  `key`          VARCHAR(512)  NOT NULL COMMENT 'Redis key',
  `value`        TEXT          NOT NULL COMMENT 'Redis value',
  `expire_ts_ms` BIGINT        NOT NULL DEFAULT 0 COMMENT 'Unix 毫秒时间戳，0 表示永不过期',
  `create_time`  DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `update_time`  DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_key` (`key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;
```

**设计要点：**

- `key` 使用 `utf8mb4_bin`（大小写敏感），与 Redis 行为一致
- `expire_ts_ms` 为 NOT NULL，`0` 表示永不过期，避免 IS NULL 判断，简化 SQL 并对索引更友好
- `id` 为自增主键，`key` 通过 UNIQUE KEY 保证唯一性；写操作统一用 `INSERT ... ON DUPLICATE KEY UPDATE` 实现 upsert 语义
- `create_time` / `update_time` 由 MySQL 自动维护

---

## 过期 Key 处理策略

### 读时过滤（Lazy Deletion）

所有读操作的 WHERE 条件中加入：

```sql
(`expire_ts_ms` = 0 OR `expire_ts_ms` > UNIX_TIMESTAMP(NOW(3)) * 1000)
```

过期的 key 在读时视同不存在，返回 `redis.Nil`，但不立即从表中删除。

### 后台轮询清理（Active Expiry）

`ClientGetter` 调用时通过 `concurrent.Detach` 启动一个后台协程，定期执行：

```sql
DELETE FROM `{table}`
WHERE `expire_ts_ms` != 0
  AND `expire_ts_ms` <= UNIX_TIMESTAMP(NOW(3)) * 1000
LIMIT 1000
```

- **强制开启**，不可禁用；间隔和抖动通过 `RedSQLOptions` 配置
- 每次限删 1000 行，防止大批量删除造成锁争用
- 每次清理创建独立 trace ID（`log.EnsureTraceID`），便于日志追踪
- 成功删除 > 0 行时打印 `Info` 日志；获取连接失败或 DELETE 失败时打印 `Error` 日志

**Thundering Herd 防护**：协程启动时先随机 sleep `[0, ExpiryJitter)`，将多个实例的首次触发时刻打散。

---

## 命令实现一览

### 基础读写（`cmd_basic.go`）

| 命令 | 实现方式 |
|------|---------|
| `Get` | `SELECT ... WHERE key=? AND not_expired LIMIT 1` |
| `Set` | `INSERT ... ON DUPLICATE KEY UPDATE`（upsert） |
| `SetEx` | 同 `Set`，`expire_ts_ms = now + ttl` |
| `SetArgs` | 拆解为 NX/XX/KeepTTL/普通 Set，分别调用对应方法 |
| `SetNX` | 事务：先 DELETE 过期行，再 `INSERT IGNORE`；检查 affected rows 判断成功与否 |
| `SetXX` | `UPDATE ... WHERE key=? AND not_expired`；检查 affected rows |
| `GetDel` | 事务：`SELECT FOR UPDATE`（锁行）→ DELETE → 返回旧值 |
| `GetEx` | 事务：`SELECT FOR UPDATE`（过滤过期）→ `UPDATE expire_ts_ms`；`expiration=-1` 时置 0（持久化） |
| `GetSet` | 事务：`SELECT FOR UPDATE`（获取旧值）→ upsert 新值并清除 TTL |

### 批量操作（`cmd_multi.go`）

| 命令 | 实现方式 |
|------|---------|
| `MGet` | `SELECT key, value WHERE key IN (?) AND not_expired`；按原始 key 顺序填充结果 |
| `MSet` | 事务：对每个 key 执行 upsert，保证原子性 |
| `MSetNX` | 事务：`SELECT ... FOR UPDATE`（锁全部 key）→ 检查是否有未过期行 → DELETE 过期行 → INSERT 全部 |

### 数值增减（`cmd_numeric.go`）

所有操作均在事务中执行 `SELECT ... FOR UPDATE` 锁行，在 Go 侧计算新值后 upsert：

| 命令 | 说明 |
|------|------|
| `Incr` / `IncrBy` | 整数加法；key 不存在或已过期时从 0 开始 |
| `Decr` / `DecrBy` | 整数减法 |
| `IncrByFloat` | 浮点数加法；结果为 NaN / Inf 时返回错误 |

**TTL 保留策略**：若 key 存在且未过期，保留原有 `expire_ts_ms`；若 key 不存在或已过期，新建时 `expire_ts_ms = 0`。

### 字符串操作（`cmd_string.go`）

| 命令 | 实现方式 |
|------|---------|
| `Append` | 事务：`SELECT FOR UPDATE` → 拼接 → upsert；保留原有 TTL |
| `StrLen` | `SELECT value WHERE key=? AND not_expired`；在 Go 侧计算字节长度 |
| `GetRange` | `SELECT value`；在 Go 侧截取（支持负索引，与 Redis 语义一致） |
| `SetRange` | 事务：`SELECT FOR UPDATE` → Go 侧覆写（不足位置填 `\x00`）→ upsert；保留原有 TTL |
| `SetArgs` | （见基础读写） |
| `LCS` | 暂不支持，返回 `errors.New("redsql: LCS is not supported")` |

---

## 并发安全设计

| 场景 | 保障机制 |
|------|---------|
| `Set` / `MSet` 并发写同一 key | `ON DUPLICATE KEY UPDATE`，MySQL 层原子 upsert |
| `SetNX` 并发写同一 key | 事务内 DELETE 过期行 + `INSERT IGNORE`；InnoDB UNIQUE KEY 保证唯一性 |
| `MSetNX` 多 key 原子检查 | 事务内 `SELECT FOR UPDATE` 锁全部相关行，消除 TOCTOU 竞争 |
| `Incr` / `Append` 等读-改-写 | 事务内 `SELECT ... FOR UPDATE` 锁行（InnoDB gap lock 防止幻读） |

---

## 返回值封装约定

go-redis 的命令方法返回指针类型（如 `*redis.StringCmd`），所有实现遵循以下模式：

```go
cmd := redis.NewStringCmd(ctx)
cmd.SetVal("hello")   // 成功时设置值
cmd.SetErr(err)       // 失败时设置错误
cmd.SetErr(redis.Nil) // key 不存在时设置 redis.Nil
return cmd
```

---

## 文件结构

```
client/redsql/
├── go.mod
├── DESIGN.md          // 本文件
├── PLAN.md            // 开发过程规划与决策记录
├── redsql.go          // 公共接口、ClientGetter、wrapper / RedSQLOptions 定义
├── helpers.go         // 工具函数（notExpiredCond、expiresAtMs、formatValue 等）
├── expiry.go          // 后台过期清理协程（concurrent.Detach + jitter + log）
├── cmd_basic.go       // Get / Set / SetEx / SetArgs / SetNX / SetXX / GetDel / GetEx / GetSet
├── cmd_multi.go       // MGet / MSet / MSetNX
├── cmd_numeric.go     // Incr / IncrBy / Decr / DecrBy / IncrByFloat
└── cmd_string.go      // Append / StrLen / GetRange / SetRange / LCS
```

---

## 与真实 Redis 的行为差异

| 差异点 | 说明 |
|--------|------|
| **TTL 精度** | Redis TTL 精确到毫秒；本实现依赖 MySQL `UNIX_TIMESTAMP(NOW(3))` 精度，误差在 1ms 以内 |
| **过期清理延迟** | Redis 的 Active Expiry 几乎实时；本实现依赖后台轮询（默认 60s），期间过期 key 不会出现在读结果中，但磁盘上的行仍然存在 |
| **原子性范围** | Redis 单命令天然原子；本实现的写操作通过 MySQL 事务或 `INSERT IGNORE` 保证原子性，但跨命令不保证（无 MULTI/EXEC） |
| **`LCS` 命令** | 暂不支持 |
| **`GetSet` 废弃命令** | 已完整实现，保持接口兼容 |
| **key 长度限制** | MySQL `VARCHAR(512)` 限制 key 最长 512 字节；Redis 无此限制 |
| **value 大小限制** | MySQL `TEXT` 上限 65 535 字节；如需更大可改为 `MEDIUMTEXT` |
