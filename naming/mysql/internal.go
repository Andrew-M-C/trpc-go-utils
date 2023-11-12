package mysql

import (
	"github.com/Andrew-M-C/go.util/maps"
	"trpc.group/trpc-go/trpc-database/mysql"
)

var internal = struct {
	config       *mysqlConfig
	clients      maps.RWSafeMap[string, []*node]
	tableCreated bool
	register *registerImpl
}{
	config:  &mysqlConfig{},
	clients: maps.NewRWSafeMap[string, []*node](),
}

func getMySQLClientProxy() mysql.Client {
	return mysql.NewClientProxy(internal.config.GetMySQLName())
}
