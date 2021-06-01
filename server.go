package tsf

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-kratos/kratos/v2/middleware"
	kgrpc "github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/gorilla/mux"
	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/propagation/b3"
	"github.com/tencentyun/tsf-go/pkg/auth/authenticator"
	"github.com/tencentyun/tsf-go/pkg/config/consul"
	"github.com/tencentyun/tsf-go/pkg/grpc/status"
	tsfHttp "github.com/tencentyun/tsf-go/pkg/http"
	"github.com/tencentyun/tsf-go/pkg/log"
	"github.com/tencentyun/tsf-go/pkg/meta"
	"github.com/tencentyun/tsf-go/pkg/naming"
	"github.com/tencentyun/tsf-go/pkg/statusError"
	"github.com/tencentyun/tsf-go/pkg/sys/env"
	"github.com/tencentyun/tsf-go/pkg/sys/monitor"
	"github.com/tencentyun/tsf-go/pkg/sys/trace"
	"github.com/tencentyun/tsf-go/pkg/util"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

func spanName(api string) string {
	name := strings.TrimPrefix(api, "/")
	name = strings.Replace(name, "/", ".", -1)
	return name
}

func startContext(ctx context.Context, serviceName string, api string, tracer *zipkin.Tracer) context.Context {
	// add system metadata into ctx
	var sysPairs []meta.SysPair
	var userPairs []meta.UserPair
	var md map[string][]string
	var sc model.SpanContext

	if info, ok := http.FromServerContext(ctx); ok {
		md = info.Request.Header
		sc = tracer.Extract(b3.ExtractHTTP(info.Request))
	} else if gmd, ok := metadata.FromIncomingContext(ctx); ok {
		for k, v := range gmd {
			md[k] = v
		}
		sc = tracer.Extract(b3.ExtractGRPC(&gmd))
	}
	// In practice, ok never seems to be false but add a defensive check.
	if md == nil {
		md = make(map[string][]string)
	}

	name := spanName(api)
	span := tracer.StartSpan(name, zipkin.Kind(model.Server), zipkin.Parent(sc), zipkin.RemoteEndpoint(remoteEndpointFromContext(ctx)))
	ctx = zipkin.NewContext(ctx, span)

	for key, vals := range md {
		if vals[0] == "" {
			continue
		}
		if meta.IsIncomming(key) {
			sysPairs = append(sysPairs, meta.SysPair{Key: meta.SourceKey(key), Value: vals[0]})
		} else if meta.IsUserKey(key) {
			userPairs = append(userPairs, meta.UserPair{Key: meta.GetUserKey(key), Value: vals[0]})
		} else if meta.IsLinkKey(key) {
			sysPairs = append(sysPairs, meta.SysPair{Key: key, Value: vals[0]})
		} else if key == "tsf-metadata" {
			var tsfMeta tsfHttp.Metadata
			e := json.Unmarshal([]byte(vals[0]), &tsfMeta)
			if e != nil {
				v, e := url.QueryUnescape(vals[0])
				if e == nil {
					e = json.Unmarshal([]byte(v), &tsfMeta)
				} else {
					log.Info(ctx, "grpc http parse header TSF-Metadata failed!", zap.String("meta", v), zap.Error(e))
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
			e := json.Unmarshal([]byte(vals[0]), &tags)
			if e != nil {
				v, e := url.QueryUnescape(vals[0])
				if e == nil {
					e = json.Unmarshal([]byte(v), &tags)
				} else {
					log.Info(ctx, "grpc http parse header TSF-Tags failed!", zap.String("tags", vals[0]), zap.Error(e))
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
	sysPairs = append(sysPairs, meta.SysPair{Key: meta.Interface, Value: api})
	sysPairs = append(sysPairs, meta.SysPair{Key: meta.Tracer, Value: tracer})
	sysPairs = append(sysPairs, meta.SysPair{Key: meta.GroupID, Value: env.GroupID()})
	sysPairs = append(sysPairs, meta.SysPair{Key: meta.ApplicationID, Value: env.ApplicationID()})
	sysPairs = append(sysPairs, meta.SysPair{Key: meta.ApplicationVersion, Value: env.ProgVersion()})
	sysPairs = append(sysPairs, meta.SysPair{Key: meta.ConnnectionIP, Value: env.LocalIP()})
	ctx = meta.WithSys(ctx, sysPairs...)
	ctx = meta.WithUser(ctx, userPairs...)
	return ctx
}

func remoteEndpointFromContext(ctx context.Context) *model.Endpoint {
	remoteAddr := ""

	p, ok := peer.FromContext(ctx)
	if ok {
		remoteAddr = p.Addr.String()
	}
	var name string
	name, _ = meta.Sys(ctx, meta.SourceKey(meta.ServiceName)).(string)
	ep, _ := zipkin.NewEndpoint(name, remoteAddr)
	return ep
}

func getStat(serviceName string, method string) *monitor.Stat {
	return monitor.NewStat(monitor.CategoryMS, monitor.KindServer, &monitor.Endpoint{ServiceName: serviceName, InterfaceName: method, Path: method, Method: "POST"}, nil)
}

// ServerMiddleware is a grpc server middleware.
func ServerMiddleware(serviceName string, port int) middleware.Middleware {
	builder := &authenticator.Builder{}
	authen := builder.Build(consul.DefaultConsul(), naming.NewService(env.NamespaceID(), serviceName))
	// create our local service endpoint
	endpoint, err := zipkin.NewEndpoint(serviceName, fmt.Sprintf("%s:%d", env.LocalIP(), port))
	if err != nil {
		panic(err)
	}
	// initialize our tracer
	tracer, err := zipkin.NewTracer(trace.GetReporter(), zipkin.WithLocalEndpoint(endpoint))
	if err != nil {
		panic(err)
	}

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (resp interface{}, err error) {
			var api string
			var method string
			if info, ok := kgrpc.FromServerContext(ctx); ok {
				api = info.FullMethod
				method = "POST"
			} else if info, ok := http.FromServerContext(ctx); ok {
				req := info.Request.WithContext(ctx)
				method = req.Method
				if route := mux.CurrentRoute(req); route != nil {
					// /path/123 -> /path/{id}
					api, _ = route.GetPathTemplate()
				} else {
					api = req.URL.Path
				}
			}
			ctx = startContext(ctx, serviceName, api, tracer)
			stat := getStat(serviceName, api)
			span := zipkin.SpanFromContext(ctx)
			span.Tag("http.method", method)
			span.Tag("localInterface", api)
			span.Tag("http.path", api)
			span.SetRemoteEndpoint(remoteEndpointFromContext(ctx))
			defer func() {
				var code = 200
				if err != nil {
					if ec, ok := err.(*statusError.StatusError); ok {
						code = int(ec.Code())
					} else {
						code = 500
					}
					span.Tag("exception", err.Error())
					err = status.ToGrpcStatus(err)
				}
				span.Tag("resultStatus", strconv.FormatInt(int64(code), 10))
				stat.Record(code)

				if err != nil {
					zipkin.TagError.Set(span, err.Error())
				}
				span.Finish()
			}()

			// 鉴权
			err = authen.Verify(ctx, api)
			if err != nil {
				return
			}

			resp, err = handler(ctx, req)
			return
		}
	}
}
