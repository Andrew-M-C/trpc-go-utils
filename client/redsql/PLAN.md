# redsql 实现计划

## 目标

让 `wrapper{cli sqlx.Client}` 实现 `redis.StringCmdable` 接口，以 MySQL 作为存储后端模拟 Redis String 操作。

---

## 一、数据库表结构设计

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

说明：
- `id` 为自增主键，`key` 通过 `UNIQUE KEY` 保证唯一性
- `key` 使用 `utf8mb4_bin`（大小写敏感，与 Redis 行为一致）
- `value` 用 `TEXT`（上限 65 KB），如需存储更大内容可改为 `MEDIUMTEXT`
- `expire_ts_ms` NOT NULL，`0` 表示永不过期，非零值为 Unix 毫秒时间戳
- NOT NULL 设计使 SQL 条件更简洁，且对索引更友好
- `create_time` / `update_time` 由 MySQL 自动维护
- **默认表名 `t_redsql_kv`，可通过 `RedSQLOptions.TableName` 自定义**

---

## 二、`redis.StringCmdable` 方法清单与实现策略

### 2.1 基础读写 → `cmd_basic.go`

| 方法 | 对应 SQL 操作 | 备注 |
|------|-------------|------|
| `Get` | `SELECT value, expire_ts_ms WHERE key=?`，过期返回 `redis.Nil` | |
| `Set` | `INSERT ... ON DUPLICATE KEY UPDATE`，expiration=0 写 0 | |
| `SetEx` | 同 Set，expire_ts_ms = now + expiration | |
| `SetNX` | 先 DELETE 过期同名 key，再 `INSERT IGNORE`，检查 affected rows | |
| `SetXX` | `UPDATE ... WHERE key=? AND (expire_ts_ms = 0 OR expire_ts_ms > now)`，检查 affected rows | |
| `GetDel` | 事务：SELECT + DELETE | |
| `GetEx` | 事务：SELECT + UPDATE expire_ts_ms；expiration=-1 表示持久化（SET expire_ts_ms=0） | |
| `GetSet` | 事务：SELECT + UPDATE（Redis 已废弃，完整实现保持接口兼容） | |

### 2.2 批量操作 → `cmd_multi.go`

| 方法 | 对应 SQL 操作 | 备注 |
|------|-------------|------|
| `MGet` | `SELECT key, value, expire_ts_ms WHERE key IN (?)` | 按原始 key 顺序填充结果，不存在或过期的位置返回 `nil` |
| `MSet` | 事务：批量 `INSERT ... ON DUPLICATE KEY UPDATE` | 事务保证原子性，与 Redis 语义一致 |
| `MSetNX` | 事务：先 COUNT 所有有效 key（未过期），全不存在才批量 INSERT | 需要串行化事务或行锁 |

### 2.3 数值增减 → `cmd_numeric.go`

| 方法 | 实现方式 |
|------|---------|
| `Incr` | 事务：SELECT FOR UPDATE → 解析整数 → UPDATE，不存在则 INSERT 1 |
| `IncrBy` | 同上，步长由参数决定 |
| `Decr` | 同 Incr，步长 -1 |
| `DecrBy` | 同上，步长 -decrement |
| `IncrByFloat` | 同上，解析 float64，NaN/Inf 返回错误 |

### 2.4 字符串操作 → `cmd_string.go`

| 方法 | 实现方式 | 备注 |
|------|---------|------|
| `Append` | 事务：SELECT FOR UPDATE + `UPDATE value = CONCAT(value, ?)`；不存在则 INSERT | |
| `StrLen` | `SELECT CHAR_LENGTH(value)` | |
| `GetRange` | SELECT → Go 侧截取子串 | 支持负索引，与 Redis 语义对齐 |
| `SetRange` | 事务：SELECT FOR UPDATE → Go 侧拼接覆盖 → UPDATE | |
| `SetArgs` | 拆解 `SetArgs` 结构体，转换为 `Set`/`SetEx`/`SetNX`/`SetXX` 逻辑 | |
| `LCS` | 暂不支持，返回 `errors.New("LCS not supported by redsql")` | |

---

## 三、过期 Key 的处理策略

### 3.1 读时过滤（Lazy Deletion）

