package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func sampling(rate int) bool {
	return rand.Intn(rate) == 1
}

func showUsage() {
	flag.PrintDefaults()

	fmt.Println()
	fmt.Println("bitwise rpc calls")
	fmt.Println(strings.Repeat("=", 20))
	fmt.Printf("%16s %3d\n", "CallNoop", CallNoop)
	fmt.Printf("%16s %3d\n", "CallPing", CallPing)
	fmt.Printf("%16s %3d\n", "CallIdGen", CallIdGen)
	fmt.Printf("%16s %3d\n", "CallLCache", CallLCache)
	fmt.Printf("%16s %3d\n", "CallMemcache", CallMemcache)
	fmt.Printf("%16s %3d\n", "CallMongo", CallMongo)
	fmt.Printf("%16s %3d\n", "CallMysql", CallMysql)
	fmt.Printf("%16s %3d\n", "CallRedis", CallRedis)
	fmt.Println()
	fmt.Printf("%16s %3d\n", "Ping+Idgen", CallPingIdgen)
	fmt.Printf("%16s %3d\n", "Lcache+Idgen", CallIdgenLcache)
	fmt.Printf("%16s %3d\n", "Default", CallDefault)
}

func pause(prompt string) {
	log.Println(prompt)
	log.Println()
	reader := bufio.NewReader(os.Stdin)
	reader.ReadString('\n')
}
