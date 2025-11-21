package file

import "trpc.group/trpc-go/trpc-go/plugin"

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
	return decoder.Decode(&internal.configs)
}

func findConfigItem(name string) (pluginConfigItem, bool) {
	for _, config := range internal.configs {
		if config.Name == name {
			return config, true
		}
	}
	return pluginConfigItem{}, false
}
