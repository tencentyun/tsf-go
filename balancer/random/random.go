package random

import (
	"context"
	"math/rand"

	"github.com/tencentyun/tsf-go/balancer"
	"github.com/tencentyun/tsf-go/naming"
)

var (
	_ balancer.Balancer = &Picker{}

	Name = "tsf-random"
)

type Picker struct {
}

func (p *Picker) Pick(ctx context.Context, nodes []naming.Instance) (node *naming.Instance, done func(balancer.DoneInfo)) {
	if len(nodes) == 0 {
		return nil, func(balancer.DoneInfo) {}
	}
	cur := rand.Intn(len(nodes))
	return &nodes[cur], func(balancer.DoneInfo) {}
}

func (p *Picker) Schema() string {
	return Name
}
