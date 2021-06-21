package tsf

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"
	"sync"

	"github.com/tencentyun/tsf-go/gin"
	"github.com/tencentyun/tsf-go/log"
	"github.com/tencentyun/tsf-go/pkg/auth"
	"github.com/tencentyun/tsf-go/pkg/auth/authenticator"
	"github.com/tencentyun/tsf-go/pkg/config/consul"
	tsfHttp "github.com/tencentyun/tsf-go/pkg/http"
	"github.com/tencentyun/tsf-go/pkg/meta"
	"github.com/tencentyun/tsf-go/pkg/naming"
	"github.com/tencentyun/tsf-go/pkg/sys/env"
	"github.com/tencentyun/tsf-go/pkg/sys/monitor"
	"github.com/tencentyun/tsf-go/pkg/util"
	"github.com/tencentyun/tsf-go/tracing"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/metadata"
	"github.com/go-kratos/kratos/v2/middleware"
	mmeta "github.com/go-kratos/kratos/v2/middleware/metadata"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"
	"google.golang.org/grpc/peer"
)

func spanName(api string) string {
	//name := strings.TrimPrefix(api, "/")
	//name = strings.Replace(name, "/", ".", -1)
	return api
}

func startServerContext(ctx context.Context, serviceName string, method string, operation string, addr string) context.Context {
	// add system metadata into ctx
	var (
		sysPairs  []meta.SysPair
		userPairs []meta.UserPair
	)
	md, _ := metadata.FromServerContext(ctx)
	for key, val := range md {
		if key == "" || val == "" {
			continue
		}
		key = strings.ToLower(key)
		if meta.IsIncomming(key) {
			sysPairs = append(sysPairs, meta.SysPair{Key: meta.SourceKey(key), Value: val})
		} else if meta.IsUserKey(key) {
			userPairs = append(userPairs, meta.UserPair{Key: meta.GetUserKey(key), Value: val})
		} else if meta.IsLinkKey(key) {
			sysPairs = append(sysPairs, meta.SysPair{Key: key, Value: val})
		} else if key == "tsf-metadata" {
			var tsfMeta tsfHttp.Metadata
			e := json.Unmarshal([]byte(val), &tsfMeta)
			if e != nil {
				v, e := url.QueryUnescape(val)
				if e == nil {
					e = json.Unmarshal([]byte(v), &tsfMeta)
				} else {
					log.DefaultLog.Infow("msg", "grpc http parse header TSF-Metadata failed!", "meta", v, "err", e)
				}
			}
			sysPairs = append(sysPairs, meta.SysPair{Key: meta.SourceKey(meta.ApplicationID), Value: tsfMeta.ApplicationID})
			sysPairs = append(sysPairs, meta.SysPair{Key: meta.SourceKey(meta.ApplicationVersion), Value: tsfMeta.ApplicationVersion})
			sysPairs = append(sysPairs, meta.SysPair{Key: meta.SourceKey(meta.ServiceName), Value: tsfMeta.ServiceName})
			sysPairs = append(sysPairs, meta.SysPair{Key: meta.SourceKey(meta.GroupID), Value: tsfMeta.GroupID})
			sysPairs = append(sysPairs, meta.SysPair{Key: meta.SourceKey(meta.ConnnectionIP), Value: tsfMeta.LocalIP})
			sysPairs = append(sysPairs, meta.SysPair{Key: meta.SourceKey(meta.Namespace), Value: tsfMeta.NamespaceID})
		} else if key == "tsf-tags" {
			var tags []map[string]interface{} = make([]map[string]interface{}, 0)
			e := json.Unmarshal([]byte(val), &tags)
			if e != nil {
				v, e := url.QueryUnescape(val)
				if e == nil {
					e = json.Unmarshal([]byte(v), &tags)
				} else {
					log.DefaultLog.Info("msg", "grpc http parse header TSF-Tags failed!", "tags", val, "err", e)
				}
			}
			for _, tag := range tags {
				for k, v := range tag {
					if value, ok := v.(string); ok {
						userPairs = append(userPairs, meta.UserPair{Key: k, Value: value})
					}
				}
			}
		}
	}
	if pr, ok := peer.FromContext(ctx); ok {
		sysPairs = append(sysPairs, meta.SysPair{Key: meta.SourceKey(meta.ConnnectionIP), Value: util.IPFromAddr(pr.Addr)})
	}
	sysPairs = append(sysPairs, meta.SysPair{Key: meta.ServiceName, Value: serviceName})
	sysPairs = append(sysPairs, meta.SysPair{Key: meta.Namespace, Value: env.NamespaceID()})
	sysPairs = append(sysPairs, meta.SysPair{Key: meta.Interface, Value: operation})
	sysPairs = append(sysPairs, meta.SysPair{Key: meta.RequestHTTPMethod, Value: method})
	sysPairs = append(sysPairs, meta.SysPair{Key: meta.GroupID, Value: env.GroupID()})
	sysPairs = append(sysPairs, meta.SysPair{Key: meta.ApplicationID, Value: env.ApplicationID()})
	sysPairs = append(sysPairs, meta.SysPair{Key: meta.ApplicationVersion, Value: env.ProgVersion()})
	sysPairs = append(sysPairs, meta.SysPair{Key: meta.ConnnectionIP, Value: addr})
	ctx = meta.WithSys(ctx, sysPairs...)
	ctx = meta.WithUser(ctx, userPairs...)

	return ctx
}

