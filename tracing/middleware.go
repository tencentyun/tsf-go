package tracing

import (
	"context"
	"net/url"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/tencentyun/tsf-go/gin"
	"github.com/tencentyun/tsf-go/pkg/meta"
	"github.com/tencentyun/tsf-go/util"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/peer"
)

// Server returns a new server middleware for OpenTelemetry.
func Server(opts ...Option) middleware.Middleware {
	tracer, e := NewTracer(trace.SpanKindServer, opts...)
	if e != nil {
		panic(e)
	}
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			if tr, ok := transport.FromServerContext(ctx); ok {
				method := "POST"
				operation := tr.Operation()
				path := tr.Operation()
				var remote string
				if tr.Kind() == transport.KindHTTP {
					if ht, ok := tr.(*http.Transport); ok {
						operation = ht.PathTemplate()
						method = ht.Request().Method
						path = ht.Request().URL.Path
						remote = ht.Request().RemoteAddr
					} else if c, ok := gin.FromGinContext(ctx); ok {
						operation = c.Ctx.FullPath()
						method = c.Ctx.Request.Method
						path = c.Ctx.Request.URL.Path
						remote = c.Ctx.Request.RemoteAddr
					}
				} else if tr.Kind() == transport.KindGRPC {
					if p, ok := peer.FromContext(ctx); ok {
						remote = p.Addr.String()
					}
				}

				var span trace.Span
				ctx, span = tracer.Start(ctx, tr.Kind().String(), operation, tr.Header())
				span.SetAttributes(attribute.String("localComponent", tr.Kind().String()))
				k, _ := kratos.FromContext(ctx)
				span.SetAttributes(attribute.String("local.service", k.Name()))
				u, _ := url.Parse(tr.Endpoint())
				if u != nil {
					localIP, localPort := util.ParseAddr(u.Host)
					span.SetAttributes(attribute.String("local.ip", localIP))
					span.SetAttributes(attribute.Int64("local.port", int64(localPort)))
				}
				if name, ok := meta.Sys(ctx, meta.SourceKey(meta.ServiceName)).(string); ok {
					span.SetAttributes(attribute.String("peer.service", name))
				}
				remoteIP, remotePort := util.ParseAddr(remote)
				span.SetAttributes(attribute.String("peer.ip", remoteIP))
				span.SetAttributes(attribute.Int64("peer.port", int64(remotePort)))
				span.SetAttributes(attribute.String("http.method", method))
				span.SetAttributes(attribute.String("localInterface", operation))
				span.SetAttributes(attribute.String("http.path", path))
				defer func() { tracer.End(ctx, span, err) }()
			}

			reply, err = handler(ctx, req)
			return
		}
	}
}

// Client returns a new client middleware for OpenTelemetry.
func Client(opts ...Option) middleware.Middleware {
	tracer, e := NewTracer(trace.SpanKindClient, opts...)
	if e != nil {
		panic(e)
	}
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			if tr, ok := transport.FromClientContext(ctx); ok {
				operation := tr.Operation()
				path := tr.Operation()
				method := "POST"
				if tr.Kind() == transport.KindHTTP {
					if ht, ok := tr.(*http.Transport); ok {
						operation = ht.PathTemplate()
						method = ht.Request().Method
						path = ht.Request().URL.Path
					}
				}
				var span trace.Span
				ctx, span = tracer.Start(ctx, tr.Kind().String(), operation, tr.Header())

				span.SetAttributes(attribute.String("remoteComponent", tr.Kind().String()))
				var localAddr string
				if str, ok := transport.FromServerContext(ctx); ok {
					span.SetAttributes(attribute.String("localComponent", str.Kind().String()))
					span.SetAttributes(attribute.String("localInterface", str.Operation()))
					u, _ := url.Parse(str.Endpoint())
					localAddr = u.Host
				}
				k, _ := kratos.FromContext(ctx)
				span.SetAttributes(attribute.String("local.service", k.Name()))
				localIP, localPort := util.ParseAddr(localAddr)
				span.SetAttributes(attribute.String("local.ip", localIP))
				span.SetAttributes(attribute.Int64("local.port", int64(localPort)))

				remoteService, _ := util.ParseTarget(tr.Endpoint())
				span.SetAttributes(attribute.String("peer.service", remoteService))
				span.SetAttributes(attribute.String("http.method", method))
				span.SetAttributes(attribute.String("http.path", path))
				defer func() {
					if tr.Kind() == transport.KindHTTP {
						if ht, ok := tr.(*http.Transport); ok {
							remoteIP, remotePort := util.ParseAddr(ht.Request().Host)
							span.SetAttributes(attribute.String("peer.ip", remoteIP))
							span.SetAttributes(attribute.Int64("peer.port", int64(remotePort)))
						}
					}
					tracer.End(ctx, span, err)
				}()
			}

			reply, err = handler(ctx, req)
			return
		}
	}
}
