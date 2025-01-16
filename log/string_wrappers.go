package log

import (
	"encoding/json"
	"fmt"
)

// ---- 直接 JSON ----

// ToJSON 在打日志的时候转为 JSON string
func ToJSON(v any) fmt.Stringer {
	return jsonWrapper{v: v}
}

type jsonWrapper struct {
	v any
}

func (j jsonWrapper) String() string {
	b, err := json.Marshal(j.v)
	if err != nil {
		return fmt.Sprint(j.v)
	}
	return string(b)
}

// ---- error string 为 JSON ----

func ToErrJSON(e error) fmt.Stringer {
	return errJSONWapper{e: e}
}

type errJSONWapper struct {
	e error
}

func (e errJSONWapper) String() string {
	b, _ := json.Marshal(e.e.Error())
	return string(b)
}

// ---- string 为 JSON ----

func ToJSONString(v any) fmt.Stringer {
	return jsonStrWrapper{v: v}
}

type jsonStrWrapper struct {
	v any
}

func (j jsonStrWrapper) String() string {
	b, _ := json.Marshal(fmt.Sprint(j.v))
	return string(b)
}
