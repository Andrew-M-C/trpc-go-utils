package config

import (
	"context"
	"sync"

	chanutil "github.com/Andrew-M-C/go.util/channel"
	syncutil "github.com/Andrew-M-C/go.util/sync"
	"github.com/google/uuid"
	"trpc.group/trpc-go/trpc-go/config"
	"trpc.group/trpc-go/trpc-go/log"
)

type watchAndDispatcher struct {
	notifiers syncutil.Map[string, chan config.Response]
	conf      config.KVConfig
	key       string

	lock     sync.Mutex
	started  bool
	watchErr error
}

func newWatchAndDispatcher(conf config.KVConfig, key string) *watchAndDispatcher {
	w := &watchAndDispatcher{}
	w.notifiers = syncutil.NewMap[string, chan config.Response]()
	w.conf = conf
	w.key = key
	return w
}

func (w *watchAndDispatcher) NewNotify() <-chan config.Response {
	ch := make(chan config.Response, 1) // 保留 1 个缓冲, 尽量确保 channel 里面是最新的
	w.notifiers.Store(uuid.NewString(), ch)
	count("watch.new.cnt")
	return ch
}

func (w *watchAndDispatcher) DoWatch(ctx context.Context) error {
	w.lock.Lock()
	defer w.lock.Unlock()

	if w.started {
		return w.watchErr
	}

	notify, err := w.conf.Watch(ctx, w.key)
	if err != nil {
		w.watchErr = err
		return err
	}

	w.started = true
	go w.watch(notify)
	return nil
}

func (w *watchAndDispatcher) watch(notify <-chan config.Response) {
	for res := range notify {
		w.notifiers.Range(func(id string, ch chan config.Response) bool {
			w.writeToChan(id, ch, res)
			return true
		})
	}
}

func (w *watchAndDispatcher) writeToChan(id string, ch chan config.Response, v config.Response) {
	full, closed := chanutil.WriteNonBlocked(ch, v)
	for full || closed {
		if closed {
			count("watch.closed.cnt")
			log.Warnf(
				"%s channel for key '%s' is closed, config name '%s'",
				w.key, w.conf.Name(),
			)
			w.notifiers.Delete(id)
			return
		}

		// 如果已经满了, 那么消费掉 channel 中的老数据, 写入最新的
		_, empty, _ := chanutil.ReadNonBlocked(ch)
		for !empty {
			count("watch.chanNotEmpty.cnt")
			log.Info(
				"%s channel for key '%s' is not empty, config name '%s'",
				w.key, w.conf.Name(),
			)
			_, empty, _ = chanutil.ReadNonBlocked(ch)
		}
	}

	// 写入完成
	count("watch.update.cnt")
}
