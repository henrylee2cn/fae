package engine

import (
	"github.com/funkygao/golib/null"
	"github.com/funkygao/thrift/lib/go/thrift"
)

type rpcClientHandler func(sock thrift.TTransport)

// Like php-fpm pm pool
// forking goroutine under benchmark is around 40k/s, if higher conn/s
// is required, need pre-fork goroutines
type rpcDispatcher struct {
	preforkMode bool
	handler     rpcClientHandler

	throttleChan     chan null.NullStruct   // if not prefork mode
	clientSocketChan chan thrift.TTransport // if prefork mode
}

func newRpcDispatcher(prefork bool, maxOutstandingSessions int,
	handler rpcClientHandler) (this *rpcDispatcher) {
	this = new(rpcDispatcher)
	this.handler = handler
	this.preforkMode = prefork

	if !this.preforkMode {
		this.throttleChan = make(chan null.NullStruct, maxOutstandingSessions)
		return
	}

	this.clientSocketChan = make(chan thrift.TTransport, maxOutstandingSessions)
	for i := 0; i < maxOutstandingSessions; i++ {
		// prefork
		go func() {
			for {
				this.handler(<-this.clientSocketChan)
			}
		}()
	}

	return
}

func (this *rpcDispatcher) Dispatch(clientSocket thrift.TTransport) {
	if this.preforkMode {
		this.clientSocketChan <- clientSocket // block if busy
		return
	}

	this.throttleChan <- null.Null // block if outstanding sessions overflows
	go func() {
		this.handler(clientSocket)
		<-this.throttleChan
	}()
}

func (this *rpcDispatcher) Runtime() map[string]interface{} {
	r := make(map[string]interface{})
	if this.preforkMode {
		r["cap"] = cap(this.clientSocketChan)
		r["pending"] = len(this.clientSocketChan)
	} else {
		r["cap"] = cap(this.throttleChan)
		r["pending"] = len(this.throttleChan)
	}
	return r
}