func remoteEndpointFromContext(ctx context.Context, remoteAddr string) *model.Endpoint {
	var name string = ""
	name, _ = meta.Sys(ctx, meta.SourceKey(meta.ServiceName)).(string)
	ep, _ := zipkin.NewEndpoint(name, remoteAddr)
	return ep
}

func getStat(serviceName string, method string) *monitor.Stat {
	return monitor.NewStat(monitor.CategoryMS, monitor.KindServer, &monitor.Endpoint{ServiceName: serviceName, InterfaceName: method, Path: method, Method: "POST"}, nil)
}

// ServerMiddleware is a grpc server middleware.
func ServerMiddleware() middleware.Middleware {
	return middleware.Chain(mmeta.Server(mmeta.WithPropagatedPrefix("")), serverMiddleware(), tracing.Server())
}

// ServerMiddleware is a grpc server middleware.
func serverMiddleware() middleware.Middleware {
	var (
		localAddr   string
		authen      auth.Auth
		once        sync.Once
		serviceName string
	)
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (resp interface{}, err error) {
			once.Do(func() {
				tr, _ := transport.FromServerContext(ctx)
				u, err := url.Parse(tr.Endpoint())
				if err != nil {
					panic(err)
				}
				k, _ := kratos.FromContext(ctx)
				serviceName = k.Name()
				builder := &authenticator.Builder{}
				authen = builder.Build(consul.DefaultConsul(), naming.NewService(env.NamespaceID(), serviceName))
				localAddr = u.Host
			})
			var operation string
			var method string = "POST"
			if tr, ok := transport.FromServerContext(ctx); ok {
				operation = tr.Operation()
				if tr.Kind() == transport.KindHTTP {
					if ht, ok := tr.(*http.Transport); ok {
						operation = ht.PathTemplate()
						method = ht.Request().Method
					}
				}
			} else if c, ok := gin.FromGinContext(ctx); ok {
				operation = c.Ctx.FullPath()
				method = c.Ctx.Request.Method
			}

			ctx = startServerContext(ctx, serviceName, method, operation, localAddr)
			stat := getStat(serviceName, operation)

			defer func() {
				var code = 200
				if err != nil {
					code = errors.FromError(err).StatusCode()
				}
				stat.Record(code)
			}()

			// 鉴权
			err = authen.Verify(ctx, operation)
			if err != nil {
				return
			}

			resp, err = handler(ctx, req)
			return
		}
	}
}
