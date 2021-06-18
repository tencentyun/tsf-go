package tsf

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"sync"

	"github.com/tencentyun/tsf-go/pkg/grpc/balancer/multi"
	"github.com/tencentyun/tsf-go/pkg/meta"
	"github.com/tencentyun/tsf-go/pkg/route/composite"
	"github.com/tencentyun/tsf-go/pkg/route/lane"
	"github.com/tencentyun/tsf-go/pkg/sys/env"
	"github.com/tencentyun/tsf-go/pkg/sys/monitor"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/metadata"
	"github.com/go-kratos/kratos/v2/middleware"
	mmeta "github.com/go-kratos/kratos/v2/middleware/metadata"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/propagation/b3"
	gmetadata "google.golang.org/grpc/metadata"
)

func getClientStat(ctx context.Context, remoteServiceName string, operation string, method string) *monitor.Stat {
	localService, _ := meta.Sys(ctx, meta.ServiceName).(string)
	localOperation, _ := meta.Sys(ctx, meta.Interface).(string)
	localMethod, _ := meta.Sys(ctx, meta.RequestHTTPMethod).(string)

	return monitor.NewStat(monitor.CategoryMS, monitor.KindClient, &monitor.Endpoint{ServiceName: localService, InterfaceName: localOperation, Path: localOperation, Method: localMethod}, &monitor.Endpoint{ServiceName: remoteServiceName, InterfaceName: operation, Path: operation, Method: method})
}

func startClientContext(ctx context.Context, remoteServiceName string, l *lane.Lane, operation string) context.Context {
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

// ClientMiddleware is client middleware
func ClientMiddleware() middleware.Middleware {
	return middleware.Chain(clientMiddleware(), mmeta.Client())
}

func clientMiddleware() middleware.Middleware {
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
			var operation string
			var path string
			var method string = "POST"
			if tr, ok := transport.FromClientContext(ctx); ok {
				operation = tr.Operation()
				path = operation
				if tr.Kind() == transport.KindHTTP {
					if ht, ok := tr.(*http.Transport); ok {
						operation = ht.PathTemplate()
						method = ht.Request().Method
						path = ht.Request().URL.Path
					}
				}
			}

			ctx = startClientContext(ctx, remoteServiceName, lane, operation)
			ctx = startClientSpan(ctx, method, operation, path)
			stat := getClientStat(ctx, remoteServiceName, operation, method)
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

func startClientSpan(ctx context.Context, method string, operation string, path string) context.Context {
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
	span := tracer.StartSpan(operation, options...)
	ctx = zipkin.NewContext(ctx, span)

	localAPI, _ := meta.Sys(ctx, meta.Interface).(string)
	span.Tag("http.method", method)
	span.Tag("localInterface", localAPI)
	span.Tag("http.path", path)

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
