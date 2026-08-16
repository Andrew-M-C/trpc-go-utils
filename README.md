# trpc-go-utils

腾讯 tRPC 框架 Go 工具库。

目前是 Andrew 个人使用，但开放代码用于参考和交流。

本仓库是多 module workspace，各 package 可独立引入：

```
github.com/Andrew-M-C/trpc-go-utils/<package>
```

## 核心

### log

封装 `trpc-go/log`：

- printf 风格 API（`Info` / `Infof` / `InfoContext` 等）
- 链式结构化日志：`log.New().With(k, v).Err(err).Text("...").InfoContext(ctx)`
- `WithLogger` 把请求级公共字段注入 context，后续 `*Context` 日志会一并输出
- `Dying(ctx)` 染色：`DebugContext` 会升为 Info，并带上 `DYING=true`
- `RegisterTraceLogFilter` 注册名为 `tracelog` 的 filter：记录 server/client 出入参与耗时，并按下面规则处理 W3C tracing

#### 与 OpenTelemetry tracing 的关系

打日志时，`log` **不创建 span**。带 context 的 API（如 `InfoContext`、`Logger.InfoContext`）会把 `ctx` 交给内部的 `logStringer`，序列化时调用 `trace.SpanContextFromContext(ctx)`：

- 有 TraceID 则写入结构化字段 `trace_id`
- 有 SpanID 则写入 `span_id`
- `context.Background()` 或未挂 span 的 ctx：这两个字段都不会出现

因此业务日志要带上 tracing，需要：上游已经把 span 放进 ctx，并且使用带 ctx 的 API。同一条 trace 下的子 span 会输出相同的 `trace_id`、不同的 `span_id`。底层仍走 `trpc-go/log`（TIME / LEVEL / FILE / LINE 等由它打印）；`trace_id` / `span_id` 是本包额外加的 JSON 字段。

`tracelog` filter 负责 **HTTP 场景下的 span 创建与传播**（W3C `traceparent` / `tracestate`）：

- **server**：仅当请求是 HTTP 且 header 带有合法 `traceparent` 时，从中 Extract 远程 context，Start 一个 server span 并 `defer End()`；同时把 `trace_id` / `span_id` 写入 ctx（OTel span + `WithLogger`），后续 `*Context` 日志会带上它们
- **client**：每次调用都会 Start 一个 client span（有父 span 则为子 span，否则为新 root）并 `defer End()`；然后把 tracing header 注入下游。若已有 HTTP `ClientReqHeader`，写入 `traceparent` / `tracestate`；否则写入 `ClientMetaData`

### concurrent

用 `Detach` / `DetachAndWait` 替代裸 `go func`：克隆 tRPC message、把父 ctx 里的 OTel span 拷到新 ctx 再 `NewSpan` 开子 span、复制已 `RegisterContextKeyWhenDetach` 的 context key，并自动 `recovery.CatchPanic`。后台任务里用 `log.*Context` 即可继续打出同一条 `trace_id`。

### recovery

`CatchPanic`：捕获 panic，可打错误日志、打 metrics、回调。`concurrent` 内部也依赖它。

### errs

封装 tRPC `errs`，可提取 code / message（可选对 wrapping 做摘要）。

`err_to_code` / `code_to_err` filter：server 把返回的 `error` 写入 rsp 的 code/msg；client 把 rsp 的 code/msg 还原成 `error`。调用体验接近普通函数，不必再单独判断 rsp.code。须在 `trpc.NewServer` 之前 `RegisterErrToCodeFilter`。

### plugin

泛型 `Register` / `Bind`，把 `trpc_go.yaml` 的 plugin 配置绑定到本地结构体，支持 `WithDependsOn` / `WithFlexDependsOn`。

### metrics

泛型 `IncrCounter` / `SetGauge`。

子包 [`metrics/log`](./metrics/log) 实现名为 `log` 的 MetricsSink：按分钟聚合单维度指标（min / max / sum / avg / count）并写入日志。

### codec

`UseOfficialJSON()` 用标准库 `encoding/json` 替换 tRPC 默认 JSON serializer。

## 配置与命名

### config

动态配置 `Bind` / `Watch`，支持 JSON / YAML / TEXT。同一 key 被多个消费者 watch 时，只向配置中心挂一次，再在进程内分发。

### config/etcd

etcd 配置后端，实现 `config.API`。`RegisterClientProvider` 可 watch etcd 中的 client YAML，热更新并覆盖 tRPC global client 配置。

### config/file

本地文件配置，监听文件变化做热更新。不支持 Put / Del。

### naming/mysql

用 MySQL 心跳做服务注册与发现（selector 名 `mysql`）。节点约 2s 心跳、5s 判定存活；查询失败时尽量返回本地缓存。适合没有 Polaris 等名字服务的小环境。

## Client

多数 client 提供 `ClientGetter(name, opts...)`，返回 `func(ctx) (Client, error)`，按 tRPC client 名动态取实例。

### client/buffer

client 连接池：对比 `client.Config(name).Target`，变化则重建实例，并按原 timeout（最长 1 分钟）延迟关闭旧连接。给本身不支持热更新的 client 补动态更新。`redis`、`gorm` 内部都用了它。

### client/redis

基于 `trpc-database/goredis` 的 Redis 客户端，配合 buffer 支持 target 热更新。

子包 [`redcache`](./client/redis/redcache) 是泛型 Redis string 缓存：Get / Set / TTL、cache-aside（`GetWithLoad` / `MGetWithLoad`），默认可 JSON 序列化。

### client/sqlx

基于 `trpc-database/mysql` 的 sqlx 封装（Query / Exec / Select / Transaction）。`SqlxWrapper` 可直接包 `*sqlx.DB`，不必启动 tRPC（如单测）。

### client/gorm

基于 `trpc-database/gorm` 的 GORM 客户端，配合 buffer 支持热更新。`ClientGetter` 返回的 `*gorm.DB` 已 `WithContext(ctx)`。

### client/kafka

基于 `trpc-database/kafka` 的 Kafka 客户端。

### client/localcache

泛型封装官方 `trpc-database/localcache`，接口与 `redcache` 相近（Get / Set / Load / 过期回调等）。

### client/redsql

用 MySQL 模拟 `redis.StringCmdable`（String 命令），适合无 Redis 的开发/测试，或更看重可靠性的场景。key 最长 512 字节，value 最长 65535 字节；不支持 LCS。详见 [DESIGN.md](./client/redsql/DESIGN.md)。
