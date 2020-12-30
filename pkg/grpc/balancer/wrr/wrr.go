package wrr

import (
	"sync"

	"github.com/openzipkin/zipkin-go"
	tBalancer "github.com/tencentyun/tsf-go/pkg/balancer"
	"github.com/tencentyun/tsf-go/pkg/balancer/random"
	"github.com/tencentyun/tsf-go/pkg/log"
	"github.com/tencentyun/tsf-go/pkg/meta"
	"github.com/tencentyun/tsf-go/pkg/naming"
	"github.com/tencentyun/tsf-go/pkg/route"
	"go.uber.org/zap"

	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
)

const Name = "wrr"

var (
	_ base.PickerBuilder = &Builder{}
	_ balancer.Picker    = &Picker{}

	mu sync.Mutex
)

// Register register balancer builder if nil.
func Register(router route.Router) {
	mu.Lock()
	defer mu.Unlock()
	if balancer.Get(Name) == nil {
		balancer.Register(newBuilder(router))
	}
}

// Set overwrite any balancer builder.
func Set(router route.Router) {
	mu.Lock()
	defer mu.Unlock()
	if balancer.Get(Name) == nil {
		balancer.Register(newBuilder(router))
	}
}

type Builder struct {
	router route.Router
}

// newBuilder creates a new weighted-roundrobin balancer builder.
func newBuilder(router route.Router) balancer.Builder {
	return base.NewBalancerBuilder(
		Name,
		&Builder{router: router},
		base.Config{HealthCheck: true},
	)
}

func (b *Builder) Build(info base.PickerBuildInfo) balancer.Picker {
	p := &Picker{
		subConns: make(map[string]balancer.SubConn),
		r:        b.router,
		b:        &random.Picker{},
	}
	for conn, info := range info.ReadySCs {
		ins := info.Address.Attributes.Value("rawInstance").(naming.Instance)
		p.instances = append(p.instances, ins)
		p.subConns[ins.Addr()] = conn
	}
	return p
}

type Picker struct {
	instances []naming.Instance
	subConns  map[string]balancer.SubConn
	r         route.Router //路由&泳道
	b         tBalancer.Balancer
}

// Pick pick instances
func (p *Picker) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	svc := naming.NewService(
		meta.Sys(info.Ctx, meta.DestKey(meta.ServiceNamespace)).(string),
		meta.Sys(info.Ctx, meta.DestKey(meta.ServiceName)).(string),
	)
	log.Debug(info.Ctx, "wrr pick", zap.Any("svc", svc), zap.Any("nodes", p.instances))

	nodes := p.r.Select(info.Ctx, svc, p.instances)
	if len(nodes) == 0 {
		log.Error(info.Ctx, "grpc picker: ErrNoSubConnAvailable!")
		return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
	}
	node, _ := p.b.Pick(info.Ctx, nodes)
	span := zipkin.SpanFromContext(info.Ctx)
	if span != nil {
		ep, _ := zipkin.NewEndpoint(node.Service.Name, node.Addr())
		span.SetRemoteEndpoint(ep)
	}
	return balancer.PickResult{
		SubConn: p.subConns[node.Addr()],
		Done:    nil,
	}, nil
}
