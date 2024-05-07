package etcd

import (
	"context"
	"fmt"

	configutil "github.com/Andrew-M-C/trpc-go-utils/config"
	"gopkg.in/yaml.v3"
	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/server"
)

const (
	// ErrClientYAML 表示 client YAML 配置错误
	ErrClientYAML = E("amc etcd client YAML error")
)

// ReplaceGlobalClient 支持通过 etcd 修改 client 配置, 并且覆盖 tRPC global config
func ReplaceGlobalClient(ctx context.Context, s *server.Service, key string) error {
	if s == nil {
		return fmt.Errorf("%w: tRPC server not yet initialized", ErrClientYAML)
	}
	newConfig, watcher, err := configutil.Watch[trpc.Config](
		ctx, API{}, configutil.YAML, key,
	)
	if err != nil {
		return fmt.Errorf(
			"%w: watch config %s error: '%v'",
			ErrClientYAML, key, err,
		)
	}

	updateClientYAML(newConfig.Client)
	go doWatch(key, watcher)
	return nil
}

func updateClientYAML(client trpc.ClientConfig) {
	// 填充 callee 字段, 在 trpc 中使用的是这个字段
	for _, cli := range client.Service {
		if cli.Callee == "" {
			cli.Callee = cli.ServiceName
		}
	}

	trpcGlobal := deepCopyGlobalConfig(trpc.GlobalConfig())
	trpcGlobal.Client = client

	if err := trpc.SetupClients(&client); err != nil {
		log.Errorf("%s Setup clients failed, raw config '%+v', error: '%v'", client, err)
		count("clientUpdate.fail")
		return
	}

	trpc.SetGlobalConfig(trpcGlobal)
	count("clientUpdate.succ")
}

func deepCopyGlobalConfig(in *trpc.Config) *trpc.Config {
	b, _ := yaml.Marshal(in)

	var out trpc.Config
	_ = yaml.Unmarshal(b, &out)
	return &out
}

func doWatch(key string, watcher <-chan *trpc.Config) {
	log.Debugf("%s Start watching client yaml config '%s'", logPrefix, key)

	for updatedClient := range watcher {
		updateClientYAML(updatedClient.Client)
	}
}
