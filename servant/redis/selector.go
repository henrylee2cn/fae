package redis

import (
	"net"
)

type ServerSelector interface {
	SetServers(servers ...string) error
	PickServer(key string) (addr string)
}
