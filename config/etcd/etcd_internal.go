package etcd

import "trpc.group/trpc-go/trpc-go/metrics"

const logPrefix = "[amc.util.config.etcd]"

func count(name string) {
	metrics.IncrCounter("amc.utils.config.etcd."+name, 1)
}

// E 表示内部错误
type E string

func (e E) Error() string {
	return string(e)
}
