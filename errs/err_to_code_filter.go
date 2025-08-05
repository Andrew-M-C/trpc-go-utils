package errs

import (
	"context"
	"reflect"
	"strings"

	"github.com/Andrew-M-C/trpc-go-utils/log"
	"github.com/Andrew-M-C/trpc-go-utils/plugin"
	"trpc.group/trpc-go/trpc-go/filter"
)

const (
	// Filter 名称
	ErrToCodeFilterName = "err_to_code"
	CodeToFilterName    = "code_to_err"
	// Plugin 配置类型, 用于指定 json tag
	ErrToCodePluginType = "filter"
	// Plugin 配置名
	ErrToCodePluginName = "err_to_code"
)

// ErrorCodeMsgSetterFunc 表示当提取出错误时, 可以回调给业务方进行 set 操作。如果没有这个
// setter 的话, filter 会自行使用反射进行设置。
//
// 返回 true 表示已经设置 OK, 否则 filter 会继续往下走继续使用反射逻辑进行设置
type ErrorCodeMsgSetterFunc func(ctx context.Context, rsp any, code int, msg string) bool

// RegisterErrToCodeFilter 注册 err_to_code filter, 该 filter 可以将 tRPC 函数实现中
// 返回的 err 类型转换为 response 中的 code / message 字段, 也可以作为 client filter,
// 将 code / message 数据提取出来, 放在 err 字段中, 让 RPC 的调用方和实现方对 RPC 可以有
// 如函数调用般的体验, 可以只判断 error 类型字段而无需再判断 rsp 中的 code 字段。
//
// 必须在 trpc.NewServer 之前调用
func RegisterErrToCodeFilter() {
	internal.filterInitOnce.Do(registerAndBindConfig)
}

func registerAndBindConfig() {
	filter.Register(
		ErrToCodeFilterName,
		internal.filterSingleton.serverFilter,
		internal.filterSingleton.clientFilter,
	)
	filter.Register(
		CodeToFilterName,
		internal.filterSingleton.serverFilter,
		internal.filterSingleton.clientFilter,
	)
	plugin.Bind(
		ErrToCodePluginType, ErrToCodePluginName,
		&internal.filterSingleton.conf,
	)
}

// SetServerErrorCodeMsgSetter 设置在 err_to_code filter 中, 当提取出错误时, 的回调函数。
// 业务方可以使用这个回调进行 set 操作。如果没有这个 setter 的话, filter 会自行使用反射进行设置。
//
// 返回 true 表示已经设置 OK, 否则 filter 会继续往下走继续使用反射逻辑进行设置
func SetServerErrorCodeMsgSetter(fu ErrorCodeMsgSetterFunc) {
	if fu != nil {
		internal.filterSingleton.errorCodeMsgSetter = fu
	}
}

type errToCodeFilter struct {
	conf errToCodeConfig

	errorCodeMsgSetter ErrorCodeMsgSetterFunc
}

func (f *errToCodeFilter) serverFilter(ctx context.Context, req any, next filter.ServerHandleFunc) (any, error) {
	rsp, err := next(ctx, req)
	if err == nil {
		return rsp, err // 没有错误, 那么直接返回
	}

	code, msg := ExtractCodeMessageDigest[int64](err)
	rsp, ok := ensureAlloc(ctx, rsp)
	if !ok {
		return rsp, err
	}
	if setter := f.errorCodeMsgSetter; setter != nil {
		if done := setter(ctx, rsp, int(code), msg); done {
			return rsp, err
		}
	}

	rsp, ok = f.setCode(ctx, rsp, code)
	if !ok {
		return rsp, err
	}

	rsp = f.setMsg(rsp, msg)
	log.New().Text("err_to_code 转换").With("code", code).With("msg", msg).Err(err).WarnContext(ctx)
	return rsp, nil
}

func (f *errToCodeFilter) setCode(ctx context.Context, rsp any, code int64) (any, bool) {
	rsp, ok := ensureAlloc(ctx, rsp)
	if !ok {
		return rsp, false
	}

	val := reflect.ValueOf(rsp)

	for i := 0; i < val.Elem().NumField(); i++ {
		tag := getJSONTag(val.Elem().Type().Field(i))
		if _, exist := f.conf.GetCodeTags()[tag]; !exist {
			continue
		}

		field := val.Elem().Field(i)
		switch field.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			log.New().Text("设置返回 code 值").With("code", code).With("field", field).DebugContext(ctx)
			field.SetInt(code)
			return rsp, true
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			log.New().Text("设置返回 code 值").With("code", code).With("field", field).DebugContext(ctx)
			field.SetUint(uint64(code))
			return rsp, true
		default:
			continue
		}
	}
	log.New().Text("没有合法的 field 值可供设置, 放弃").With("code", code).DebugContext(ctx)
	return rsp, false
}

func (f *errToCodeFilter) setMsg(rsp any, msg string) any {
	typ := reflect.TypeOf(rsp)
	if typ == nil {
		return rsp // untyped nil, 无法生成响应
	}
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
