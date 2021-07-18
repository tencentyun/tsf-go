package tsf

import (
	"context"
	"sync"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/tencentyun/tsf-go/pkg/meta"
	"github.com/tencentyun/tsf-go/pkg/sys/monitor"
	"github.com/tencentyun/tsf-go/util"
)

func getStat(serviceName string, operation string, method string) *monitor.Stat {
	return monitor.NewStat(monitor.CategoryMS, monitor.KindServer, &monitor.Endpoint{ServiceName: serviceName, InterfaceName: operation, Path: operation, Method: method}, nil)
}

func getClientStat(ctx context.Context, remoteServiceName string, operation string, method string) *monitor.Stat {
	localService, _ := meta.Sys(ctx, meta.ServiceName).(string)
	localOperation, _ := meta.Sys(ctx, meta.Interface).(string)
	localMethod, _ := meta.Sys(ctx, meta.RequestHTTPMethod).(string)

	return monitor.NewStat(monitor.CategoryMS, monitor.KindClient, &monitor.Endpoint{ServiceName: localService, InterfaceName: localOperation, Path: localOperation, Method: localMethod}, &monitor.Endpoint{ServiceName: remoteServiceName, InterfaceName: operation, Path: operation, Method: method})
}

func serverMetricsMiddleware() middleware.Middleware {
	var (
		once        sync.Once
		serviceName string
	)
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			once.Do(func() {
				k, _ := kratos.FromContext(ctx)
				serviceName = k.Name()
			})

			method, operation := ServerOperation(ctx)
			stat := getStat(serviceName, operation, method)
			defer func() {
				var code = 200
				if err != nil {
					code = int(errors.FromError(err).GetCode())
				}
				stat.Record(code)
			}()

			reply, err = handler(ctx, req)
			return
		}
	}
}
func clientMetricsMiddleware() middleware.Middleware {
	var remoteServiceName string
	var once sync.Once
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			once.Do(func() {
				tr, _ := transport.FromClientContext(ctx)
				remoteServiceName, _ = util.ParseTarget(tr.Endpoint())
			})

			method, operation := ClientOperation(ctx)
			stat := getClientStat(ctx, remoteServiceName, operation, method)
			defer func() {
				var code = 200
				if err != nil {
					code = int(errors.FromError(err).GetCode())
				}
				stat.Record(code)
			}()

			reply, err = handler(ctx, req)
			return
		}
	}
}
