package mysql

import (
	"github.com/funkygao/fae/config"
)

type ClientPool struct {
	selector ServerSelector
	conf     *config.ConfigMysql
	clients  map[string]*mysql
}

func New(cf *config.ConfigMysql) *ClientPool {
	this := new(ClientPool)
	this.conf = cf
	switch cf.ShardStrategy {
	default:
		this.selector = newStandardServerSelector()
	}
	this.selector.SetServers(cf)
	this.clients = make(map[string]*mysql)
	for _, pool := range cf.Pools() {
		this.clients[pool] = newMysql(cf.Servers[pool].DSN())
	}
	return this
}

func (this *ClientPool) Query(pool string, table string, shardId int32,
	sql string, args []interface{}) (r [][]byte, err error) {
	return
}