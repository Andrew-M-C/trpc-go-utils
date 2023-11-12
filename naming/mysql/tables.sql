CREATE TABLE IF NOT EXISTS `t_service_registry` (
  `id` bigint(13) NOT NULL AUTO_INCREMENT COMMENT '自增主键',
  `name` varchar(256) NOT NULL COMMENT '服务名',
  `host` varchar(256) NOT NULL COMMENT 'ip:port',
  `weight` int(11) NOT NULL DEFAULT 100 COMMENT '权重',
  `json_desc` varchar(1024) NOT NULL DEFAULT '{}' COMMENT 'trpc node 描述',
  `deregister_time_msec` bigint(13) NOT NULL DEFAULT 0 COMMENT '注销时间',
  `update_time_msec` bigint(13) NOT NULL COMMENT '更新时间',
  `create_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '可视化创建时间',
  `update_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '可视化更新时间',
  PRIMARY KEY (`id`),
  KEY `i_update_name` (`deregister_time_msec`, `update_time_msec`, `name`)
) ENGINE=InnoDB CHARSET=utf8mb4 AUTO_INCREMENT=1 COMMENT='服务名字自注册';
