package tracelog

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/filter"
	thttp "trpc.group/trpc-go/trpc-go/http"
	"trpc.group/trpc-go/trpc-go/log"
)

const (
	// tracelog 的 filter 名称
	FilterName = "tracelog"
	// TraceIDMetadataKey 定义用于传递 trace ID 的 trpc metadata 字段
	TraceIDStackMetadataKey = "trace_id_stack"
)

// RegisterTraceLogFilter 注册 tracelog filter。请在 trpc.NewServer 之前调用。
func RegisterTraceLogFilter() {
	filter.Register(FilterName, serverFilter, clientFilter)
}

func serverFilter(ctx context.Context, req any, next filter.ServerHandleFunc) (rsp any, err error) {
	ctx, msg := codec.EnsureMessage(ctx)
	var stack []string
	if b := msg.ServerMetaData()[TraceIDStackMetadataKey]; len(b) > 0 {
		_ = json.Unmarshal(b, &stack)
	}
	if len(stack) == 0 {
		ctx = EnsureTraceID(ctx)
	} else {
		ctx = WithTraceIDStack(ctx, stack)
	}

	caller := func() string {
		if addr := codec.Message(ctx).RemoteAddr(); addr != nil {
			return addr.String()
		}
		return "unknown"
	}()
	metadata := codec.Message(ctx).ServerMetaData()
	httpReq := thttp.Request(ctx)
	start := time.Now()
	rsp, err = next(ctx, req)
	ela := time.Since(start)

	if err != nil {
		log.WarnContextf(ctx,
			`{"caller":%v,"http_req":%v,"server_metadata":%v,`+
				`"req":%v,"req_type":%v,"rsp":%v,"rsp_type":%v,`+
				`"cost":"%v","error":%v}`,
			ToJSON(caller), ToJSON(httpReq), ToJSON(metadata),
			ToJSON(req), typeString(reflect.TypeOf(req)), ToJSON(rsp), typeString(reflect.TypeOf(rsp)),
			ela, ToErrJSON(err),
		)
	} else {
		log.DebugContextf(ctx,
			`{"caller":%v,"http_req":%v,"server_metadata":%v,`+
				`"req":%v,"req_type":%v,"rsp":%v,"rsp_type":%v,`+
				`"cost":"%v"}`,
			ToJSON(caller), ToJSON(httpReq), ToJSON(metadata),
			ToJSON(req), typeString(reflect.TypeOf(req)), ToJSON(rsp), typeString(reflect.TypeOf(rsp)),
			ela,
		)
	}
	return
}

func clientFilter(ctx context.Context, req, rsp any, next filter.ClientHandleFunc) (err error) {
	// ctx = EnsureTraceID(ctx)
	if stack := TraceIDStack(ctx); len(stack) > 0 {
		b, _ := json.Marshal(stack)
		trpc.SetMetaData(ctx, TraceIDStackMetadataKey, b)
	} else {
		b, _ := json.Marshal([]string{generateTraceID()})
		trpc.SetMetaData(ctx, TraceIDStackMetadataKey, b)
	}

	callee := func() string {
		if addr := codec.Message(ctx).RemoteAddr(); addr != nil {
			return addr.String()
		}
		return "unknown"
	}()
	metadata := codec.Message(ctx).ServerMetaData()
	httpReq := thttp.Request(ctx)
	start := time.Now()
	err = next(ctx, req, rsp)
	ela := time.Since(start)

	if err != nil {
		log.ErrorContextf(ctx,
			`{"callee":%v,"http_req":%v,"server_metadata":%v,`+
				`"req":%v,"req_type":%v,"rsp":%v,"rsp_type":%v,`+
				`"cost":"%v"`,
			ToJSON(callee), ToJSON(httpReq), ToJSON(metadata),
			ToJSON(req), typeString(reflect.TypeOf(req)), ToJSON(rsp), typeString(reflect.TypeOf(rsp)),
			ela, ToErrJSON(err),
		)
	} else {
		log.DebugContextf(ctx,
			`{"callee":%v,"http_req":%v,"server_metadata":%v,`+
				`"req":%v,"req_type":%v,"rsp":%v,"rsp_type":%v,`+
				`"cost":"%v"}`,
			ToJSON(callee), ToJSON(httpReq), ToJSON(metadata),
			ToJSON(req), typeString(reflect.TypeOf(req)), ToJSON(rsp), typeString(reflect.TypeOf(rsp)),
			ela,
		)
	}
	return err
}

func typeString(t reflect.Type) fmt.Stringer {
	prefix := ""
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
		prefix = "*"
	}
	s := fmt.Sprintf("%s.%s%s", t.PkgPath(), prefix, t.Name())
	return ToJSONString(s)
}
