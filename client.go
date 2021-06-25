package tsf

import (
	"context"
	"fmt"
	"sync"

	"github.com/tencentyun/tsf-go/balancer"
	"github.com/tencentyun/tsf-go/balancer/p2c"
	"github.com/tencentyun/tsf-go/breaker"
	"github.com/tencentyun/tsf-go/grpc/balancer/multi"
	httpMulti "github.com/tencentyun/tsf-go/http/balancer/multi"
	"github.com/tencentyun/tsf-go/naming/consul"
	"github.com/tencentyun/tsf-go/pkg/meta"
	"github.com/tencentyun/tsf-go/pkg/sys/env"
	"github.com/tencentyun/tsf-go/route/composite"
	"github.com/tencentyun/tsf-go/route/lane"
	"github.com/tencentyun/tsf-go/tracing"
	"github.com/tencentyun/tsf-go/util"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/metadata"
	"github.com/go-kratos/kratos/v2/middleware"
	mmeta "github.com/go-kratos/kratos/v2/middleware/metadata"
	"github.com/go-kratos/kratos/v2/transport"
	tgrpc "github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	"google.golang.org/grpc"
)

type ClientOption func(*clientOpionts)

type clientOpionts struct {
	tracerOpts       []tracing.Option
	breakerCfg       *breaker.Config
	breakerErrorHook func(ctx context.Context, operation string, err error) (success bool)
	m                []middleware.Middleware
	balancer         balancer.Balancer
}

func WithTracerOpts(opts ...tracing.Option) ClientOption {
	return func(o *clientOpionts) {
		o.tracerOpts = opts
	}
}

func WithBreakerConfig(cfg *breaker.Config) ClientOption {
	return func(o *clientOpionts) {
		o.breakerCfg = cfg
	}
}

func WithBreakerErrorHook(h func(ctx context.Context, operation string, err error) (success bool)) ClientOption {
	return func(o *clientOpionts) {
		o.breakerErrorHook = h
	}
}

func WithMiddlewares(m ...middleware.Middleware) ClientOption {
	return func(o *clientOpionts) {
		o.m = append(o.m, m...)
	}
}

func startClientContext(ctx context.Context, remoteServiceName string, l *lane.Lane, operation string) context.Context {
	// 注入远端服务名
	pairs := []meta.SysPair{
		{Key: meta.DestKey(meta.ServiceName), Value: remoteServiceName},
		{Key: meta.DestKey(meta.ServiceNamespace), Value: env.NamespaceID()},
	}
	// 注入自己的服务名
	k, _ := kratos.FromContext(ctx)
	serviceName := k.Name()
	if res := meta.Sys(ctx, meta.ServiceName); res == nil {
		pairs = append(pairs, meta.SysPair{Key: meta.ServiceName, Value: serviceName})
	} else {
		serviceName = res.(string)
	}

	pairs = append(pairs, meta.SysPair{Key: meta.DestKey(meta.Interface), Value: operation})
	if laneID := l.GetLaneID(ctx); laneID != "" {
		pairs = append(pairs, meta.SysPair{Key: meta.LaneID, Value: laneID})
	}
	ctx = meta.WithSys(ctx, pairs...)

	md := metadata.Metadata{}
	meta.RangeUser(ctx, func(key string, value string) {
		md.Set(meta.UserKey(key), value)
	})
	meta.RangeSys(ctx, func(key string, value interface{}) {
		if meta.IsOutgoing(key) {
			if str, ok := value.(string); ok {
				md.Set(key, str)
			} else if fmtStr, ok := value.(fmt.Stringer); ok {
				md.Set(key, fmtStr.String())
			}
		}
	})
	md.Set(meta.GroupID, env.GroupID())
	md.Set(meta.ServiceNamespace, env.NamespaceID())
	md.Set(meta.ApplicationID, env.ApplicationID())
	md.Set(meta.ApplicationVersion, env.ProgVersion())
	return metadata.MergeToClientContext(ctx, md)
}

func clientMiddleware() middleware.Middleware {
	router := composite.DefaultComposite()
	lane := router.Lane()
	var remoteServiceName string
	var once sync.Once
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			once.Do(func() {
				tr, _ := transport.FromClientContext(ctx)
				remoteServiceName, _ = util.ParseTarget(tr.Endpoint())
			})
			_, operation := ClientOperation(ctx)
			ctx = startClientContext(ctx, remoteServiceName, lane, operation)

			reply, err = handler(ctx, req)
			return
		}
	}
}

// ClientMiddleware is client middleware
func ClientMiddleware() middleware.Middleware {
	return middleware.Chain(clientMiddleware(), tracingClient(), clientMetricsMiddleware(), mmeta.Client())
}

func ClientGrpcOptions(copts ...ClientOption) []tgrpc.ClientOption {
	var o clientOpionts = clientOpionts{
		m:        []middleware.Middleware{clientMiddleware(), tracingClient(copts...), clientMetricsMiddleware(), breakerMiddleware(copts...), mmeta.Client()},
		balancer: p2c.New(nil),
		//balancer: random.New(),
		//balancer: hash.New(),
	}
	for _, opt := range copts {
		opt(&o)
	}

	var opts []tgrpc.ClientOption
	// 将负载均衡模块注册至grpc
	multi.Register(composite.DefaultComposite(), o.balancer)
	opts = []tgrpc.ClientOption{
		tgrpc.WithOptions(grpc.WithBalancerName(o.balancer.Schema()), grpc.WithStatsHandler(&tracing.ClientHandler{})),
		tgrpc.WithMiddleware(o.m...),
		tgrpc.WithDiscovery(consul.DefaultConsul()),
	}
	return opts
}

func ClientHTTPOptions(copts ...ClientOption) []http.ClientOption {
	var o clientOpionts = clientOpionts{
		m:        []middleware.Middleware{clientMiddleware(), tracingClient(copts...), clientMetricsMiddleware(), breakerMiddleware(copts...), mmeta.Client()},
		balancer: p2c.New(nil),
		//balancer: random.New(),
		//balancer: hash.New(),
	}
	for _, opt := range copts {
		opt(&o)
	}

	var opts []http.ClientOption
	opts = []http.ClientOption{
		http.WithBalancer(httpMulti.New(composite.DefaultComposite(), o.balancer)),
		http.WithMiddleware(o.m...),
		http.WithDiscovery(consul.DefaultConsul()),
	}
	return opts
}
