package util

import (
	"net"
	"strings"
)

func IPFromAddr(addr net.Addr) (ip string) {
	if len(addr.String()) == 0 {
		return
	}
	addrs := strings.SplitN(addr.String(), ":", 2)
	if len(addrs) == 0 {
		return
	}
	ip = addrs[0]
	return
}