`expire_ts_ms` 改为 NOT NULL 后，未过期的过滤条件简化为：

```sql
WHERE `key` = ? AND (`expire_ts_ms` = 0 OR `expire_ts_ms` > UNIX_TIMESTAMP(NOW(3)) * 1000)
```

超时的 key 视同不存在（返回 `redis.Nil`），但不立即从表中删除。

---

### 3.2 后台轮询清理（Active Expiry）→ `expiry.go`

> **⚠️ 待确认：是否保留轮询？详见下方可行性分析（§3.3）**

若保留，每个 `wrapper` 实例初始化时启动一个后台 goroutine，定期执行：

```sql
DELETE FROM `t_redsql_kv`
WHERE `expire_ts_ms` != 0 AND `expire_ts_ms` <= UNIX_TIMESTAMP(NOW(3)) * 1000
LIMIT 1000;
```

- 轮询间隔由 `RedSQLOptions.ExpiryInterval` 配置，默认 **60 秒**
- 启动抖动由 `RedSQLOptions.ExpiryJitter` 配置，默认 **10 秒**；若 `ExpiryJitter >= ExpiryInterval`，`applyDefaults()` 会自动调整两者（见 §3.3 问题 A）
- 每次限删 1000 行，防止大批量删除造成锁争用
- goroutine 与 `context.Background()` 绑定，进程退出自然停止

---

### 3.3 可行性分析：是否可以去掉后台轮询？

#### 用户的出发点

- SELECT 读时过滤已保证正确性，不读到过期数据
- Set/SetEx 等写操作采用 `ON DUPLICATE KEY UPDATE`，会直接覆盖过期行，不产生新行

#### 分析结论：**在正确性上可行，但存在以下工程问题**

---

**问题一：写入型操作不能完全覆盖过期行**

`ON DUPLICATE KEY UPDATE` 仅在 key 已存在时触发，对以下写操作有效：

| 操作 | 是否自然覆盖过期行 | 说明 |
|------|-----------------|------|
| `Set` / `SetEx` | ✅ 是 | UPSERT 直接覆盖 |
| `Incr` / `IncrBy` 系列 | ✅ 是 | SELECT FOR UPDATE + UPSERT |
| `Append` / `SetRange` | ✅ 是 | 事务内 UPSERT |
| `GetSet` | ✅ 是 | 事务内 UPSERT |
| **`SetNX`** | ❌ 否（需改造） | 见下方说明 |
| **`MSetNX`** | ❌ 否（需改造） | 见下方说明 |

**`SetNX` 问题**

当前实现（`DELETE 过期行 + INSERT IGNORE`）在去掉轮询后**仍然正确**，因为 DELETE 步骤本身就是对过期行的清理。

如果想去掉 DELETE 步骤，只用条件性 UPSERT 来代替：

```sql
INSERT INTO t (key, value, expire_ts_ms) VALUES (?, ?, ?)
ON DUPLICATE KEY UPDATE
  value       = IF(expire_ts_ms != 0 AND expire_ts_ms <= NOW_MS, VALUES(value),       value),
  expire_ts_ms = IF(expire_ts_ms != 0 AND expire_ts_ms <= NOW_MS, VALUES(expire_ts_ms), expire_ts_ms)
```

通过 `rows_affected`（1=INSERT、2=UPDATE有变化、0=UPDATE无变化）判断成功失败。
该方案存在一个**极小概率边界情况**：若新旧 `expire_ts_ms` 恰好相同（例如以完全相同的 TTL 重设同一个过期 key），MySQL 视为"无变化"返回 0，导致 SetNX 误判为 false。

**补救方法**：当 `rows_affected = 0` 时，追加一次 SELECT 查询以区分两种情况：

```sql
SELECT expire_ts_ms FROM t WHERE key = ? LIMIT 1
```

- 若查到的 `expire_ts_ms` **已过期**（`!= 0 AND <= now`）→ 说明是上述边界情况，UPSERT 实际已用相同值覆盖，判定为 **SetNX 成功**
- 若查到的 `expire_ts_ms` **未过期**（`= 0 OR > now`）→ key 确实存在且有效，**SetNX 失败**
- 若行不存在 → 极端情况（另一进程在 UPSERT 和 SELECT 之间删除了该行），保守判定为 **SetNX 成功**

