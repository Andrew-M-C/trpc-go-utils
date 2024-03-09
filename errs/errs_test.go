package errs_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/Andrew-M-C/trpc-go-utils/errs"
	"github.com/glycerine/goconvey/convey"
)

var (
	cv = convey.Convey
	so = convey.So
	eq = convey.ShouldEqual
	ne = convey.ShouldNotEqual
)

func TestMain(m *testing.M) {
	_ = m.Run()
}

func TestSetUndefinedError(t *testing.T) {
	cv("SetUndefinedError", t, func() {
		const desc = "未定义错误"
		const code = 10000
		errs.SetUndefinedError(10000, desc)
		err := errors.New("db error")

		c, m := errs.ExtractCodeMessage[int](err)
		so(c, eq, code)
		so(m, eq, desc)

		c, m = errs.ExtractCodeMessageDigest[int](err)
		so(c, eq, code)
		so(m, ne, desc)
		so(strings.HasPrefix(m, desc), eq, true)
		t.Log(m)
	})
}

func TestSetSuccess(t *testing.T) {
	cv("SetSuccess", t, func() {
		const desc = "成功"
		const code = 200
		errs.SetSuccess(code, desc)

		c, m := errs.ExtractCodeMessage[int](nil)
		so(c, eq, code)
		so(m, eq, desc)
	})
}

func TestSetDigestDescription(t *testing.T) {
	cv("SetDigestDescription", t, func() {
		const code = 404
		const desc = "服务错误"
		err := errs.New(code, desc)
		err = fmt.Errorf("%w, 崩溃啦", err)

		c, m := errs.ExtractCodeMessageDigest[int](err)
		so(c, eq, code)
		so(m, ne, desc)
		so(m, ne, err.Error())
		so(strings.Contains(m, desc), eq, true)
	})
}
