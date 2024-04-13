package config

import (
	"encoding/json"

	"gopkg.in/yaml.v3"
)

type jsonUnmarshaler struct{}

func (jsonUnmarshaler) Unmarshal(b []byte, tgt any) error {
	return json.Unmarshal(b, tgt)
}

type yamlUnmarshaler struct{}

func (yamlUnmarshaler) Unmarshal(b []byte, tgt any) error {
	return yaml.Unmarshal(b, tgt)
}
