package main

import (
	"fmt"
	"github.com/funkygao/fae/servant/gen-go/fun/rpc"
	"github.com/funkygao/fae/servant/proxy"
	"labix.org/v2/mgo/bson"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"
)

func recordIoError(err error) {
	const (
		RESET_BY_PEER = "connection reset by peer"
		BROKEN_PIPE   = "broken pipe"
		IO_TIMEOUT    = "i/o timeout"
	)
	if strings.HasSuffix(err.Error(), RESET_BY_PEER) ||
		strings.HasSuffix(err.Error(), IO_TIMEOUT) ||
		strings.HasSuffix(err.Error(), BROKEN_PIPE) {
		report.incIoErrs()
	}
}

func runSession(proxy *proxy.Proxy, wg *sync.WaitGroup, round int, seq int) {
	report.updateConcurrency(1)
	report.incSessions()
	defer func() {
		wg.Done()
		report.updateConcurrency(-1)
		//log.Printf("session{round^%d seq^%d} done", round, seq)
	}()

	t1 := time.Now()
	client, err := proxy.ServantByAddr(host + ":9001")
	if err != nil {
		report.incConnErrs()
		log.Printf("session{round^%d seq^%d} error: %v", round, seq, err)
		return
	}
	defer client.Recycle() // when err occurs, do we still need recycle?

	var enableLog = false
	if !logTurnOff && sampling(SampleRate) {
		enableLog = true
	}

	if enableLog {
		log.Printf("session{round^%d seq^%d} connected within %s",
			round, seq, time.Since(t1))
	}

	ctx := rpc.NewContext()
	ctx.Reason = "stress.go"
	for i := 0; i < LoopsPerSession; i++ {
		//time.Sleep(time.Millisecond * 5)
		ctx.Rid = time.Now().UnixNano()

		if Cmd&CallNoop != 0 {
			var r int32
			r, err = client.Noop(1)
			if err != nil {
				recordIoError(err)
				report.incCallErr()
				if !errOff {
					log.Printf("session{round^%d seq^%d noop}: %v", round, seq, err)
				}
				client.Close()
				return
			} else {
				report.incCallOk()
				if enableLog {
					log.Printf("session{round^%d seq^%d noop}: %v", round, seq, r)
				}
			}
		}

		if Cmd&CallPing != 0 {
			var r string
			r, err = client.Ping(ctx)
			if err != nil {
				recordIoError(err)
				report.incCallErr()
				if !errOff {
					log.Printf("session{round^%d seq^%d ping}: %v", round, seq, err)
				}
				client.Close()
				return
			} else {
				report.incCallOk()
				if enableLog {
					log.Printf("session{round^%d seq^%d ping}: %v", round, seq, r)
				}
			}
		}

		if Cmd&CallLCache != 0 {
			key := fmt.Sprintf("lc_stress:%d", rand.Int())
			value := []byte("value of " + key)
			_, err = client.LcSet(ctx, key, value)
			if err != nil {
				recordIoError(err)
				report.incCallErr()
				if !errOff {
					log.Printf("session{round^%d seq^%d lc.set}: %v", round, seq, err)
				}
				client.Close()
				return
			} else {
				report.incCallOk()
			}

			value, _, err = client.LcGet(ctx, key)
			if err != nil {
				recordIoError(err)
				report.incCallErr()
				if !errOff {
					log.Printf("session{round^%d seq^%d lc.get}: %v", round, seq, err)
				}
				client.Close()
				return
			} else {
				report.incCallOk()
				if enableLog {
					log.Printf("session{round^%d seq^%d lcache}: %s => %s",
						round, seq, key, string(value))
				}
			}
		}

		if Cmd&CallIdGen != 0 {
			var r int64
			r, err = client.IdNext(ctx)
			if err != nil {
				recordIoError(err)
				report.incCallErr()
				if !errOff {
					log.Printf("session{round^%d seq^%d idgen}: %v", round, seq, err)
				}
				client.Close()
				return
			} else {
				if enableLog {
					log.Printf("session{round^%d seq^%d idgen}: %d",
						round, seq, r)
				}
				report.incCallOk()
			}
		}

		if Cmd&CallMysql != 0 {
			// with cache
			if true {
				r, err := client.MyQuery(ctx, "UserShard", "UserInfo", 1,
					"SELECT * FROM UserInfo WHERE uid=?",
					[]string{"1"}, "user:1")
				if err != nil {
					recordIoError(err)
					report.incCallErr()
					if !errOff {
						log.Printf("session{round^%d seq^%d mysql.cache}: %v", round, seq, err)
					}
					client.Close()
					return
				} else {
					if enableLog {
						log.Printf("session{round^%d seq^%d mysql}: %+v",
							round, seq, r)
					}
					report.incCallOk()
				}
			}

			// without cache
			if true {
				var rows *rpc.MysqlResult
				rows, err = client.MyQuery(ctx, "UserShard", "UserInfo", 1,
					"SELECT * FROM UserInfo WHERE uid=?",
					[]string{"1"}, "")
				if err != nil {
					recordIoError(err)
					report.incCallErr()
					if !errOff {
						log.Printf("session{round^%d seq^%d mysql.nocache}: %v", round, seq, err)
					}
					client.Close()
					return
				} else {
					if enableLog {
						log.Printf("session{round^%d seq^%d mysql}: %+v",
							round, seq, rows.Rows)
					}
					report.incCallOk()
				}
			}

		}

		continue // TODO

		var (
			key     string
			value   []byte
			mcValue = rpc.NewTMemcacheData()
			result  []byte
		)
		key = fmt.Sprintf("mc_stress:%d", rand.Int())
		value = []byte("value of " + key)
		mcValue.Data = value

		if Cmd&CallMemcache != 0 {
			_, err = client.McSet(ctx, MC_POOL, key, mcValue, 36000)
			if err != nil {
				report.incCallErr()
				log.Printf("session{round^%d seq^%d mc_set} %v", round, seq, err)
			} else {
				report.incCallOk()
			}
			_, miss, err := client.McGet(ctx, MC_POOL, key)
			if miss != nil || err != nil {
				report.incCallErr()
				log.Printf("session{round^%d seq^%d mc_get} miss^%v, err^%v",
					round, seq, miss, err)
			} else {
				report.incCallOk()
			}
		}

		if Cmd&CallMongo != 0 {
			mgQuery, _ := bson.Marshal(bson.M{"snsid": "100003391571259"})
			mgFields, _ := bson.Marshal(bson.M{})
			result, _, err = client.MgFindOne(ctx, "default", "idmap",
				0, mgQuery, mgFields)
			if err != nil {
				report.incCallErr()
				log.Printf("session{round^%d seq^%d mg_findOne} %v", round, seq, err)
			} else {
				report.incCallOk()

				if false {
					log.Println(result)
				}
			}
		}

	}

	if enableLog {
		log.Printf("session{round^%d seq^%d} done", round, seq)
	}

}