此 SELECT 无需加入事务。即使 UPSERT 和 SELECT 之间有并发写入，语义依然正确：若另一进程此期间写入了有效 key，SELECT 会看到未过期的行，返回 false（行为符合预期）。

**`MSetNX` 问题**

MSetNX 需要保证"全部 key 都不存在"才全量设置，是多行原子操作，无法简化为单条 SQL，仍需事务。现有的 `SELECT FOR UPDATE + DELETE expired + INSERT` 模式在无轮询的情况下完全正确，无需改动。

---

**问题二：过期行（ghost row）积累**

无轮询时，过期行永远不会被主动清理，只有当同名 key 被再次写入时才会被覆盖。

- **写密集型场景**（相同的 key 反复 SET）：过期行随即被覆盖，等同于有清理，无问题
- **一次性短生命周期 key**（OTP 验证码、限流计数、会话 token 等）：key 用一次就失效，永远不被覆盖，ghost row 持续积累：
  - 磁盘空间随时间线性增长
  - `UNIQUE KEY` 的 B+ 树索引随之膨胀，查询性能缓慢劣化
  - 在写入量大的场景（例如每秒产生 1000 个不同 key，TTL=5min）下，1 天后将积累约 8600 万行

---

#### 已确认方案：强制开启后台轮询

> **已确认**：保留后台轮询，且**强制开启**（不可禁用），`ExpiryInterval` 仅用于调整间隔，默认 60 秒。

---

#### 强制开启带来的遗留问题

**问题 A：多次调用 `ClientGetter` 会启动多个 goroutine（Thundering Herd 惊群效应）✅ 已处理**

若同一进程对同一 `name`（同一张表）多次调用 `ClientGetter`，每次都会启动一个新的清理 goroutine，导致多个 goroutine 在同一时刻同时触发 DELETE——即 **Thundering Herd（惊群效应）**：所有 goroutine 以相同的 `interval` 启动，间隔完全对齐，每次都同时打向数据库。

**解决方案：引入 Jitter（随机抖动）**

在 `RedSQLOptions` 中新增 `ExpiryJitter time.Duration` 字段，每个 goroutine 启动时先随机 sleep `[0, ExpiryJitter)` 后再开始计时循环，将各 goroutine 的触发时刻打散。

- 默认值：`ExpiryJitter = 10s`
- 取值验证逻辑（在 `applyDefaults()` 中执行）：
  - 若 `ExpiryJitter < 0`，修正为 `0`（不抖动）
  - 若 `ExpiryJitter >= ExpiryInterval`，则执行：
    ```
    ExpiryInterval = (ExpiryInterval + ExpiryJitter) / 2
    ExpiryJitter   = ExpiryInterval  // 抖动等于修正后的间隔
    ```
    此规则保证 jitter 始终小于 interval，避免抖动覆盖范围超过一个完整周期。

示意伪代码：
```go
func startExpiryWorker(...) {
    go func() {
        // 启动抖动：将多个 goroutine 的首次触发打散在 [0, jitter) 内
        if jitter > 0 {
            time.Sleep(time.Duration(rand.Int63n(int64(jitter))))
        }
        ticker := time.NewTicker(interval)
        defer ticker.Stop()
        for range ticker.C {
            runExpiryCleanup(...)
        }
    }()
}
```

---

**问题 B：goroutine 无法主动停止 ✅ 无需处理**

goroutine 绑定 `context.Background()`，随进程退出自然停止，没有优雅关闭机制。对长期运行的服务进程无影响；goroutine 未完成的 DELETE 不影响数据正确性。

每次清理时，在 `runExpiryCleanup` 内部通过 `log.EnsureTraceID` 为该次清理生成独立的 trace ID，便于日志关联追踪：

```go
func runExpiryCleanup(name string, opts []client.Option, redSQLOpts RedSQLOptions) {
    ctx := log.EnsureTraceID(context.Background())
    // ... 后续用 log.InfoContextf / log.ErrorContextf 记录日志
}
```

---

**问题 C：清理失败接入日志 ✅ 已确认**

引入 `github.com/Andrew-M-C/trpc-go-utils/log`，在清理流程的关键节点记录日志：

