package config

import "trpc.group/trpc-go/trpc-go/metrics"

const logPrefix = "[amc.util.config]"

var internal = struct {
	unmarshalerByName map[Encoding]unmarshaler
}{
	unmarshalerByName: map[Encoding]unmarshaler{
		JSON: jsonUnmarshaler{},
		YAML: yamlUnmarshaler{},
		TEXT: textUnmarshaler{},
	},
}

type unmarshaler interface {
	Unmarshal([]byte, any) error
}

func count(name string) {
	metrics.IncrCounter("amc.utils.config."+name, 1)
}

// E 表示内部错误
type E string

func (e E) Error() string {
	return string(e)
}
