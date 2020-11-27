package util

import (
	"flag"
	"sync"
)

var mu sync.Mutex

func ParseFlag() {
	mu.Lock()
	defer mu.Unlock()
	if !flag.Parsed() {
		flag.Parse()
	}
}
