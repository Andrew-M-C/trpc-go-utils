package mysql

import (
	"encoding/json"
	"fmt"

	"github.com/Andrew-M-C/go.util/maps"
	"trpc.group/trpc-go/trpc-database/mysql"
)

var internal = struct {
	config       *mysqlConfig
	clients      maps.RWSafeMap[string, []*node]
	tableCreated bool
	register     *registerImpl
}{
	config:  &mysqlConfig{},
	clients: maps.NewRWSafeMap[string, []*node](),
}

func getMySQLClientProxy() mysql.Client {
	return mysql.NewClientProxy(internal.config.GetMySQLName())
}

type toJSON struct {
	v any
}

func (j toJSON) String() string {
	b, err := json.Marshal(j.v)
	if err != nil {
		return fmt.Sprint(j.v)
	}
	return string(b)
}
