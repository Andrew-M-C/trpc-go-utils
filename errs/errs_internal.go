package errs

import "github.com/Andrew-M-C/go.util/errors"

var internal = struct {
	unknownErrorCode int32
	unknownErrorMsg  *string

	successCode int32
	successMsg  *string

	digestDesc *string
	hashFunc   func(error) string
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
}
