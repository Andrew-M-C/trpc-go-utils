package mysql

import (
	"context"
	_ "embed"
	"fmt"
	"strings"
	"time"

	"trpc.group/trpc-go/trpc-go/log"
)

type dbItem struct {
	ID int64 `db:"id"`
	// 常规参数
	Name     string `db:"name"`
	Host     string `db:"host"`
	Weight   int32  `db:"weight"`
	JSONDesc string `db:"json_desc"`
	// 时间参数
	DeregisterTimeMsec int64     `db:"deregister_time_msec"`
	UpdateTimeMsec     int64     `db:"update_time_msec"`
	CreateTime         time.Time `db:"create_time"`
	UpdateTime         time.Time `db:"update_time"`
}

func (dbItem) TableName() string {
	return "t_service_registry"
}

//go:embed tables.sql
var createTableStatements string

func initTable(ctx context.Context) error {
	if internal.tableCreated {
		return nil
	}

	proxy := getMySQLClientProxy()
	statements := strings.Split(createTableStatements, ";")

	for i, s := range statements {
		if s = strings.TrimSpace(s); s == "" {
			continue
		}
		if _, err := proxy.Exec(ctx, s); err != nil {
			log.ErrorContextf(ctx, "执行 sql 失败, 错误信息: %v, SQL 语句: %v", err, toJSON{s})
			return fmt.Errorf("执行初始化命令 #%d 失败: %w", i, err)
		}
	}

	log.TraceContextf(ctx, "mysql 注册表初始化成功")
	internal.tableCreated = true
	return nil
}
