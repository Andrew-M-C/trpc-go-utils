package errs

import "github.com/Andrew-M-C/go.util/errors"

var internal = struct {
	unknownErrorCode int32
	unknownErrorMsg  *string

	successCode int32
	successMsg  *string

	digestDesc *string
	hashFunc   func(error) string

	filterSingleton *errToCodeFilter

	defaultCodeTags []string
	defaultMsgTags  []string
}{}

func init() {
	unknown := "undefined error"
	internal.unknownErrorCode = -1
	internal.unknownErrorMsg = &unknown

	succ := "success"
	internal.successCode = 0
	internal.successMsg = &succ

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
