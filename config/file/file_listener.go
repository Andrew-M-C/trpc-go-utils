package file

import (
	"context"
	"fmt"
	"os"

	"github.com/Andrew-M-C/go.util/runtime/caller"
	"github.com/Andrew-M-C/go.util/unsafe"
	"github.com/Andrew-M-C/trpc-go-utils/log"
	"github.com/Andrew-M-C/trpc-go-utils/recovery"
	"github.com/fsnotify/fsnotify"
	"trpc.group/trpc-go/trpc-go/config"
)

type fileListener struct {
	key     string
	path    string
	watcher *fsnotify.Watcher
	ch      chan config.Response
}

func newWatcher(name string) (*fileListener, error) {
	conf, ok := findConfigItem(name)
	if !ok {
		return nil, fmt.Errorf("config '%s' not found", name)
	}
	// 检查文件确实是一个文件
	if err := checkPathIsFileAndReadable(conf.Path); err != nil {
		return nil, fmt.Errorf("check path '%s' error: %w", conf.Path, err)
	}
	// 使用 fsnotify 监听文件变化
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("create fsnotify watcher error: %w", err)
	}
	if err := watcher.Add(conf.Path); err != nil {
		return nil, fmt.Errorf("add path '%s' to fsnotify watcher error: %w", conf.Path, err)
	}
	l := &fileListener{
		key:     name,
		path:    conf.Path,
		watcher: watcher,
		ch:      make(chan config.Response, 10),
	}
	go l.doWatch()
	return l, nil
}

func checkPathIsFileAndReadable(path string) error {
	fi, err := os.Stat(path)
	if err != nil {
		return err
	}
	if fi.IsDir() {
		return fmt.Errorf("path %s is a directory", path)
	}
	if fi.Mode()&0400 == 0 {
		return fmt.Errorf("path %s is not readable", path)
	}
	return nil
}

func (l *fileListener) doWatch() {
	defer recovery.CatchPanic(
		recovery.WithErrorLog(),
		recovery.WithCallback(func(context.Context, any, []caller.Caller) {
			go l.doWatch()
		}),
	)
	for {
		select {
		case event := <-l.watcher.Events:
			l.handleFileChange(event)

		case err := <-l.watcher.Errors:
			log.New().With("path", l.path).Err(err).Text("监听配置文件失败").Error()
		}
	}
}

func (l *fileListener) handleFileChange(ev fsnotify.Event) {
	logger := log.New().With("name", l.key).With("path", l.path).With("op", ev.Op)

	if ev.Has(fsnotify.Write) {
		b, err := os.ReadFile(l.path)
		if err != nil {
			logger.Err(err).Text("读取文件内容失败").Error()
			return
		}

		logger.Text("监听到文件写入").Info()
		c := &content{
			value: unsafe.BtoS(b),
			event: config.EventTypePut,
		}
		internal.cache.Store(l.key, c.value)
		l.ch <- c
	}

	if ev.Has(fsnotify.Remove) {
		logger.Text("监听到文件删除").Info()
		c := &content{
			value: "",
			event: config.EventTypeDel,
		}
		l.ch <- c
	}

	logger.Text("无需关注的文件变更类型").Info()
}
