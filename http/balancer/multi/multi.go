package multi

import (
	"context"
	"fmt"

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
}

func New(router route.Router, b tBalancer.Balancer) *Balancer {
	return &Balancer{
		r: router, b: b,
	}
}
func (b *Balancer) Pick(ctx context.Context, pathPattern string, nodes []*registry.ServiceInstance) (node *registry.ServiceInstance, done func(context.Context, balancer.DoneInfo), err error) {
	var inss []naming.Instance
	for _, node := range nodes {
		inss = append(inss, *naming.FromKratosInstance(node)[0])
	}
	svc := naming.NewService(meta.Sys(ctx, meta.DestKey(meta.ServiceName)).(string), meta.Sys(ctx, meta.DestKey(meta.ServiceNamespace)).(string))
	log.DefaultLog.Debugw("msg", "picker pick", "svc", svc, "nodes", inss)
	filters := b.r.Select(ctx, *svc, inss)
	if len(nodes) == 0 {
		log.DefaultLog.Errorw("msg", "picker: ErrNoSubConnAvailable!", "service", svc.Name)
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
