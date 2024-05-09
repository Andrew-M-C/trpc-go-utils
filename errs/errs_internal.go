package errs

import "github.com/Andrew-M-C/go.util/errors"

var internal = struct {
	digestDesc *string
	hashFunc   func(error) string

	filterSingleton *errToCodeFilter

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
