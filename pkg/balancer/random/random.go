package random

import (
	"context"
	"math/rand"

	"github.com/tencentyun/tsf-go/pkg/balancer"
	"github.com/tencentyun/tsf-go/pkg/naming"
)

var (
	_ balancer.Balancer = &Picker{}
)

type Picker struct {
}

func (p *Picker) Pick(ctx context.Context, nodes []naming.Instance) (node *naming.Instance) {
	if len(nodes) == 0 {
		return nil
	}
	cur := rand.Intn(len(nodes))
	return &nodes[cur]
}
