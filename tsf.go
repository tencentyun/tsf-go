package tsf

import (
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/middleware"
	tgrpc "github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/go-kratos/swagger-api/openapiv2"
	"github.com/tencentyun/tsf-go/balancer/random"
	"github.com/tencentyun/tsf-go/grpc/balancer/multi"
	httpMulti "github.com/tencentyun/tsf-go/http/balancer/multi"
	"github.com/tencentyun/tsf-go/naming/consul"
	"github.com/tencentyun/tsf-go/pkg/sys/env"
	"github.com/tencentyun/tsf-go/pkg/version"
	"github.com/tencentyun/tsf-go/route/composite"
	"google.golang.org/grpc"
)

// Option is HTTP server option.
type Option func(*serverOptions)

func ProtoServiceName(fullname string) Option {
	return func(s *serverOptions) {
		s.protoService = fullname
	}
}

func GRPCServer(srv *grpc.Server) Option {
	return func(s *serverOptions) {
		s.srv = srv
	}
}

type serverOptions struct {
	protoService string
	srv          *grpc.Server
	apiMeta      bool
}

func APIMeta(enable bool) Option {
	return func(s *serverOptions) {
		s.apiMeta = enable
	}
}

func Metadata(optFuncs ...Option) (opt kratos.Option) {
	enableApiMeta := true
	if env.Token() == "" {
		enableApiMeta = false
	}

	var opts serverOptions = serverOptions{}
	for _, o := range optFuncs {
		o(&opts)
	}
	if opts.apiMeta {
		enableApiMeta = true
	}

	md := map[string]string{
		"TSF_APPLICATION_ID": env.ApplicationID(),
		"TSF_GROUP_ID":       env.GroupID(),
		"TSF_INSTNACE_ID":    env.InstanceId(),
		"TSF_PROG_VERSION":   env.ProgVersion(),
		"TSF_ZONE":           env.Zone(),
		"TSF_REGION":         env.Region(),
		"TSF_NAMESPACE_ID":   env.NamespaceID(),
		"TSF_SDK_VERSION":    version.GetHumanVersion(),
	}
	if enableApiMeta {
		apiSrv := openapiv2.New(opts.srv)
		genAPIMeta(md, apiSrv, opts.protoService)
	}

	opt = kratos.Metadata(md)
	return
}

func ID(optFuncs ...Option) kratos.Option {
	return kratos.ID(env.InstanceId())
}
func Registrar(optFuncs ...Option) kratos.Option {
	return kratos.Registrar(consul.DefaultConsul())
}

func ClientGrpcOptions(m ...middleware.Middleware) []tgrpc.ClientOption {
	var opts []tgrpc.ClientOption

	m = append(m, ClientMiddleware())
	// 将wrr负载均衡模块注入至grpc
	router := composite.DefaultComposite()
	multi.Register(router)
	opts = []tgrpc.ClientOption{
		tgrpc.WithOptions(grpc.WithBalancerName("tsf-random")),
		tgrpc.WithMiddleware(m...),
		tgrpc.WithDiscovery(consul.DefaultConsul()),
	}
	return opts
}

func ClientHTTPOptions(m ...middleware.Middleware) []http.ClientOption {
	var opts []http.ClientOption

	router := composite.DefaultComposite()
	b := &random.Picker{}
	m = append(m, ClientMiddleware())
	opts = []http.ClientOption{
		http.WithBalancer(httpMulti.New(router, b)),
		http.WithMiddleware(m...),
		http.WithDiscovery(consul.DefaultConsul()),
	}
	return opts
}

func AppOptions(optFuncs ...Option) []kratos.Option {
	return []kratos.Option{
		ID(optFuncs...), Registrar(optFuncs...), Metadata(optFuncs...),
	}
}
