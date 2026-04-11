package redsql_test

import (
	"context"
	_ "embed"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Andrew-M-C/trpc-go-utils/client/redsql"
	sqlxutil "github.com/Andrew-M-C/trpc-go-utils/client/sqlx"
	"github.com/Andrew-M-C/trpc-go-utils/log"
	jmoiron "github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"github.com/smartystreets/goconvey/convey"

	// MySQL 驱动
	_ "github.com/go-sql-driver/mysql"
)

// ---------------------------------------------------------------------------
// goconvey 别名
// ---------------------------------------------------------------------------

var (
	cv = convey.Convey
	so = convey.So

	eq     = convey.ShouldEqual
	notNil = convey.ShouldNotBeNil
	isNil  = convey.ShouldBeNil
)

// ---------------------------------------------------------------------------
// 测试环境
// ---------------------------------------------------------------------------

const dsn = "root:123456@tcp(127.0.0.1:3306)/db_test?charset=utf8mb4&parseTime=true&loc=Local"

//go:embed table.sql
var createTableSQL string

var (
	testDB  *jmoiron.DB
	testRed redsql.RedSQL
	testCtx = context.Background()
)

func TestMain(m *testing.M) {
	log.SetLevel("debug")

	db, err := jmoiron.Connect("mysql", dsn)
	if err != nil {
		panic("连接测试数据库失败: " + err.Error())
	}
	testDB = db

	mustExec("DROP TABLE IF EXISTS `" + redsql.DefaultTableName + "`")
	mustExec(createTableSQL)

	cli := &sqlxutil.SqlxWrapper{DB: testDB}
	testRed = redsql.New(cli, redsql.Options{TableName: redsql.DefaultTableName})

	m.Run()
}

func mustExec(query string) {
	if _, err := testDB.Exec(query); err != nil {
		panic("执行 SQL 失败: " + err.Error() + "\nSQL: " + query)
	}
}

func truncate() {
	mustExec("DELETE FROM `" + redsql.DefaultTableName + "`")
}

// ---------------------------------------------------------------------------
// Get / Set / SetEx
// ---------------------------------------------------------------------------

func TestGetSet(t *testing.T) {
	cv("Get 与 Set", t, func() {
		truncate()

		cv("不存在的 key，Get 返回 redis.Nil", func() {
			cmd := testRed.Get(testCtx, "no_such_key")
			so(errors.Is(cmd.Err(), redis.Nil), eq, true)
		})

		cv("Set 后 Get 能读到值", func() {
			so(testRed.Set(testCtx, "k1", "hello", 0).Err(), isNil)
			val, err := testRed.Get(testCtx, "k1").Result()
			so(err, isNil)
			so(val, eq, "hello")
		})

		cv("Set 覆盖已有值", func() {
			so(testRed.Set(testCtx, "k1", "world", 0).Err(), isNil)
			val, err := testRed.Get(testCtx, "k1").Result()
			so(err, isNil)
			so(val, eq, "world")
		})

		cv("SetEx 在 TTL 内可读", func() {
			so(testRed.SetEx(testCtx, "k_ttl", "v", 500*time.Millisecond).Err(), isNil)
			val, err := testRed.Get(testCtx, "k_ttl").Result()
			so(err, isNil)
			so(val, eq, "v")
		})

		cv("SetEx 过期后 Get 为 redis.Nil", func() {
			so(testRed.SetEx(testCtx, "k_exp", "x", 100*time.Millisecond).Err(), isNil)
			time.Sleep(200 * time.Millisecond)
			cmd := testRed.Get(testCtx, "k_exp")
			so(errors.Is(cmd.Err(), redis.Nil), eq, true)
		})
	})
}

// ---------------------------------------------------------------------------
// SetNX / SetXX
// ---------------------------------------------------------------------------

