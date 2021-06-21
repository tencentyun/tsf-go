package util

import (
	"net/url"
	"strconv"
	"strings"
)

// ParseTarget parse targe
func ParseTarget(endpoint string) (string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		if u, err = url.Parse("http://" + endpoint); err != nil {
			return "", err
		}
	}
	var service string
	if len(u.Path) > 1 {
		service = u.Path[1:]
	}
	return service, nil
}

func ParseAddr(addr string) (ip string, port uint16) {
	strs := strings.Split(addr, ":")
	if len(strs) > 0 {
		ip = strs[0]
	}
	if len(strs) > 1 {
		uport, _ := strconv.ParseUint(strs[1], 10, 16)
		port = uint16(uport)
	}
	return
}
