package tracelog

import (
	"encoding/json"
	"fmt"
)

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