func TestSetNX(t *testing.T) {
	cv("SetNX", t, func() {
		truncate()

		cv("key 不存在时 SetNX 成功", func() {
			ok, err := testRed.SetNX(testCtx, "nx_key", "v1", 0).Result()
			so(err, isNil)
			so(ok, eq, true)
			val, _ := testRed.Get(testCtx, "nx_key").Result()
			so(val, eq, "v1")

			cv("key 已存在时 SetNX 失败", func() {
				ok2, err2 := testRed.SetNX(testCtx, "nx_key", "v2", 0).Result()
				so(err2, isNil)
				so(ok2, eq, false)
				val2, _ := testRed.Get(testCtx, "nx_key").Result()
				so(val2, eq, "v1") // 值不变
			})
		})

		cv("仅当旧 key 已过期时 SetNX 可再次成功", func() {
			so(testRed.SetEx(testCtx, "nx_exp", "old", 100*time.Millisecond).Err(), isNil)
			time.Sleep(200 * time.Millisecond)
			ok, err := testRed.SetNX(testCtx, "nx_exp", "new", 0).Result()
			so(err, isNil)
			so(ok, eq, true)
			val, _ := testRed.Get(testCtx, "nx_exp").Result()
			so(val, eq, "new")
		})
	})
}

func TestSetXX(t *testing.T) {
	cv("SetXX", t, func() {
		truncate()

		cv("key 不存在时 SetXX 失败", func() {
			ok, err := testRed.SetXX(testCtx, "xx_key", "v", 0).Result()
			so(err, isNil)
			so(ok, eq, false)
		})

		cv("key 存在且未过期时 SetXX 成功", func() {
			so(testRed.Set(testCtx, "xx_key", "old", 0).Err(), isNil)
			ok, err := testRed.SetXX(testCtx, "xx_key", "new", 0).Result()
			so(err, isNil)
			so(ok, eq, true)
			val, _ := testRed.Get(testCtx, "xx_key").Result()
			so(val, eq, "new")
		})

		cv("key 已过期时 SetXX 失败", func() {
			so(testRed.SetEx(testCtx, "xx_exp", "v", 100*time.Millisecond).Err(), isNil)
			time.Sleep(200 * time.Millisecond)
			ok, err := testRed.SetXX(testCtx, "xx_exp", "new", 0).Result()
			so(err, isNil)
			so(ok, eq, false)
		})
	})
}

// ---------------------------------------------------------------------------
// GetDel
// ---------------------------------------------------------------------------

func TestGetDel(t *testing.T) {
	cv("GetDel", t, func() {
		truncate()

		cv("不存在的 key 返回 redis.Nil", func() {
			cmd := testRed.GetDel(testCtx, "missing")
			so(errors.Is(cmd.Err(), redis.Nil), eq, true)
		})

		cv("返回旧值并删除 key", func() {
			so(testRed.Set(testCtx, "gd_key", "bye", 0).Err(), isNil)
			val, err := testRed.GetDel(testCtx, "gd_key").Result()
			so(err, isNil)
			so(val, eq, "bye")
			so(errors.Is(testRed.Get(testCtx, "gd_key").Err(), redis.Nil), eq, true)
		})

		cv("已过期 key 上 GetDel 返回 redis.Nil 但仍删除行", func() {
			so(testRed.SetEx(testCtx, "gd_exp", "v", 100*time.Millisecond).Err(), isNil)
			time.Sleep(200 * time.Millisecond)
			cmd := testRed.GetDel(testCtx, "gd_exp")
			so(errors.Is(cmd.Err(), redis.Nil), eq, true)
		})
	})
}

// ---------------------------------------------------------------------------
// GetEx
// ---------------------------------------------------------------------------

func TestGetEx(t *testing.T) {
	cv("GetEx", t, func() {
		truncate()

		cv("不存在的 key 返回 redis.Nil", func() {
			cmd := testRed.GetEx(testCtx, "missing", 0)
			so(errors.Is(cmd.Err(), redis.Nil), eq, true)
		})

		cv("延长 TTL", func() {
			so(testRed.SetEx(testCtx, "ge_key", "v", 200*time.Millisecond).Err(), isNil)
			val, err := testRed.GetEx(testCtx, "ge_key", time.Second).Result()
			so(err, isNil)
			so(val, eq, "v")
			time.Sleep(250 * time.Millisecond)
			val2, err2 := testRed.Get(testCtx, "ge_key").Result()
			so(err2, isNil)
			so(val2, eq, "v")
		})

		cv("expiration=-1 时去掉过期时间（持久化）", func() {
			so(testRed.SetEx(testCtx, "ge_persist", "v", 200*time.Millisecond).Err(), isNil)
			_, err := testRed.GetEx(testCtx, "ge_persist", -1).Result()
			so(err, isNil)
			time.Sleep(300 * time.Millisecond)
			val, err := testRed.Get(testCtx, "ge_persist").Result()
			so(err, isNil)
			so(val, eq, "v")
		})
	})
}

