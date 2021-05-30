package composite

import (
	"context"
	"sync"

	"github.com/tencentyun/tsf-go/naming"
	"github.com/tencentyun/tsf-go/route"
	"github.com/tencentyun/tsf-go/route/router"

	"github.com/tencentyun/tsf-go/route/lane"
)

var (
	_ route.Router = &Composite{}

	mu               sync.Mutex
	defaultComposite *Composite
)

type Composite struct {
	route route.Router
	lane  *lane.Lane
}

func DefaultComposite() *Composite {
	mu.Lock()
	defer mu.Unlock()
	if defaultComposite == nil {
		defaultComposite = New(router.DefaultRouter(), lane.DefaultLane())
	}
	return defaultComposite
}

func New(router *router.Router, lane *lane.Lane) *Composite {
	return &Composite{route: router, lane: lane}
}

func (c *Composite) Select(ctx context.Context, svc naming.Service, nodes []naming.Instance) []naming.Instance {
	res := c.lane.Select(ctx, svc, nodes)
	if len(res) == 0 {
		return res
	}
	return c.route.Select(ctx, svc, res)
}

func (c *Composite) Lane() *lane.Lane {
	return c.lane
}
