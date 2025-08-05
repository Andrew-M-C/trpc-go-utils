package errs

import (
	"context"
	"reflect"
	"sync"

	"github.com/Andrew-M-C/go.util/errors"
	"github.com/Andrew-M-C/trpc-go-utils/log"
)

var internal = struct {
	digestDesc *string
	hashFunc   func(error) string

	filterSingleton *errToCodeFilter
	filterInitOnce  sync.Once

	defaultCodeTags []string
	defaultMsgTags  []string
}{}

func init() {
	s := "digest code"
	internal.digestDesc = &s

	internal.hashFunc = errors.ErrorToCode

	internal.filterSingleton = &errToCodeFilter{}

	internal.defaultCodeTags = []string{
		"code",
		"ret",
		"err_ret",
		"err_code",
	}
	internal.defaultMsgTags = []string{
		"message",
		"msg",
		"err_msg",
	}
}

func ensureAlloc(ctx context.Context, prototype any) (any, bool) {
	typ := reflect.TypeOf(prototype)
	if typ == nil {
		log.New().Text("req 可能是 untyped nil, 无法创建实例").DebugContext(ctx)
		return prototype, false
	}
	if typ.Kind() != reflect.Pointer || typ.Elem().Kind() != reflect.Struct {
		log.New().Text("数据类型不是 struct 指针, 无法创建实例").With("type", typ).DebugContext(ctx)
		return prototype, false // 不是 *struct 类型, 什么都不做
	}

	val := reflect.ValueOf(prototype)
	if val.IsNil() {
		val = reflect.New(typ.Elem())
		return val.Interface(), true
	}

	// 到这里说明 prototype 本身就是可用的
	return prototype, true
}