// ---------------------------------------------------------------------------
// GetSet
// ---------------------------------------------------------------------------

func TestGetSetCmd(t *testing.T) {
	cv("GetSet", t, func() {
		truncate()

		cv("key 不存在时返回 redis.Nil 但仍写入新值", func() {
			cmd := testRed.GetSet(testCtx, "gs_key", "new")
			so(errors.Is(cmd.Err(), redis.Nil), eq, true)
			val, _ := testRed.Get(testCtx, "gs_key").Result()
			so(val, eq, "new")

			cv("再次 GetSet 返回旧值并更新为新值", func() {
				old, err := testRed.GetSet(testCtx, "gs_key", "newer").Result()
				so(err, isNil)
				so(old, eq, "new")
				val2, _ := testRed.Get(testCtx, "gs_key").Result()
				so(val2, eq, "newer")
			})
		})

		cv("GetSet 会清除 TTL", func() {
			so(testRed.SetEx(testCtx, "gs_ttl", "v", 500*time.Millisecond).Err(), isNil)
			testRed.GetSet(testCtx, "gs_ttl", "persistent")
			time.Sleep(600 * time.Millisecond)
			val, err := testRed.Get(testCtx, "gs_ttl").Result()
			so(err, isNil)
			so(val, eq, "persistent")
		})
	})
}

// ---------------------------------------------------------------------------
// SetArgs
// ---------------------------------------------------------------------------

func TestSetArgs(t *testing.T) {
	cv("SetArgs", t, func() {
		truncate()

		cv("Mode NX 行为同 SetNX", func() {
			so(testRed.SetArgs(testCtx, "sa_key", "v1", redis.SetArgs{Mode: "NX"}).Err(), isNil)
			so(errors.Is(testRed.SetArgs(testCtx, "sa_key", "v2", redis.SetArgs{Mode: "NX"}).Err(), redis.Nil), eq, true)
			val, _ := testRed.Get(testCtx, "sa_key").Result()
			so(val, eq, "v1")
		})

		cv("Mode XX 行为同 SetXX", func() {
			err := testRed.SetArgs(testCtx, "sa_xx_new", "v", redis.SetArgs{Mode: "XX"}).Err()
			so(errors.Is(err, redis.Nil), eq, true)
			so(testRed.Set(testCtx, "sa_xx_exists", "old", 0).Err(), isNil)
			so(testRed.SetArgs(testCtx, "sa_xx_exists", "vxx", redis.SetArgs{Mode: "XX"}).Err(), isNil)
			val, _ := testRed.Get(testCtx, "sa_xx_exists").Result()
			so(val, eq, "vxx")
		})

		cv("KeepTTL 更新值但保留过期时间", func() {
			so(testRed.SetEx(testCtx, "sa_keep", "orig", 500*time.Millisecond).Err(), isNil)
			so(testRed.SetArgs(testCtx, "sa_keep", "updated", redis.SetArgs{KeepTTL: true}).Err(), isNil)
			val, _ := testRed.Get(testCtx, "sa_keep").Result()
			so(val, eq, "updated")
			time.Sleep(100 * time.Millisecond)
			val2, err := testRed.Get(testCtx, "sa_keep").Result()
			so(err, isNil)
			so(val2, eq, "updated")
		})
	})
}

// ---------------------------------------------------------------------------
// MGet / MSet / MSetNX
// ---------------------------------------------------------------------------

