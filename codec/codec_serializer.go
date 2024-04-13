// Package codec 提供一些覆盖 codec 的工具
package codec

import (
	"encoding/json"

	"trpc.group/trpc-go/trpc-go/codec"
)

func UseOfficialJSON() {
	codec.RegisterSerializer(codec.SerializationTypeJSON, jsonSerializer{})
}

type jsonSerializer struct{}

func (jsonSerializer) Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func (jsonSerializer) Unmarshal(b []byte, v any) error {
	return json.Unmarshal(b, v)
}
