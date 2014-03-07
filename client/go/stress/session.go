package main

import (
	"fmt"
	"github.com/funkygao/fae/servant/gen-go/fun/rpc"
	"github.com/funkygao/fae/servant/proxy"
	"github.com/funkygao/golib/fixture"
	"labix.org/v2/mgo/bson"
	"log"
	"math/rand"
	"sync"
	"time"
)

func runSession(proxy *proxy.Proxy, wg *sync.WaitGroup, round int, seq int) {
	defer wg.Done()

	if sampling(100) {
		log.Printf("session{round^%3d seq^%6d} started", round, seq)
	}

	report.incSessions()

	t1 := time.Now()
	client, err := proxy.Servant(host + ":9001")
	if err != nil {
		report.incConnErrs()
		log.Printf("session{round^%3d seq^%6d} %v", round, seq, err)
		return
	}
	defer client.Recycle() // when err occurs, do we still need recyle?

	if sampling(1000) {
		log.Printf("session{round^%3d seq^%6d} connected within %s",
			round, seq, time.Since(t1))
	}

	report.modifyConcurrency(1)
	defer func() {
		report.modifyConcurrency(-1)
	}()

	var (
		key     string
		value   []byte
		mcValue = rpc.NewTMemcacheData()
		result  []byte
	)
	mgQuery, _ := bson.Marshal(bson.M{"snsid": "100003391571259"})
	mgFields, _ := bson.Marshal(bson.M{})
	for i := 0; i < LoopsPerSession; i++ {
		if Cmd&CallPing != 0 {
			_, err = client.Ping(ctx)
			if err != nil {
				report.incCallErr()
				log.Printf("session{round^%3d seq^%6d ping} %v", round, seq, err)
			}
		}

		if Cmd&CallIdGen != 0 {
			_, _, err = client.IdNext(ctx, 0)
			if err != nil {
				report.incCallErr()
				log.Printf("session{round^%3d seq^%6d idgen} %v", round, seq, err)
			}
		}

		key = fmt.Sprintf("mc_stress:%d", rand.Int())
		value = []byte("value of " + key)
		mcValue.Data = value

		if Cmd&CallLCache != 0 {
			_, err = client.LcSet(ctx, key, value)
			if err != nil {
				report.incCallErr()
				log.Printf("session{round^%3d seq^%6d lc_set} %v", round, seq, err)
			}
			_, _, err = client.LcGet(ctx, key)
			if err != nil {
				report.incCallErr()
				log.Printf("session{round^%3d seq^%6d lc_get} %v", round, seq, err)
			}
		}

		if Cmd&CallMemcache != 0 {
			_, err = client.McSet(ctx, key, mcValue, 36000)
			if err != nil {
				report.incCallErr()
				log.Printf("session{round^%3d seq^%6d mc_set} %v", round, seq, err)
			}
			_, err, _ = client.McGet(ctx, key)
			if err != nil {
				report.incCallErr()
				log.Printf("session{round^%3d seq^%6d mc_get} %v", round, seq, err)
			}
		}

		if Cmd&CallMongo != 0 {
			result, _, err = client.MgFindOne(ctx, "default", "idmap", 0,
				mgQuery,
				mgFields)
			if err != nil {
				report.incCallErr()
				log.Printf("session{round^%3d seq^%6d mc_get} %v", round, seq, err)
			} else if false {
				log.Println(result)
			}
		}

		if Cmd&CallKvdb != 0 {
			client.KvdbSet(ctx, fixture.RandomByteSlice(30),
				fixture.RandomByteSlice(10<<10))
		}
	}

	if sampling(100) {
		log.Printf("session{round^%3d seq^%6d} finished", round, seq)
	}

}