func TestMGet(t *testing.T) {
	cv("MGet", t, func() {
		truncate()

		so(testRed.MSet(testCtx, "mg_a", "1", "mg_b", "2").Err(), isNil)

		cv("按传入 key 顺序返回值", func() {
			vals, err := testRed.MGet(testCtx, "mg_a", "mg_b", "mg_missing").Result()
			so(err, isNil)
			so(len(vals), eq, 3)
			so(vals[0], eq, "1")
			so(vals[1], eq, "2")
			so(vals[2], isNil)
		})

		cv("已过期 key 对应位置为 nil", func() {
			so(testRed.SetEx(testCtx, "mg_exp", "v", 100*time.Millisecond).Err(), isNil)
			time.Sleep(200 * time.Millisecond)
			vals, err := testRed.MGet(testCtx, "mg_a", "mg_exp").Result()
			so(err, isNil)
			so(vals[0], eq, "1")
			so(vals[1], isNil)
		})
	})
}

func TestMSet(t *testing.T) {
	cv("MSet", t, func() {
		truncate()

		cv("原子设置多对 key-value", func() {
			so(testRed.MSet(testCtx, "ms_a", "va", "ms_b", "vb").Err(), isNil)
			a, _ := testRed.Get(testCtx, "ms_a").Result()
			b, _ := testRed.Get(testCtx, "ms_b").Result()
			so(a, eq, "va")
			so(b, eq, "vb")
		})

		cv("MSet 会清除 TTL", func() {
			so(testRed.SetEx(testCtx, "ms_ttl", "old", 500*time.Millisecond).Err(), isNil)
			so(testRed.MSet(testCtx, "ms_ttl", "new").Err(), isNil)
			time.Sleep(600 * time.Millisecond)
			val, err := testRed.Get(testCtx, "ms_ttl").Result()
			so(err, isNil)
			so(val, eq, "new")
		})
	})
}

func TestMSetNX(t *testing.T) {
	cv("MSetNX", t, func() {
		truncate()

		cv("全部 key 均不存在时成功", func() {
			ok, err := testRed.MSetNX(testCtx, "mnx_a", "va", "mnx_b", "vb").Result()
			so(err, isNil)
			so(ok, eq, true)
			a, _ := testRed.Get(testCtx, "mnx_a").Result()
			so(a, eq, "va")
		})

		cv("任一 key 已存在则整体失败且不应写入新 key", func() {
			so(testRed.Set(testCtx, "mnx_a", "existing", 0).Err(), isNil)
			ok, err := testRed.MSetNX(testCtx, "mnx_a", "new_a", "mnx_c", "vc").Result()
			so(err, isNil)
			so(ok, eq, false)
			so(errors.Is(testRed.Get(testCtx, "mnx_c").Err(), redis.Nil), eq, true)
		})

		cv("已有 key 全部过期时可成功", func() {
			so(testRed.SetEx(testCtx, "mnx_exp", "old", 100*time.Millisecond).Err(), isNil)
			time.Sleep(200 * time.Millisecond)
			ok, err := testRed.MSetNX(testCtx, "mnx_exp", "new").Result()
			so(err, isNil)
			so(ok, eq, true)
			val, _ := testRed.Get(testCtx, "mnx_exp").Result()
			so(val, eq, "new")
		})
	})
}

// ---------------------------------------------------------------------------
// Incr / IncrBy / Decr / DecrBy / IncrByFloat
// ---------------------------------------------------------------------------

