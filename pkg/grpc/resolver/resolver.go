package resolver

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/tencentyun/tsf-go/log"
	"github.com/tencentyun/tsf-go/pkg/naming"
	"github.com/tencentyun/tsf-go/pkg/sys/env"
	"google.golang.org/grpc/attributes"
	"google.golang.org/grpc/resolver"
)

var (
	_ resolver.Builder  = &Builder{}
	_ resolver.Resolver = &Resolver{}

	mu sync.Mutex
)

// Register register resolver builder if nil.
func Register(d naming.Discovery) {
	mu.Lock()
	defer mu.Unlock()
	if resolver.Get(d.Scheme()) == nil {
		resolver.Register(&Builder{d})
	}
}

// Set overwrite any registered builder
func Set(b naming.Discovery) {
	mu.Lock()
	defer mu.Unlock()
	resolver.Register(&Builder{b})
}

// builder is also a resolver builder.
// It's build() function always returns itself.
type Builder struct {
	naming.Discovery
}

// Build returns itself for Resolver, because it's both a builder and a resolver.
// consul://local/provider-demo
func (b *Builder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	str := strings.SplitN(target.Endpoint, "?", 2)
	if len(str) == 0 {
		return nil, fmt.Errorf("[resolver] parse target.Endpoint(%s) failed!err:=endpoint is empty", target.Endpoint)
	}
	nid := target.Authority
	if target.Authority == "local" {
		nid = env.NamespaceID()
	}
	svc := naming.NewService(nid, str[0])
	log.DefaultLog.Debugw("msg", "[grpc resovler] start subscribe service", "svc", svc.Name)
	r := &Resolver{
		watcher: b.Subscribe(svc),
		cc:      cc,
		svc:     svc,
	}
	go r.updateproc()
	return r, nil
}

// Resolver watches for the updates on the specified target.
// Updates include address updates and service config updates.
type Resolver struct {
	svc     naming.Service
	watcher naming.Watcher
	cc      resolver.ClientConn
}

// Close is a noop for Resolver.
func (r *Resolver) Close() {
	log.DefaultLog.Infow("msg", "[grpc resovler] close subscribe service", "serviceName", r.svc.Name, "namespace", r.svc.Namespace)
	r.watcher.Close()
}

// ResolveNow is a noop for Resolver.
func (r *Resolver) ResolveNow(o resolver.ResolveNowOptions) {
}

func (r *Resolver) updateproc() {
	ctx := context.Background()
	for {
		nodes, err := r.watcher.Watch(ctx)
		if errors.IsClientClosed(err) {
			return
		}
		if len(nodes) > 0 {
			r.newAddress(nodes)
		}
	}
}
func (r *Resolver) newAddress(instances []naming.Instance) {
	if len(instances) <= 0 {
		return
	}
	addrs := make([]resolver.Address, 0, len(instances))
	for _, ins := range instances {
		addr := resolver.Address{
			Addr:       ins.Addr(),
			ServerName: ins.Service.Name,
		}
		addr.Attributes = attributes.New("rawInstance", ins)
		addrs = append(addrs, addr)
	}
	log.DefaultLog.Info("msg", "[resolver] newAddress found!", "length", len(addrs), "serviceName", r.svc.Name, "namespace", r.svc.Namespace)
	r.cc.NewAddress(addrs)
}
