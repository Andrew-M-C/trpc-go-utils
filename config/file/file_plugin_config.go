package file

import (
	"fmt"

	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/plugin"
)

type pluginConfigItem struct {
	Name string `yaml:"name"`
	Path string `yaml:"path"`
}

func registerPlugin() {
	plugin.Register("file", pluginFactory{})
}

type pluginFactory struct{}

func (pluginFactory) Type() string {
	return "config"
}

func (pluginFactory) Setup(name string, decoder plugin.Decoder) error {
	log.Debugf("开始解析 plugin %s", name)
	if err := decoder.Decode(&internal.configs); err != nil {
		return fmt.Errorf("decode config %s error: '%w'", name, err)
	}
	log.Infof("解析 plugin %s 成功, 配置: %+v", name, internal.configs)
	return nil
}

func findConfigItem(name string) (pluginConfigItem, bool) {
	for _, config := range internal.configs {
		if config.Name == name {
			return config, true
		}
	}
	return pluginConfigItem{}, false
}
