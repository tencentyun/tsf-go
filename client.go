package tsf

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"sync"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/metadata"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/propagation/b3"
	"github.com/tencentyun/tsf-go/pkg/grpc/balancer/multi"
	"github.com/tencentyun/tsf-go/pkg/meta"
	"github.com/tencentyun/tsf-go/pkg/route/composite"
	"github.com/tencentyun/tsf-go/pkg/route/lane"
	"github.com/tencentyun/tsf-go/pkg/sys/env"
	"github.com/tencentyun/tsf-go/pkg/sys/monitor"
	gmetadata "google.golang.org/grpc/metadata"
)

func getClientStat(ctx context.Context, remoteServiceName string, method string) *monitor.Stat {
	localService, ok := meta.Sys(ctx, meta.ServiceName).(string)
	if !ok {
		localService = env.ServiceName()
	}
	localMethod, ok := meta.Sys(ctx, meta.Interface).(string)
	if !ok {
		localMethod = "/defaultInterface"
	}
	return monitor.NewStat(monitor.CategoryMS, monitor.KindClient, &monitor.Endpoint{ServiceName: localService, InterfaceName: localMethod, Path: localMethod, Method: "POST"}, &monitor.Endpoint{ServiceName: remoteServiceName, InterfaceName: method})
}

func startClientContext(ctx context.Context, remoteServiceName string, l *lane.Lane, api string) context.Context {
	// 注入远端服务名
	pairs := []meta.SysPair{
		{Key: meta.DestKey(meta.ServiceName), Value: remoteServiceName},
		{Key: meta.DestKey(meta.ServiceNamespace), Value: env.NamespaceID()},
	}
	// 注入自己的服务名
	k, _ := kratos.FromContext(ctx)
	serviceName := k.Name
	if res := meta.Sys(ctx, meta.ServiceName); res == nil {
		pairs = append(pairs, meta.SysPair{Key: meta.ServiceName, Value: serviceName})
	} else {
		serviceName = res.(string)
	}

	pairs = append(pairs, meta.SysPair{Key: meta.DestKey(meta.Interface), Value: api})
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

func parseTarget(endpoint string) (string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		if u, err = url.Parse("http://" + endpoint); err != nil {
			return "", err
		}
	}
	var service string
	if len(u.Path) > 1 {
		service = u.Path[1:]
	}
	return service, nil
}

func ClientMiddleware() middleware.Middleware {
	router := composite.DefaultComposite()
	multi.Register(router)
	lane := router.Lane()
	var remoteServiceName string
	var once sync.Once
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			once.Do(func() {
				tr, _ := transport.FromClientContext(ctx)
				remoteServiceName, _ = parseTarget(tr.Endpoint())
			})
			var api string
			var method string = "POST"
			if tr, ok := transport.FromClientContext(ctx); ok {
				api = tr.Operation()
				if tr.Kind() == transport.KindHTTP {
					if ht, ok := tr.(*http.Transport); ok {
						method = ht.Request().Method
					}
				}
			}

			ctx = startClientContext(ctx, remoteServiceName, lane, api)
			ctx = startClientSpan(ctx, method, api)
			stat := getClientStat(ctx, remoteServiceName, api)
			defer func() {
				var code = 200
				if err != nil {
					code = errors.FromError(err).StatusCode()
				}
				stat.Record(code)
				span := zipkin.SpanFromContext(ctx)
				if span != nil {
					if err != nil {
						span.Tag("exception", err.Error())
					}
					span.Tag("resultStatus", strconv.FormatInt(int64(code), 10))
					span.Finish()
				}
			}()
			reply, err = handler(ctx, req)
			return
		}
	}
}

func startClientSpan(ctx context.Context, method string, api string) context.Context {
	tracer, _ := meta.Sys(ctx, meta.Tracer).(*zipkin.Tracer)
	if tracer == nil {
		tracer, _ = ctx.Value(meta.Tracer).(*zipkin.Tracer)
		if tracer == nil {
			return ctx
		}
	}

	parentSpan := zipkin.SpanFromContext(ctx)
	if parentSpan == nil {
		parentSpan, _ = ctx.Value("tsf.spankey").(zipkin.Span)
	}

	options := []zipkin.SpanOption{zipkin.Kind(model.Client)}
	if parentSpan != nil {
		options = append(options, zipkin.Parent(parentSpan.Context()))
	}
	span := tracer.StartSpan(api, options...)
	ctx = zipkin.NewContext(ctx, span)

	span.Tag("http.method", method)
	span.Tag("localInterface", api)
	span.Tag("http.path", api)

	if tr, ok := transport.FromClientContext(ctx); ok {
		if tr.Kind() == transport.KindHTTP {
			if ht, ok := tr.(*http.Transport); ok {
				b3.InjectHTTP(ht.Request())(span.Context())
			}
		} else if tr.Kind() == transport.KindGRPC {
			var gmd gmetadata.MD
			if gmd, ok = gmetadata.FromOutgoingContext(ctx); !ok {
				gmd = gmetadata.New(nil)
				ctx = gmetadata.NewOutgoingContext(ctx, gmd)
			}
			b3.InjectGRPC(&gmd)(span.Context())
		}
	}
	return ctx
}
