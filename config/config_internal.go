package config

import (
	syncutil "github.com/Andrew-M-C/go.util/sync"
	"trpc.group/trpc-go/trpc-go/metrics"
)

const logPrefix = "[amc.util.config]"

var internal = struct {
	unmarshalerByName map[Encoding]unmarshaler
	watchersByKey     syncutil.Map[string, *watchAndDispatcher] // key: ("%s+%s", name, key)
}{
	unmarshalerByName: map[Encoding]unmarshaler{
		JSON: jsonUnmarshaler{},
		YAML: yamlUnmarshaler{},
		TEXT: textUnmarshaler{},
	},
	watchersByKey: syncutil.NewMap[string, *watchAndDispatcher](),
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