func TestNumeric(t *testing.T) {
	cv("数值类命令", t, func() {
		truncate()

		cv("Incr 对不存在的 key 从 0 开始", func() {
			n, err := testRed.Incr(testCtx, "num_key").Result()
			so(err, isNil)
			so(n, eq, int64(1))

			cv("Incr 递增已有值", func() {
				n2, err2 := testRed.Incr(testCtx, "num_key").Result()
				so(err2, isNil)
				so(n2, eq, int64(2))

				cv("IncrBy 增加指定增量", func() {
					n3, err3 := testRed.IncrBy(testCtx, "num_key", 10).Result()
					so(err3, isNil)
					so(n3, eq, int64(12))

					cv("Decr 递减", func() {
						n4, err4 := testRed.Decr(testCtx, "num_key").Result()
						so(err4, isNil)
						so(n4, eq, int64(11))

						cv("DecrBy 减去指定量", func() {
							n5, err5 := testRed.DecrBy(testCtx, "num_key", 5).Result()
							so(err5, isNil)
							so(n5, eq, int64(6))
						})
					})
				})
			})
		})

		cv("已过期 key 上 Incr 从 1 开始", func() {
			so(testRed.Set(testCtx, "num_exp", "99", 100*time.Millisecond).Err(), isNil)
			time.Sleep(200 * time.Millisecond)
			n, err := testRed.Incr(testCtx, "num_exp").Result()
			so(err, isNil)
			so(n, eq, int64(1))
		})

		cv("值非整数时 Incr 返回错误", func() {
			so(testRed.Set(testCtx, "num_str", "abc", 0).Err(), isNil)
			cmd := testRed.Incr(testCtx, "num_str")
			so(cmd.Err(), notNil)
		})
	})
}

func TestIncrByFloat(t *testing.T) {
	cv("IncrByFloat", t, func() {
		truncate()

		cv("不存在的 key 从 0 加浮点数", func() {
			f, err := testRed.IncrByFloat(testCtx, "flt_key", 1.5).Result()
			so(err, isNil)
			so(f, eq, 1.5)

			cv("在已有浮点值上继续加", func() {
				f2, err2 := testRed.IncrByFloat(testCtx, "flt_key", 0.1).Result()
				so(err2, isNil)
				so(f2 > 1.59 && f2 < 1.61, eq, true)
			})
		})

		cv("值非合法浮点数时返回错误", func() {
			so(testRed.Set(testCtx, "flt_bad", "not_a_float", 0).Err(), isNil)
			cmd := testRed.IncrByFloat(testCtx, "flt_bad", 1.0)
			so(cmd.Err(), notNil)
		})
	})
}

// ---------------------------------------------------------------------------
// Append / StrLen / GetRange / SetRange / LCS
// ---------------------------------------------------------------------------

func TestAppend(t *testing.T) {
	cv("Append", t, func() {
		truncate()

		cv("对不存在的 key 追加等价于 SET", func() {
			n, err := testRed.Append(testCtx, "app_key", "hello").Result()
			so(err, isNil)
			so(n, eq, int64(5))

			cv("对已存在 key 做字符串拼接", func() {
				n2, err2 := testRed.Append(testCtx, "app_key", " world").Result()
				so(err2, isNil)
				so(n2, eq, int64(11))
				val, _ := testRed.Get(testCtx, "app_key").Result()
				so(val, eq, "hello world")
			})
		})

		cv("对已过期 key 追加相当于新建", func() {
			so(testRed.SetEx(testCtx, "app_exp", "old", 100*time.Millisecond).Err(), isNil)
			time.Sleep(200 * time.Millisecond)
			n, err := testRed.Append(testCtx, "app_exp", "fresh").Result()
			so(err, isNil)
			so(n, eq, int64(5))
		})

		cv("追加时保留未过期 key 的 TTL", func() {
			so(testRed.SetEx(testCtx, "app_ttl", "hi", 500*time.Millisecond).Err(), isNil)
			testRed.Append(testCtx, "app_ttl", "!")
			time.Sleep(600 * time.Millisecond)
			so(errors.Is(testRed.Get(testCtx, "app_ttl").Err(), redis.Nil), eq, true)
		})
	})
}

func TestStrLen(t *testing.T) {
	cv("StrLen", t, func() {
		truncate()

		cv("不存在的 key 长度为 0", func() {
			n, err := testRed.StrLen(testCtx, "sl_missing").Result()
			so(err, isNil)
			so(n, eq, int64(0))
		})

		cv("返回值的字节长度", func() {
			so(testRed.Set(testCtx, "sl_key", "hello", 0).Err(), isNil)
			n, err := testRed.StrLen(testCtx, "sl_key").Result()
			so(err, isNil)
			so(n, eq, int64(5))
		})

		cv("已过期 key 长度为 0", func() {
			so(testRed.SetEx(testCtx, "sl_exp", "abc", 100*time.Millisecond).Err(), isNil)
			time.Sleep(200 * time.Millisecond)
			n, err := testRed.StrLen(testCtx, "sl_exp").Result()
			so(err, isNil)
			so(n, eq, int64(0))
		})
	})
}

