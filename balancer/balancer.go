package balancer

import (
	"context"

	"github.com/tencentyun/tsf-go/naming"
)

// DoneInfo is callback when rpc done
type DoneInfo struct {
	Err     error
	Trailer map[string]string
}

// Balancer is picker
type Balancer interface {
	Pick(ctx context.Context, nodes []naming.Instance) (node *naming.Instance, done func(DoneInfo))
	Schema() string
}

type Printable interface {
	PrintStats()
}
