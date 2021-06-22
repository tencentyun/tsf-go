package tsf

import (
	"context"
	"sync"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/tencentyun/tsf-go/pkg/auth"
	"github.com/tencentyun/tsf-go/pkg/auth/authenticator"
	"github.com/tencentyun/tsf-go/pkg/config/consul"
	"github.com/tencentyun/tsf-go/pkg/naming"
	"github.com/tencentyun/tsf-go/pkg/sys/env"
)

func authMiddleware() middleware.Middleware {
	var authen auth.Auth
	var once sync.Once

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (resp interface{}, err error) {
			once.Do(func() {
				k, _ := kratos.FromContext(ctx)
				serviceName := k.Name()
				builder := &authenticator.Builder{}
				authen = builder.Build(consul.DefaultConsul(), naming.NewService(env.NamespaceID(), serviceName))
			})
			_, operation := ServerOperation(ctx)
			// 鉴权
			err = authen.Verify(ctx, operation)
			if err != nil {
				return
			}
			return handler(ctx, req)
		}
	}
}
