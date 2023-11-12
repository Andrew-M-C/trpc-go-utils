package mysql

import (
	"fmt"

	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/naming/registry"
	"trpc.group/trpc-go/trpc-go/plugin"
)

type configParser struct{}

type mysqlConfig struct {
	MySQLName string               `yaml:"mysql_name"` // 制定一个 MySQL trpc client 名字
	Service   []mysqlConfigService `yaml:"service"`
}

type mysqlConfigService struct {
	Name   string `yaml:"name"`
	Weight int    `yaml:"weight"`
}

// 读取权重, nil 安全
func (s *mysqlConfigService) GetWeight() int {
	if s == nil {
		return 100
	}
	if s.Weight <= 0 {
		return 100
	}
	return s.Weight
}

// GetName 获取服务名称
func (s *mysqlConfigService) GetName() string {
	if s == nil {
		return ""
	}
	return s.Name
}

// 读取 client name, nil 安全
func (c *mysqlConfig) GetMySQLName() string {
	if c == nil {
		return ""
	}
	return c.MySQLName
}

// GetService 找到一个 service 配置
func (c *mysqlConfig) GetService(name string) mysqlConfigService {
	for _, svc := range c.Service {
		if svc.Name == name {
			return svc
		}
	}
	return mysqlConfigService{}
}

func initPlugin() {
	plugin.Register(PluginName, configParser{})
}

func (configParser) Type() string {
	return PluginType
}

func (configParser) Setup(name string, decoder plugin.Decoder) error {
	if decoder == nil {
		return fmt.Errorf("nil decoder for plugin name '%s'", name)
	}

	c := mysqlConfig{}
	if err := decoder.Decode(&c); err != nil {
		return fmt.Errorf("decode config for plugin with name '%s' error: %w", name, err)
	}
	if c.MySQLName == "" {
		return fmt.Errorf("未配置有效的 plugins.%s.%s.mysql_name 字段", PluginType, PluginName)
	}

	// 注册注册器
	for i, svc := range c.Service {
		if svc.GetName() == "" {
			log.Warnf("plugins.%s.%s.[%d].name 为空", PluginType, PluginName, i)
			continue
		}
		registry.Register(svc.GetName(), internal.register)
	}

	internal.config = &c
	return nil
}
