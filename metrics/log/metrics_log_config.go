package log

import (
	"os"
	"strings"

	"github.com/Andrew-M-C/trpc-go-utils/config"
	"trpc.group/trpc-go/trpc-go/log"
)

type sinkConfig struct {
	Name  string `json:"name"`  // 服务名称
	Level string `json:"level"` // 日志级别
}

func (impl *sinkImpl) registerPlugin() {
	impl.server = os.Args[0]

	config.RegisterPlugin(PluginType, PluginName, func(c *sinkConfig) error {
		if c.Name != "" {
			impl.server = c.Name
		}
		switch strings.TrimSpace(strings.ToLower(c.Level)) {
		default:
			fallthrough
		case "info":
			impl.logger = log.Infof
		case "debug":
			impl.logger = log.Debugf
		case "warn", "warning":
			impl.logger = log.Warnf
		case "error":
			impl.logger = log.Errorf
		}

		return nil
	})
}
