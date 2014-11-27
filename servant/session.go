package servant

import (
	"github.com/funkygao/fae/config"
	"github.com/funkygao/fae/servant/gen-go/fun/rpc"
	"github.com/funkygao/golib/sampling"
	log "github.com/funkygao/log4go"
	"sync/atomic"
	"time"
)

type session struct {
	profiler *profiler
	ctx      *rpc.Context // will stay the same during a session
}

func (this *FunServantImpl) getSession(ctx *rpc.Context) *session {
	s, present := this.sessions.Get(ctx.Rid)
	if !present {
		atomic.AddInt64(&this.sessionN, 1)
		s = &session{ctx: ctx}
		this.sessions.Set(ctx.Rid, s)

		var uid int64 = 0
		if ctx.IsSetUid() {
			uid = *ctx.Uid
		}
		log.Trace("new session {uid^%d rid^%s reason^%s}", uid, ctx.Rid, ctx.Reason)
	}

	return s.(*session)
}

func (this *session) startProfiler() (*profiler, error) {
	if this.profiler == nil {
		if this.ctx.Rid == "" || this.ctx.Reason == "" {
			log.Error("Invalid context: %s", this.ctx.String())
			return nil, ErrInvalidContext
		}

		this.profiler = &profiler{}
		// TODO 某些web server需要100%采样
		this.profiler.on = sampling.SampleRateSatisfied(config.Servants.ProfilerRate) // rand(1000) <= ProfilerRate
		this.profiler.t0 = time.Now()
		this.profiler.t1 = this.profiler.t0
	}

	this.profiler.t1 = time.Now()
	return this.profiler, nil
}
