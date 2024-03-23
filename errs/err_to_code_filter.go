package errs

import (
	"context"
	"reflect"
	"strings"

	"github.com/Andrew-M-C/trpc-go-utils/plugin"
	"trpc.group/trpc-go/trpc-go/filter"
)

const (
	// Filter 名称
	ErrToCodeFilterName = "err_to_code"
	// Plugin 配置类型, 用于指定 json tag
	ErrToCodePluginType = "filter"
	// Plugin 配置名
	ErrToCodePluginName = "err_to_code"
)

// RegisterErrToCodeFilter 注册 err_to_code filter, 该 filter 可以将 tRPC 函数实现中
// 返回的 err 类型转换为 response 中的 code / message 字段, 也可以作为 client filter,
// 将 code / message 数据提取出来, 放在 err 字段中, 让 RPC 的调用方和实现方对 RPC 可以有
// 如函数调用般的体验, 可以只判断 error 类型字段而无需再判断 rsp 中的 code 字段。
func RegisterErrToCodeFilter() {
	filter.Register(
		ErrToCodeFilterName,
		internal.filterSingleton.serverFilter,
		internal.filterSingleton.clientFilter,
	)
	plugin.Bind(
		ErrToCodePluginType, ErrToCodePluginName,
		&internal.filterSingleton.conf,
	)
}

type errToCodeFilter struct {
	conf errToCodeConfig
}

func (f *errToCodeFilter) serverFilter(ctx context.Context, req any, next filter.ServerHandleFunc) (any, error) {
	rsp, err := next(ctx, req)
	if err == nil {
		return rsp, err
	}

	code, msg := ExtractCodeMessageDigest[int64](err)
	rsp = f.setCode(rsp, code)
	rsp = f.setMsg(rsp, msg)
	return rsp, nil
}

func (f *errToCodeFilter) setCode(rsp any, code int64) any {
	typ := reflect.TypeOf(rsp)
	if typ.Kind() != reflect.Pointer || typ.Elem().Kind() != reflect.Struct {
		return rsp // 不是 *struct 类型, 什么都不做
	}

	val := reflect.ValueOf(rsp)
	if val.IsNil() {
		val = reflect.New(typ.Elem())
		rsp = val.Interface()
	}

	for i := 0; i < val.Elem().NumField(); i++ {
		tag := getJSONTag(val.Elem().Type().Field(i))
		if _, exist := f.conf.GetCodeTags()[tag]; !exist {
			continue
		}

		field := val.Elem().Field(i)
		switch field.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			field.SetInt(code)
			return rsp
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			field.SetUint(uint64(code))
			return rsp
		default:
			continue
		}
	}
	return rsp
}

func (f *errToCodeFilter) setMsg(rsp any, msg string) any {
	typ := reflect.TypeOf(rsp)
	if typ.Kind() != reflect.Pointer || typ.Elem().Kind() != reflect.Struct {
		return rsp // 不是 *struct 类型, 什么都不做
	}
	if rsp == nil {
		// 如果 rsp 为空, 则 new 一个 struct 出来
		rsp = reflect.New(typ.Elem()).Interface()
	}

	val := reflect.ValueOf(rsp).Elem()
	for i := 0; i < val.NumField(); i++ {
		tag := getJSONTag(val.Type().Field(i))
		if _, exist := f.conf.GetMsgTags()[tag]; !exist {
			continue
		}

		field := val.Field(i)
		switch field.Kind() {
		case reflect.String:
			field.SetString(msg)
			return rsp
		default:
			continue
		}
	}
	return rsp
}

func getJSONTag(f reflect.StructField) string {
	tag := f.Tag.Get("json")
	if tag == "" {
		return ""
	}
	parts := strings.Split(tag, ",")
	return parts[0]
}

func (f *errToCodeFilter) clientFilter(ctx context.Context, req, rsp any, next filter.ClientHandleFunc) error {
	err := next(ctx, req, rsp)
	if err != nil {
		return err // 应该是框架错误, 直接返回
	}
	if rsp == nil {
		return nil
	}

	typ := reflect.TypeOf(rsp)
	if typ.Kind() != reflect.Pointer || typ.Elem().Kind() != reflect.Struct {
		return nil // 不是 *struct 类型, 也不处理
	}

	val := reflect.ValueOf(rsp).Elem()
	code, ok := f.getCode(val)
	if !ok || code == 0 {
		return nil
	}
	msg, _ := f.getMsg(val)
	return New(code, msg)
}

func (f *errToCodeFilter) getCode(val reflect.Value) (int64, bool) {
	for i := 0; i < val.NumField(); i++ {
		tag := getJSONTag(val.Type().Field(i))
		if _, exist := f.conf.GetCodeTags()[tag]; !exist {
			continue
		}

		field := val.Field(i)
		switch field.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return field.Int(), true
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return int64(field.Uint()), true
		default:
			continue
		}
	}
	return -1, false
}

func (f *errToCodeFilter) getMsg(val reflect.Value) (string, bool) {
	for i := 0; i < val.NumField(); i++ {
		tag := getJSONTag(val.Type().Field(i))
		if _, exist := f.conf.GetMsgTags()[tag]; !exist {
			continue
		}

		field := val.Field(i)
		switch field.Kind() {
		case reflect.String:
			return field.String(), true
		default:
			continue
		}
	}
	return "", false
}

// ==== 以下是 filter 配置 ====

type errToCodeConfig struct {
	Tags struct {
		Code []string `yaml:"code"`
		Msg  []string `yaml:"msg"`
	} `yaml:"tags"`

	codeTags map[string]struct{}
	msgTags  map[string]struct{}
}

func (c *errToCodeConfig) GetCodeTags() map[string]struct{} {
	if c == nil {
		return map[string]struct{}{}
	}
	if c.codeTags != nil {
		return c.codeTags
	}

	keys := internal.defaultCodeTags
	if len(c.Tags.Code) > 0 {
		keys = c.Tags.Code
	}

	res := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		res[k] = struct{}{}
	}
	c.codeTags = res
	return res
}

func (c *errToCodeConfig) GetMsgTags() map[string]struct{} {
	if c == nil {
		return map[string]struct{}{}
	}
	if c.msgTags != nil {
		return c.msgTags
	}

	keys := internal.defaultMsgTags
	if len(c.Tags.Msg) > 0 {
		keys = c.Tags.Msg
	}

	res := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		res[k] = struct{}{}
	}
	c.msgTags = res
	return res
}
