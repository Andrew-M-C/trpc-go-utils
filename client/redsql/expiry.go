package redsql

import (
	"context"
	"math/rand"
	"time"

	"github.com/Andrew-M-C/trpc-go-utils/client/sqlx"
	"github.com/Andrew-M-C/trpc-go-utils/concurrent"
	"github.com/Andrew-M-C/trpc-go-utils/log"
	"trpc.group/trpc-go/trpc-go/client"
)

// startExpiryWorker 使用 concurrent.Detach 启动后台过期清理协程
// 首次启动时会在 [0, ExpiryJitter) 内随机 sleep，将多个实例的触发时刻打散（避免 Thundering Herd）
func startExpiryWorker(name string, opts []client.Option, redSQLOpts Options) {
	concurrent.Detach(context.Background(), func(ctx context.Context) {
		if redSQLOpts.ExpiryJitter > 0 {
			sleepDur := time.Duration(rand.Int63n(int64(redSQLOpts.ExpiryJitter)))
			time.Sleep(sleepDur)
		}
		ticker := time.NewTicker(redSQLOpts.ExpiryInterval)
		defer ticker.Stop()
		for range ticker.C {
			runExpiryCleanup(name, opts, redSQLOpts)
		}
	})
}

func runExpiryCleanup(name string, opts []client.Option, redSQLOpts Options) {
	ctx := log.EnsureTraceID(context.Background())

	cli, err := sqlx.ClientGetter(name, opts...)(ctx)
	if err != nil {
		log.ErrorContextf(ctx, "redsql: expiry cleanup failed to get DB client, table=%s err=%v",
			redSQLOpts.TableName, err)
		return
	}

	query := "DELETE FROM `" + redSQLOpts.TableName +
		"` WHERE `expire_ts_ms` != 0 AND `expire_ts_ms` <= UNIX_TIMESTAMP(NOW(3)) * 1000 LIMIT 1000"
	result, err := cli.ExecContext(ctx, query)
	if err != nil {
		log.ErrorContextf(ctx, "redsql: expiry cleanup DELETE failed, table=%s err=%v",
			redSQLOpts.TableName, err)
		return
	}

	if affected, _ := result.RowsAffected(); affected > 0 {
		log.InfoContextf(ctx, "redsql: expiry cleanup deleted %d rows, table=%s",
			affected, redSQLOpts.TableName)
	}
}
