package multi

import (
	"fmt"
	"sync"

	"github.com/go-kratos/kratos/v2/registry"
	"github.com/openzipkin/zipkin-go"
	tBalancer "github.com/tencentyun/tsf-go/balancer"
	"github.com/tencentyun/tsf-go/balancer/random"
	"github.com/tencentyun/tsf-go/naming"
	"github.com/tencentyun/tsf-go/pkg/log"
	"github.com/tencentyun/tsf-go/pkg/meta"
	"github.com/tencentyun/tsf-go/route"
	"go.uber.org/zap"

	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
)

var (
	_ base.PickerBuilder = &Builder{}
	_ balancer.Picker    = &Picker{}

	mu sync.Mutex

	balancers []tBalancer.Balancer
)

func init() {

	// random
	balancers = append(balancers, &random.Picker{})

}

// Register register balancer builder if nil.
func Register(router route.Router) {
	mu.Lock()
	defer mu.Unlock()
	for _, b := range balancers {
		if balancer.Get(b.Schema()) == nil {
			balancer.Register(newBuilder(router, b))
		}
	}

}

// Set overwrite any balancer builder.
func Set(router route.Router) {
	mu.Lock()
	defer mu.Unlock()
	for _, b := range balancers {
		balancer.Register(newBuilder(router, b))
	}
}

type Builder struct {
	router route.Router
	b      tBalancer.Balancer
}

// newBuilder creates a new weighted-roundrobin balancer builder.
func newBuilder(router route.Router, b tBalancer.Balancer) balancer.Builder {
	return base.NewBalancerBuilder(
		b.Schema(),
		&Builder{router: router, b: b},
		base.Config{HealthCheck: true},
	)
}

func (b *Builder) Build(info base.PickerBuildInfo) balancer.Picker {
	p := &Picker{
		subConns: make(map[string]balancer.SubConn),
		r:        b.router,
		b:        b.b,
	}
	for conn, info := range info.ReadySCs {
		metadata := make(map[string]string)
		metadata["protocol"] = info.Address.Attributes.Value("protocol").(string)
		metadata["tsf_status"] = info.Address.Attributes.Value("tsf_status").(string)
		metadata["tsf_tags"] = info.Address.Attributes.Value("tsf_tags").(string)
		metadata["TSF_APPLICATION_ID"] = info.Address.Attributes.Value("TSF_APPLICATION_ID").(string)
		metadata["TSF_GROUP_ID"] = info.Address.Attributes.Value("TSF_GROUP_ID").(string)
		metadata["TSF_INSTNACE_ID"] = info.Address.Attributes.Value("TSF_INSTNACE_ID").(string)
		metadata["TSF_PROG_VERSION"] = info.Address.Attributes.Value("TSF_PROG_VERSION").(string)
		metadata["TSF_ZONE"] = info.Address.Attributes.Value("TSF_ZONE").(string)
		metadata["TSF_REGION"] = info.Address.Attributes.Value("TSF_REGION").(string)
		metadata["TSF_NAMESPACE_ID"] = info.Address.Attributes.Value("TSF_NAMESPACE_ID").(string)
		metadata["TSF_SDK_VERSION"] = info.Address.Attributes.Value("TSF_SDK_VERSION").(string)

		si := &registry.ServiceInstance{
			Name:      info.Address.ServerName,
			Endpoints: []string{fmt.Sprintf("grpc://%s", info.Address.Addr)},
			Metadata:  metadata,
		}

		p.instances = append(p.instances, *naming.FromKratosInstance(si)[0])
		p.subConns[info.Address.Addr] = conn
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
	svc := naming.NewService(meta.Sys(info.Ctx, meta.DestKey(meta.ServiceName)).(string), meta.Sys(info.Ctx, meta.DestKey(meta.ServiceNamespace)).(string))
	log.Debug(info.Ctx, "picker pick", zap.Any("svc", svc), zap.Any("nodes", p.instances))

	nodes := p.r.Select(info.Ctx, *svc, p.instances)
	if len(nodes) == 0 {
		log.Error(info.Ctx, "picker: ErrNoSubConnAvailable!", zap.String("service", svc.Name))
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