| 事件 | 日志级别 | 说明 |
|------|---------|------|
| 获取 DB 连接失败 | `log.ErrorContextf` | 连接异常，清理无法进行 |
| DELETE 执行失败 | `log.ErrorContextf` | SQL 执行失败 |
| DELETE 成功且行数 > 0 | `log.InfoContextf` | 记录本次清理的行数，便于监控积压 |
| DELETE 成功且行数 = 0 | 不打印 | 避免无意义的日志刷屏 |

日志中统一携带 `redSQLOpts.TableName` 和本次清理的 trace ID，方便排查。

---

## 四、返回值封装

go-redis 的命令方法（如 `Get`）返回的是 `*redis.StringCmd` 等指针类型，而非 `(string, error)`。这些 Cmd 对象需要通过内部构造函数创建：

```go
// 成功示例
cmd := redis.NewStringCmd(ctx)
cmd.SetVal("hello")
return cmd

// 错误示例
cmd := redis.NewStringCmd(ctx)
cmd.SetErr(err)
return cmd

// Key 不存在
cmd := redis.NewStringCmd(ctx)
cmd.SetErr(redis.Nil)
return cmd
```

所有方法遵循此模式封装。

---

## 五、文件结构规划

```
client/redsql/
├── go.mod
├── redsql.go          // 公共接口、ClientGetter、wrapper 定义
├── helpers.go         // 工具函数（notExpiredCond、expiresAtMs 等）
├── expiry.go          // 后台过期清理 goroutine（对应 §3.2）
├── cmd_basic.go       // §2.1：Get / Set / SetEx / SetNX / SetXX / GetDel / GetEx / GetSet
├── cmd_multi.go       // §2.2：MGet / MSet / MSetNX
├── cmd_numeric.go     // §2.3：Incr / IncrBy / Decr / DecrBy / IncrByFloat
├── cmd_string.go      // §2.4：Append / StrLen / GetRange / SetRange / SetArgs / LCS
└── redsql_test.go     // 单元测试（可选）
```

---

## 六、已确认问题

### Q1：表名配置 ✅

> **已确认**：方案 B，通过 `RedSQLOptions.TableName` 自定义，默认值 `t_redsql_kv`。

---

### Q2：自动建表 ✅

> **已确认**：方案 A，`wrapper` 初始化时自动执行 `CREATE TABLE IF NOT EXISTS`（`table.go`）。

---

### Q3：`MSet` 的原子性 ✅

> **已确认**：方案 A，用事务包裹，保证原子性。

---

### Q4：`LCS` 支持 ✅

> **已确认**：方案 B，暂不支持，返回 `errors.New("LCS not supported by redsql")`。

---

### Q5：`SetNX` 并发安全 ✅

> **已确认**：方案 A，依赖 MySQL `INSERT IGNORE` + UNIQUE KEY，天然原子。

---

### Q6：后台清理 goroutine ✅

> **已确认**：强制开启，不可禁用。`ExpiryInterval` 默认 60 秒，零值使用默认。
> 新增 `ExpiryJitter`（默认 10s）解决 Thundering Herd；接入 `trpc-go-utils/log` 记录清理结果；goroutine 无法主动停止，无需处理。详见 §3.3。

---

### Q7：`value` 存储格式 ✅

> **已确认**：使用 `TEXT`（utf8mb4）。如将来需要存储更大内容可改为 `MEDIUMTEXT`。

---

### Q8：`GetSet` 实现 ✅

> **已确认**：方案 A，完整实现（事务：SELECT + UPDATE），保持接口兼容。

---

### Q9：`expire_ts_ms` 字段语义 ⚠️ 待确认

> **新问题**：`expire_ts_ms` 从 `NULL`（永不过期）改为 `NOT NULL DEFAULT 0`（0 表示永不过期）。
> 此改动需同步修改代码中的 `notExpiredCond()`、`isExpiredRow()`、`kvRow` 结构体、`expiresAtMs()` 等。

---

## 七、暂不实现 / 超出范围

以下操作超出 `StringCmdable` 的范畴，不在本次实现计划内：

- Key 过期通知（Keyspace Notification）
- 多数据库（SELECT db）
- 持久化（RDB / AOF 快照）
- 集群 / 主从复制语义
