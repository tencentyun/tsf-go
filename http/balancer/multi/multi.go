package multi

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-kratos/kratos/v2/registry"
	"github.com/go-kratos/kratos/v2/transport/http/balancer"
	"github.com/openzipkin/zipkin-go"
	tBalancer "github.com/tencentyun/tsf-go/balancer"
	"github.com/tencentyun/tsf-go/log"
	"github.com/tencentyun/tsf-go/naming"
	"github.com/tencentyun/tsf-go/pkg/meta"
	"github.com/tencentyun/tsf-go/route"
)

type Balancer struct {
	r route.Router //路由&泳道
	b tBalancer.Balancer

	lock  sync.RWMutex
	nodes []naming.Instance
}

func New(router route.Router, b tBalancer.Balancer) *Balancer {
	return &Balancer{
		r: router, b: b,
	}
}

func (b *Balancer) Pick(ctx context.Context) (node *registry.ServiceInstance, done func(context.Context, balancer.DoneInfo), err error) {
	b.lock.RLock()
	nodes := b.nodes
	b.lock.RUnlock()
	svc := naming.NewService(meta.Sys(ctx, meta.DestKey(meta.ServiceNamespace)).(string), meta.Sys(ctx, meta.DestKey(meta.ServiceName)).(string))
	if len(nodes) == 0 {
		log.DefaultLog.Errorf("picker: ErrNoSubConnAvailable! %s", svc.Name)
		return nil, nil, fmt.Errorf("no instances avaiable")
	}
	log.DefaultLog.Debugw("msg", "picker pick", "svc", svc, "nodes", nodes)
	filters := b.r.Select(ctx, *svc, nodes)
	if len(filters) == 0 {
		log.DefaultLog.Errorf("picker: ErrNoSubConnAvailable after route filter!  %s", svc.Name)
		return nil, nil, fmt.Errorf("no instances avaiable")
	}
	ins, _ := b.b.Pick(ctx, filters)
	span := zipkin.SpanFromContext(ctx)
	if span != nil {
		ep, _ := zipkin.NewEndpoint(ins.Service.Name, ins.Addr())
		span.SetRemoteEndpoint(ep)
	}
	return ins.ToKratosInstance(), func(context.Context, balancer.DoneInfo) {}, nil
}

func (b *Balancer) Update(nodes []*registry.ServiceInstance) {
	b.lock.Lock()
	defer b.lock.Unlock()
	var inss []naming.Instance
	for _, node := range nodes {
		inss = append(inss, *naming.FromKratosInstance(node)[0])
	}
	b.nodes = inss
}
