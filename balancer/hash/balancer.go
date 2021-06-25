package hash

import (
	"context"
	"math/rand"
	"sync"

	"github.com/tencentyun/tsf-go/balancer"
	"github.com/tencentyun/tsf-go/log"
	"github.com/tencentyun/tsf-go/naming"
)

type hashKey struct{}

// NewContext creates a new context with hashkey.
func NewContext(ctx context.Context, key string) context.Context {
	return context.WithValue(ctx, hashKey{}, key)
}

// FromContext returns the hashkey in ctx if it exists.
func FromContext(ctx context.Context) (string, bool) {
	key, ok := ctx.Value(hashKey{}).(string)
	return key, ok
}

var (
	_ balancer.Balancer = &Picker{}

	Name = "consistent_hash"
)

type Picker struct {
	c  *Consistent
	lk sync.Mutex
}

func New() *Picker {
	return &Picker{
		c: NewHash(),
	}
}

func (p *Picker) Pick(ctx context.Context, nodes []naming.Instance) (node *naming.Instance, done func(balancer.DoneInfo)) {
	if len(nodes) == 0 {
		return nil, func(balancer.DoneInfo) {}
	}

	key, ok := FromContext(ctx)
	if !ok {
		cur := rand.Intn(len(nodes))
		return &nodes[cur], func(balancer.DoneInfo) {}
	}
	p.lk.Lock()
	defer p.lk.Unlock()
	var inss []Node
	for i, node := range nodes {
		inss = append(inss, Node{name: node.Addr(), idx: i})
	}
	p.c.Set(inss)
	addr, err := p.c.Get(key)
	if err != nil {
		log.DefaultLog.Errorf("hash picker: get failed! %s", err)
		cur := rand.Intn(len(nodes))
		return &nodes[cur], func(balancer.DoneInfo) {}
	}
	return &nodes[p.c.Index(addr)], func(balancer.DoneInfo) {}
}

func (p *Picker) Schema() string {
	return Name
}
