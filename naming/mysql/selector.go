package mysql

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/Andrew-M-C/trpc-go-utils/tracelog"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/naming/registry"
	"trpc.group/trpc-go/trpc-go/naming/selector"
)

func initSelector() {
	selector.Register(SelectorName, myqlSelector{})
}

type myqlSelector struct{}

func (myqlSelector) Select(name string, opts ...selector.Option) (_ *registry.Node, err error) {
	opt := mergeSelectorOptions(opts)
	ctx := opt.Ctx

	defer func() {
		if err != nil {
			log.ErrorContextf(ctx, "选择器失败: %v", err)
		}
	}()

	// TODO: 支持各种轮询选项
	log.DebugContextf(ctx, "请求 selector name '%s', 选项: %v", name, tracelog.ToJSON(opt))

	// 首先尝试从缓存中查找节点
	if n := getOneNodeFromCacheAndCheckTimeout(ctx, name, opt); n != nil {
		return n.GetNode(), nil
	}

	// 如果需要更新缓存, 则从 DB 中获取数据
	aliveTime := time.Now().Add(-liveTimeInterval)
	statement := fmt.Sprintf(
		"SELECT * FROM `%s` WHERE `deregister_time_msec` = 0 AND `update_time_msec` >= ? AND `name` = ?",
		dbItem{}.TableName(),
	)

	var items []dbItem
	proxy := getMySQLClientProxy()
	if err := proxy.Select(ctx, &items, statement, aliveTime.UnixMilli(), name); err != nil {
		n := getOneNodeFromCache(ctx, name, opt)
		if n == nil {
			return nil, fmt.Errorf("查询名字服务节点失败: %w", err)
		}
		log.WarnContextf(ctx, "查询名字服务节点失败: %v, 柔性返回节点 %v", err, n.node.Address)
		return n.GetNode(), nil
	}

	// 没有可用的节点, 也只能用缓存了
	if len(items) == 0 {
		log.DebugContext(ctx, "无法更新名字服务, 尝试本地缓存")
		n := getOneNodeFromCache(ctx, name, opt)
		if n == nil {
			return nil, fmt.Errorf("服务 %s 没有可用的节点", name)
		}
		return n.GetNode(), nil
	}

	// 更新节点
	nodes := make([]*node, 0, len(items))
	for _, item := range items {
		trpcNode := registry.Node{}
		if err := json.Unmarshal([]byte(item.JSONDesc), &trpcNode); err != nil {
			log.ErrorContextf(ctx, "节点数据 %s 不合法", item.JSONDesc)
			continue
		}
		tm := time.UnixMilli(item.UpdateTimeMsec)
		n := &node{
			node:           &trpcNode,
			lastUpdateTime: &tm,
		}
		nodes = append(nodes, n)
	}

	internal.clients.Store(name, nodes)
	idx := rand.Intn(len(nodes))
	return nodes[idx].GetNode(), nil
}

func (myqlSelector) Report(node *registry.Node, cost time.Duration, err error) error {
	log.Infof("要求汇报错误, node %+v, cost %v, err %v", node, cost, err)
	return err
}

// 读取一个节点, 如果发现有过期的, 就返回 nil, 触发更新缓存
func getOneNodeFromCacheAndCheckTimeout(ctx context.Context, name string, opt *selector.Options) *node {
	node := getOneNodeFromCache(ctx, name, opt)
	if node == nil {
		log.DebugContextf(ctx, "客户端 %s 未找到过, 需要请求名字信息", name)
		return nil
	}
	if tm := node.GetLastUpdateTime(); time.Since(tm) > liveTimeInterval {
		log.DebugContextf(ctx, "客户端 %s 更新于 %v, 需要刷新缓存", name, tm)
		return nil
	}
	return node
}

// 从缓存中获取一个节点, 不论是否超时。这是为了处理缓存过期的情况, 柔性逻辑
func getOneNodeFromCache(ctx context.Context, name string, _ *selector.Options) *node {
	nodes, _ := internal.clients.Load(name)
	if len(nodes) == 0 {
		log.DebugContextf(ctx, "客户端 %s 未找到, 需请求名字缓存", name)
		return nil
	}
	// TODO: 支持各种寻址算法
	// 随机获取一个节点
	idx := rand.Intn(len(nodes))
	return nodes[idx]
}

func mergeSelectorOptions(opts []selector.Option) *selector.Options {
	opt := selector.Options{
		Ctx: context.Background(),
	}
	for _, o := range opts {
		o(&opt)
	}
	return &opt
}
