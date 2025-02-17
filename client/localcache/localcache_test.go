package localcache_test

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Andrew-M-C/trpc-go-utils/client/localcache"
	"github.com/smartystreets/goconvey/convey"
)

var (
	cv = convey.Convey
	so = convey.So
	eq = convey.ShouldEqual
	gt = convey.ShouldBeGreaterThan
	lt = convey.ShouldBeLessThan

	notNil = convey.ShouldNotBeNil
)

func printf(f string, a ...any) {
	_, _ = convey.Printf(f, a...)
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

type testCache struct {
	counter uint64
}

func TestGetWithCustomLoad(t *testing.T) {
	cv("测试多协程同时 load", t, func() {
		const concurrency = 100000
		const eachRepeat = 10
		const key = "key"

		loadCount := int64(0)

		cache := localcache.New[*testCache]()

		wg := sync.WaitGroup{}
		wg.Add(concurrency)

		allStart := time.Now()

		iter := func(iterStart time.Time) {
			defer wg.Done()
			time.Sleep(time.Second - iterStart.Sub(allStart)) // 延迟启动搞事情

			for i := 0; i < eachRepeat; i++ {
				c, err := cache.GetWithCustomLoad(
					context.Background(), key,
					func(context.Context, string) (*testCache, error) {
						time.Sleep(time.Second) // 模拟耗时
						atomic.AddInt64(&loadCount, 1)
						return &testCache{}, nil
					},
					time.Hour, // 等同于不过期
				)
				if err != nil {
					panic(err)
				}
				atomic.AddUint64(&c.counter, 1)
			}
		}
		for i := 0; i < concurrency; i++ {
			go iter(time.Now())
		}

		wg.Wait()
		c, exist := cache.Get(key)
		so(c, notNil)
		so(exist, eq, true)

		printf("加载次数: %d, 计数 %d", loadCount, c.counter)

		// 理想情况下, 我们可能会期望同一个 key 只 load 一次。
		// 但是实际情况下, 多个 load 并不是保证互斥的
		actualSituation := true
		if actualSituation {
			so(loadCount, gt, 1)
			so(c.counter, lt, concurrency*eachRepeat)
		} else {
			so(loadCount, eq, 1)
			so(c.counter, eq, concurrency*eachRepeat)
		}

	})
}
