package main

import (
	"context"
	"fmt"
	"time"

	"github.com/tencentyun/tsf-go/pkg/grpc/client"
	"github.com/tencentyun/tsf-go/pkg/meta"
	"github.com/tencentyun/tsf-go/pkg/sys/env"
	"github.com/tencentyun/tsf-go/pkg/sys/trace"
	pb "github.com/tencentyun/tsf-go/testdata"

	"github.com/gin-gonic/gin"
	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/propagation/b3"
	"github.com/openzipkin/zipkin-go/reporter"
)

type noopReporter struct{}

func (r *noopReporter) Send(model.SpanModel) {}
func (r *noopReporter) Close() error         { return nil }

// NewNoopReporter returns a no-op Reporter implementation.
func NewNoopReporter() reporter.Reporter {
	return &noopReporter{}
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	cc, err := client.DialWithBlock(ctx, "consul://local/provider-demo")
	if err != nil {
		panic(err)
	}
	greeter := pb.NewGreeterClient(cc.GrpcConn())

	newHTTP(greeter)
}

func newHTTP(client pb.GreeterClient) {
	// create our local service endpoint
	endpoint, err := zipkin.NewEndpoint("provider-demo", fmt.Sprintf("%s:%d", env.LocalIP(), 8080))
	if err != nil {
		panic(err)
	}
	// initialize our tracer
	serverTracer, err := zipkin.NewTracer(&noopReporter{}, zipkin.WithLocalEndpoint(endpoint))
	if err != nil {
		panic(err)
	}
	clientTracer, err := zipkin.NewTracer(trace.GetReporter(), zipkin.WithLocalEndpoint(endpoint))
	if err != nil {
		panic(err)
	}
	engine := gin.Default()
	g := engine.Group("/").Use(func(c *gin.Context) {
		var spanName string
		r := c.Request

		// try to extract B3 Headers from upstream
		sc := serverTracer.Extract(b3.ExtractHTTP(r))
		remoteEndpoint, _ := zipkin.NewEndpoint("", c.ClientIP())
		spanName = r.Method
		// create Span using SpanContext if found
		sp := serverTracer.StartSpan(
			spanName,
			zipkin.Kind(model.Server),
			zipkin.Parent(sc),
			zipkin.RemoteEndpoint(remoteEndpoint),
		)
		sp.Tag("http.method", "POST")
		sp.Tag("localInterface", r.Method)
		sp.Tag("http.path", r.Method)

		// add our span to context
		c.Set("tsf.spankey", sp)
		c.Set(meta.Tracer, clientTracer)
		// tag typical HTTP request items
		zipkin.TagHTTPMethod.Set(sp, r.Method)
		zipkin.TagHTTPPath.Set(sp, r.URL.Path)

		// tag found response size and status code on exit
		defer func() {
			sp.Finish()
		}()
		c.Next()
	})

	g.GET("/ping", func(c *gin.Context) {
		reply, err := client.SayHello(c, &pb.HelloRequest{Name: "test xiaomin"})
		if err != nil {
			c.JSON(500, gin.H{
				"err": err.Error(),
			})
			return
		}
		c.JSON(200, gin.H{
			"reply": reply,
		})
	})
	engine.Run(":8081")
}
