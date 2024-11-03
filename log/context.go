package log

import (
	"context"
	"fmt"

	jsonvalue "github.com/Andrew-M-C/go.jsonvalue"
)

type fieldKey struct{}

type fieldValue struct {
	nesting context.Context
	field   string
	setter  func() *jsonvalue.V
}

func ctxWithKV(parent context.Context, field string, f func() *jsonvalue.V) context.Context {
	value := &fieldValue{
		nesting: parent,
		field:   field,
		setter:  f,
	}
	return context.WithValue(parent, fieldKey{}, value)
}

func getKV(ctx context.Context) *fieldValue {
	v := ctx.Value(fieldKey{})
	if v == nil {
		return nil
	}
	fv, _ := v.(*fieldValue)
	return fv
}

func CtxWithAny(parent context.Context, field string, value any) context.Context {
	return ctxWithKV(parent, field, func() *jsonvalue.V {
		j, err := jsonvalue.Import(value)
		if err != nil {
			s := fmt.Sprintf("%+v", value)
			return jsonvalue.NewString(s)
		}
		return j
	})
}

func CtxWithStr(parent context.Context, field string, s string) context.Context {
	return ctxWithKV(parent, field, func() *jsonvalue.V {
		return jsonvalue.NewString(s)
	})
}

func CtxWithStringer(parent context.Context, field string, s fmt.Stringer) context.Context {
	return ctxWithKV(parent, field, func() *jsonvalue.V {
		return jsonvalue.NewString(s.String())
	})
}

func CtxWithBool(parent context.Context, field string, b bool) context.Context {
	return ctxWithKV(parent, field, func() *jsonvalue.V {
		return jsonvalue.NewBool(b)
	})
}

func CtxWithInt(parent context.Context, field string, i int) context.Context {
	return ctxWithKV(parent, field, func() *jsonvalue.V {
		return jsonvalue.NewInt(i)
	})
}

func CtxWithUint(parent context.Context, field string, u uint) context.Context {
	return ctxWithKV(parent, field, func() *jsonvalue.V {
		return jsonvalue.NewUint(u)
	})
}

func CtxWithInt64(parent context.Context, field string, i int64) context.Context {
	return ctxWithKV(parent, field, func() *jsonvalue.V {
		return jsonvalue.NewInt64(i)
	})
}

func CtxWithUint64(parent context.Context, field string, u uint64) context.Context {
	return ctxWithKV(parent, field, func() *jsonvalue.V {
		return jsonvalue.NewUint64(u)
	})
}

func CtxWithFloat32(parent context.Context, field string, f float32) context.Context {
	return ctxWithKV(parent, field, func() *jsonvalue.V {
		return jsonvalue.NewFloat32(f)
	})
}

func CtxWithFloat64(parent context.Context, field string, f float64) context.Context {
	return ctxWithKV(parent, field, func() *jsonvalue.V {
		return jsonvalue.NewFloat64(f)
	})
}

func extractCtxKVs(ctx context.Context, tgt *jsonvalue.V) {
	if ctx == nil {
		return
	}
	for v := getKV(ctx); v != nil; v = getKV(v.nesting) {
		if _, err := tgt.Get(v.field); err == nil {
			continue // 不覆盖, 后来者优先
		}
		tgt.MustSet(v.setter()).At(v.field)
	}
}
