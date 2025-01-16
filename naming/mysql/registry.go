package mysql

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/Andrew-M-C/go.util/log/trace"
	"github.com/Andrew-M-C/go.util/maps"
	"github.com/Andrew-M-C/go.util/recovery"
	"github.com/Andrew-M-C/go.util/runtime/caller"
	"trpc.group/trpc-go/trpc-database/mysql"
	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/naming/registry"
)

func initRegistry() {
	internal.register = &registerImpl{
		nodes: maps.NewRWSafeMap[string, *node](1),
	}
	go internal.register.doUpdate()
}

type registerImpl struct {
	nodes maps.RWSafeMap[string, *node]
}

type node struct {
	// 数据库 ID, 仅作为 server 的时候有效
	id int64
	// trpc 节点
	node *registry.Node
	// 上一次更新时间
	lastUpdateTime *time.Time
}

func (n *node) GetLastUpdateTime() time.Time {
	if n == nil {
		return time.Time{}
	}
	tm := n.lastUpdateTime
	if tm == nil {
		return time.Time{}
	}
	return *tm
}

func (n *node) GetNode() *registry.Node {
	res := *n.node
	return &res
}

// 实现 registry.Registry 接口
func (r *registerImpl) Register(service string, opts ...registry.Option) (err error) {
	ctx := context.Background()
	opt := mergeRegistryOpts(opts)

	// 获取 client
	conf := internal.config
	if err := initTable(ctx); err != nil {
		return fmt.Errorf("初始化服务注册表失败: %w", err)
	}

	svc := readService(service)
	if svc == nil {
		return errors.New("服务未配置")
	}

	var addr string
	if opt.Address != "" {
		addr = opt.Address
	} else if addr, err = readServiceAddress(svc); err != nil {
		return fmt.Errorf("读取服务地址失败: %w", err)
	}

	svcConfig := internal.config.GetService(svc.Name)
	trpcNode := &registry.Node{
		ServiceName: service,
		Address:     addr,
		Network:     svc.Network,
		Protocol:    svc.Protocol,
		Weight:      svcConfig.GetWeight(),
	}
	if opt.Address != "" {
		trpcNode.Address = opt.Address
	}

	proxy := mysql.NewClientProxy(conf.GetMySQLName())
	statement := fmt.Sprintf(
		"INSERT INTO `%s` (`name`, `host`, `weight`, `json_desc`, `update_time_msec`)"+
			"VALUES (?, ?, ?, ?, ?)",
		dbItem{}.TableName(),
	)
	b, _ := json.Marshal(trpcNode)
	res, err := proxy.Exec(
		ctx, statement,
		service, addr, svcConfig.GetWeight(), string(b), time.Now().UnixMilli(),
	)
	if err != nil {
		return fmt.Errorf("注册服务失败: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("获取服务注册 ID 失败: %w", err)
	}

	now := time.Now().Local()
	node := &node{
		id:             id,
		node:           trpcNode,
		lastUpdateTime: &now,
	}
	r.nodes.Store(service, node)
	log.Infof("已注册服务 %s, 节点信息: %v", service, toJSON{trpcNode})

	return nil
}

func mergeRegistryOpts(opts []registry.Option) *registry.Options {
	opt := &registry.Options{}
	for _, o := range opts {
		o(opt)
	}
	return opt
}

// 实现 registry.Registry 接口
func (r *registerImpl) Deregister(service string) error {
	node, exist := r.nodes.LoadAndDelete(service)
	if !exist {
		return fmt.Errorf("服务 %s 未注册", service)
	}

	statement := fmt.Sprintf(
		"UPDATE `%s` SET `deregister_time_msec` = ? WHERE `id` = ?",
		dbItem{}.TableName(),
	)
	proxy := mysql.NewClientProxy(internal.config.GetMySQLName())
	_, err := proxy.Exec(
		context.Background(), statement,
		time.Now().UnixMilli(), node.id,
	)
	if err != nil {
		return fmt.Errorf("注销服务 %s 失败: %w", service, err)
	}

	log.Infof("已注销服务 %s", service)
	return nil
}

func (r *registerImpl) doUpdate() {
	defer recovery.CatchPanic(recovery.WithCallback(func(info any, stack []caller.Caller) {
		log.Errorf(
			"registry %s 异常退出, 错误信息: '%v', 堆栈: %v",
			SelectorName, info, toJSON{stack},
		)
		go r.doUpdate()
	}))

	for {
		ctx := trace.WithTraceID(context.Background(), fmt.Sprintf(
			"registry-%s-%s", SelectorName, time.Now().Local().Format("060102150405"),
		))
		statement := fmt.Sprintf(
			"UPDATE `%s` SET `update_time_msec` = ? WHERE `id` = ?",
			dbItem{}.TableName(),
		)
		proxy := mysql.NewClientProxy(internal.config.GetMySQLName())

		r.nodes.Range(func(service string, node *node) (next bool) {
			next = true
			res, err := proxy.Exec(ctx, statement, time.Now().UnixMilli(), node.id)
			if err != nil {
				log.ErrorContextf(ctx, "维持服务心跳失败, 服务 %s, 错误: %v", service, err)
				return
			}
			if rows, _ := res.RowsAffected(); rows == 0 {
				log.ErrorContextf(ctx, "更新服务 %s 不生效, 注册条目可能已被删除", service)
				return
			}
			log.TraceContextf(ctx, "维持服务心跳, 服务 %s", service)
			return
		})

		time.Sleep(heartbeatInterval)
	}
}

func readServiceAddress(svc *trpc.ServiceConfig) (string, error) {
	if svc.Address != "" {
		return svc.Address, nil
	}

	ip, err := readExposeIP()
	if err != nil {
		return "", err
	}
	port, err := readExposePort(svc)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:%d", ip, port), nil
}

func readExposeIP() (string, error) {
	// TODO: 支持多种其他环境变量覆盖

	// reference: [go 获取本地ip地址](https://cloud.tencent.com/developer/article/1469160)
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	for _, addr := range addrs {
		// 检查ip地址判断是否回环地址
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}

	return "", errors.New("无法获得对外暴露的 IP 信息")
}

func readExposePort(svc *trpc.ServiceConfig) (int, error) {
	// TODO: 支持多种其他环境变量覆盖

	return int(svc.Port), nil
}

func readService(name string) *trpc.ServiceConfig {
	services := trpc.GlobalConfig().Server.Service
	for _, svc := range services {
		if svc.Name == name {
			return svc
		}
	}
	return nil
}
