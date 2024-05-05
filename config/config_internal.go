package config

import "trpc.group/trpc-go/trpc-go/metrics"

var internal = struct {
	unmarshalerByName map[Encoding]unmarshaler
}{
	unmarshalerByName: map[Encoding]unmarshaler{
		JSON: jsonUnmarshaler{},
		YAML: yamlUnmarshaler{},
	},
}

type unmarshaler interface {
	Unmarshal([]byte, any) error
}

func count(name string) {
	metrics.IncrCounter("utils.config."+name, 1)
}
