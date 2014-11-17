/*
local cache key:string, value:[]byte.
*/
package servant

import (
	"git.apache.org/thrift.git/lib/go/thrift"
	"github.com/funkygao/fae/servant/gen-go/fun/rpc"
	"github.com/funkygao/golib/cache"
	log "github.com/funkygao/log4go"
)

func (this *FunServantImpl) onLcLruEvicted(key cache.Key, value interface{}) {
	// Can't use LruCache public api
	// Because that will lead to nested LruCache RWMutex lock, dead lock
	log.Debug("lru[%v] evicted", key)
}

func (this *FunServantImpl) LcSet(ctx *rpc.Context,
	key string, value []byte) (r bool, appErr error) {
	this.stats.inc("lc.set")

	profiler, err := this.getSession(ctx).startProfiler()
	if err != nil {
		appErr = err
		return
	}

	this.lc.Set(key, value)
	r = true
	profiler.do("lc.set", ctx,
		"{key^%s val^%s} {r^%v}",
		key, value, r)

	return
}

func (this *FunServantImpl) LcGet(ctx *rpc.Context, key string) (r []byte,
	miss *rpc.TCacheMissed, appErr error) {
	this.stats.inc("lc.get")

	profiler, err := this.getSession(ctx).startProfiler()
	if err != nil {
		appErr = err
		return
	}

	result, ok := this.lc.Get(key)
	if !ok {
		miss = rpc.NewTCacheMissed()
		miss.Message = thrift.StringPtr("lcache missed: " + key) // optional
	} else {
		r = result.([]byte)
	}
	profiler.do("lc.get", ctx,
		"{key^%s} {miss^%v r^%s}",
		key, miss, r)

	return
}

func (this *FunServantImpl) LcDel(ctx *rpc.Context, key string) (appErr error) {
	this.stats.inc("lc.del")

	profiler, err := this.getSession(ctx).startProfiler()
	if err != nil {
		appErr = err
		return
	}

	this.lc.Del(key)
	profiler.do("lc.del", ctx,
		"{key^%s}", key)
	return
}
