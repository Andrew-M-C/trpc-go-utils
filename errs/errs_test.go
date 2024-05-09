package errs_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Andrew-M-C/trpc-go-utils/errs"
	"github.com/smartystreets/goconvey/convey"
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
