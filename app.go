package tsf

import (
	"github.com/tencentyun/tsf-go/naming/consul"
	"github.com/tencentyun/tsf-go/pkg/sys/env"
	"github.com/tencentyun/tsf-go/pkg/version"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/swagger-api/openapiv2"
	"google.golang.org/grpc"
)

// Option is HTTP server option.
type Option func(*appOptions)

func ProtoServiceName(fullname string) Option {
	return func(s *appOptions) {
		s.protoService = fullname
	}
}

func GRPCServer(srv *grpc.Server) Option {
	return func(s *appOptions) {
		s.srv = srv
	}
}

type appOptions struct {
	protoService string
	srv          *grpc.Server
	apiMeta      bool
}

func APIMeta(enable bool) Option {
	return func(s *appOptions) {
		s.apiMeta = enable
	}
}

func Metadata(optFuncs ...Option) (opt kratos.Option) {
	enableApiMeta := true
	if env.Token() == "" {
		enableApiMeta = false
	}

	var opts appOptions = appOptions{}
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

func AppOptions(optFuncs ...Option) []kratos.Option {
	return []kratos.Option{
		ID(optFuncs...), Registrar(optFuncs...), Metadata(optFuncs...),
	}
}
