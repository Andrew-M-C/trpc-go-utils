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
