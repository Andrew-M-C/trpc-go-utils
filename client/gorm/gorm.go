// Package gorm 实现 gorm 的热更新
package gorm

import (
	"context"
	"fmt"

	"github.com/Andrew-M-C/trpc-go-utils/client/internal"
	"gorm.io/gorm"
	trpcgorm "trpc.group/trpc-go/trpc-database/gorm"
	"trpc.group/trpc-go/trpc-go/client"
)

// ClientGetter 动态获取 gorm 实例
func ClientGetter(
	name string, opts ...client.Option,
) func(context.Context) (*gorm.DB, error) {
	return func(ctx context.Context) (*gorm.DB, error) {
		db, err := buffer.GetClient(name, opts)
		if err != nil {
			return nil, err
		}
		return db.WithContext(ctx), nil
	}
}

var buffer *internal.ClientBuffer[*gorm.DB] = internal.NewClientBuffer(
	"amc.utils.gorm.", newGorm, closeGorm,
)

func closeGorm(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get DB error (%w)", err)
	}
	return sqlDB.Close()
}

func newGorm(name string, opts ...client.Option) (*gorm.DB, error) {
	return trpcgorm.NewClientProxy(name, opts...)
}
