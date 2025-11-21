package file

import (
	"fmt"
	"os"

	"github.com/Andrew-M-C/go.util/unsafe"
	"trpc.group/trpc-go/trpc-go/config"
)

type content struct {
	value string
	event config.EventType
}

func readFile(key string) (*content, error) {
	conf, ok := findConfigItem(key)
	if !ok {
		return nil, fmt.Errorf("config '%s' not found", key)
	}
	b, err := os.ReadFile(conf.Path)
	if err != nil {
		return nil, err
	}
	c := &content{
		value: unsafe.BtoS(b),
		event: config.EventTypeNull,
	}
	return c, nil
}

func (c *content) Value() string {
	return c.value
}

func (*content) MetaData() map[string]string {
	return map[string]string{}
}

func (c *content) Event() config.EventType {
	return c.event
}