func TestGetRange(t *testing.T) {
	cv("GetRange", t, func() {
		truncate()
		so(testRed.Set(testCtx, "gr_key", "Hello, World!", 0).Err(), isNil)

		cv("正数下标", func() {
			s, err := testRed.GetRange(testCtx, "gr_key", 0, 4).Result()
			so(err, isNil)
			so(s, eq, "Hello")
		})

		cv("负数下标从末尾计算", func() {
			s, err := testRed.GetRange(testCtx, "gr_key", -6, -1).Result()
			so(err, isNil)
			so(s, eq, "World!")
		})

		cv("end 超出长度时截断", func() {
			s, err := testRed.GetRange(testCtx, "gr_key", 0, 999).Result()
			so(err, isNil)
			so(s, eq, "Hello, World!")
		})

		cv("不存在的 key 返回空串", func() {
			s, err := testRed.GetRange(testCtx, "gr_missing", 0, 10).Result()
			so(err, isNil)
			so(s, eq, "")
		})
	})
}

func TestSetRange(t *testing.T) {
	cv("SetRange", t, func() {
		truncate()

		cv("不存在的 key 用 \\x00 填充到 offset", func() {
			n, err := testRed.SetRange(testCtx, "sr_key", 5, "Hi").Result()
			so(err, isNil)
			so(n, eq, int64(7))
			val, _ := testRed.Get(testCtx, "sr_key").Result()
			so(val, eq, "\x00\x00\x00\x00\x00Hi")
		})

		cv("在中间覆写", func() {
			so(testRed.Set(testCtx, "sr_over", "Hello World", 0).Err(), isNil)
			n, err := testRed.SetRange(testCtx, "sr_over", 6, "Redis").Result()
			so(err, isNil)
			so(n, eq, int64(11))
			val, _ := testRed.Get(testCtx, "sr_over").Result()
			so(val, eq, "Hello Redis")
		})

		cv("offset+写入长度超过原串时扩展", func() {
			so(testRed.Set(testCtx, "sr_ext", "Hi", 0).Err(), isNil)
			n, err := testRed.SetRange(testCtx, "sr_ext", 5, "end").Result()
			so(err, isNil)
			so(n, eq, int64(8))
			val, _ := testRed.Get(testCtx, "sr_ext").Result()
			so(val, eq, "Hi\x00\x00\x00end")
		})
	})
}

func TestLCS(t *testing.T) {
	cv("LCS 未实现", t, func() {
		cmd := testRed.LCS(testCtx, &redis.LCSQuery{Key1: "a", Key2: "b"})
		so(cmd.Err(), notNil)
	})
}

// ---------------------------------------------------------------------------
// 自定义表名
// ---------------------------------------------------------------------------

func TestCustomTableName(t *testing.T) {
	cv("Options.TableName 指向非默认表", t, func() {
		const altTable = "t_redsql_kv_custom_test"

		mustExec("DROP TABLE IF EXISTS `" + altTable + "`")
		ddl := strings.ReplaceAll(createTableSQL, redsql.DefaultTableName, altTable)
		mustExec(ddl)
		defer mustExec("DROP TABLE IF EXISTS `" + altTable + "`")

		cli := &sqlxutil.SqlxWrapper{DB: testDB}
		r := redsql.New(cli, redsql.Options{TableName: altTable})

		so(r.Set(testCtx, "custom_k", "custom_v", 0).Err(), isNil)
		v, err := r.Get(testCtx, "custom_k").Result()
		so(err, isNil)
		so(v, eq, "custom_v")

		// 数据应只存在于自定义表，默认表上同一 key 不可见
		so(errors.Is(testRed.Get(testCtx, "custom_k").Err(), redis.Nil), eq, true)
	})
}
