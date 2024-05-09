// Package errs 实现基于 tRPC errs 的错误功能封装
package errs

import (
	"errors"
	"fmt"
	"strings"

	"golang.org/x/exp/constraints"
	"trpc.group/trpc-go/trpc-go/errs"
)

var (
	// UndefinedError 表示未定义错误
	UndefinedError = errs.New(999, "undefined error").(*errs.Error)
)

// New 是对 tRPC errs New 的简单封装, 避免引入两个同名 package
func New[T errs.ErrCode](code T, msg string) error {
	return errs.New(code, msg)
}

// SetDigestDescription 当不暴露具体错误, 但给出摘要值时, 摘要值的描述名是什么
// (默认是 "digest code")
func SetDigestDescription(desc string) {
	internal.digestDesc = &desc
}

// SetDigestFunction 设置错误描述的摘要函数。默认值为
// github.com/Andrew-M-C/go.util/errors.ErrorToCode
func SetDigestFunction(f func(error) string) {
	if f != nil {
		internal.hashFunc = f
	}
}

// ExtractCodeMessageDigest 提取错误中的 code 和描述, 如果是对 trpc errs 的 wrapping,
// 则在 message 之后对完整信息进行摘要。
func ExtractCodeMessageDigest[T constraints.Integer](err error) (code T, msg string) {
	defer func() {
		msg = singleLine(msg)
	}()

	if err == nil {
		return 0, "success"
	}

	if trpcErr, ok := err.(*errs.Error); ok {
		return T(trpcErr.Code), trpcErr.Msg
	}

	var trpcErr *errs.Error
	if !errors.As(err, &trpcErr) {
		code = T(UndefinedError.Code)
		msg = fmt.Sprintf(
			"%s, %s: %s",
			UndefinedError.Msg, *internal.digestDesc, internal.hashFunc(err),
		)
		return
	}

	code = T(trpcErr.Code)
	msg = fmt.Sprintf(
		"%s, %s: %s",
		trpcErr.Msg, *internal.digestDesc, internal.hashFunc(err),
	)
	return
}

// ExtractCodeMessage 提取错误中的 code 和描述, 信息不摘要, 只保留可显示的 msg
func ExtractCodeMessage[T constraints.Integer](err error) (code T, msg string) {
	defer func() {
		msg = singleLine(msg)
	}()

	if err == nil {
		return 0, "success"
	}

	var trpcErr *errs.Error
	if !errors.As(err, &trpcErr) {
		code = T(UndefinedError.Code)
		msg = UndefinedError.Msg
		return
	}

	code = T(trpcErr.Code)
	msg = trpcErr.Msg
	return
}

func singleLine(s string) string {
	return strings.ReplaceAll(s, "\n", "\\n")
}
