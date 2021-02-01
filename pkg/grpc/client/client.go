package client

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/tencentyun/tsf-go/pkg/balancer/random"
	"github.com/tencentyun/tsf-go/pkg/grpc/balancer/multi"
	"github.com/tencentyun/tsf-go/pkg/grpc/resolver"
	"github.com/tencentyun/tsf-go/pkg/naming"
	"github.com/tencentyun/tsf-go/pkg/naming/consul"
	"github.com/tencentyun/tsf-go/pkg/route/composite"
	"github.com/tencentyun/tsf-go/pkg/route/lane"
	"github.com/tencentyun/tsf-go/pkg/sys/env"
	"github.com/tencentyun/tsf-go/pkg/util"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

// Conn is the framework's client side instance, it contains the ctx, opt and interceptors.
// Create an instance of Client, by using NewClient().
type Conn struct {
	*grpc.ClientConn
	remoteService      naming.Service
	interceptors       []grpc.UnaryClientInterceptor
	streamInterceptors []grpc.StreamClientInterceptor

	opts []grpc.DialOption
	lane *lane.Lane
}

// DialWithBlock create a grpc client conn with context
// It will block until create connection successfully
// 核心依赖建议使用DialWithBlock，确保能够拉到服务提供者节点再进行后续的启动操作
func DialWithBlock(ctx context.Context, target string, opts ...grpc.DialOption) (c *Conn, err error) {
	c = &Conn{}
	c.setup(target, true, opts...)
	if c.ClientConn, err = grpc.DialContext(ctx, target, c.opts...); err != nil {
		return
	}
	return
}

// Dial create a grpc client conn
// It will return immediately without any blocking
func Dial(target string, opts ...grpc.DialOption) (c *Conn, err error) {
	c = &Conn{}
	c.setup(target, false, opts...)
	if c.ClientConn, err = grpc.Dial(target, c.opts...); err != nil {
		return
	}
	return
}

func (c *Conn) setup(target string, block bool, o ...grpc.DialOption) error {
	util.ParseFlag()
	// 将consul服务发现模块注入至grpc
	resolver.Register(consul.DefaultConsul())
	// 将wrr负载均衡模块注入至grpc
	router := composite.DefaultComposite()
	multi.Register(router)
	// 加载框架自带的middleware
	c.Use(c.handle)
	c.UseStream(c.handleStream)
	c.lane = router.Lane()

	c.opts = append(c.opts,
		grpc.WithInsecure(),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                time.Second * 30,
			Timeout:             time.Second * 10,
			PermitWithoutStream: true,
		}),
		grpc.WithUnaryInterceptor(c.chainUnaryClient()),
		grpc.WithStreamInterceptor(c.chainStreamClient()),
	)
	if block {
		c.opts = append(c.opts, grpc.WithBlock())
	}
	// opts can be overwritten by user defined grpc options
	c.opts = append(c.opts, o...)
	if raw, err := url.Parse(target); err == nil {
		if raw.Host == "" || raw.Host == "local" {
			c.remoteService.Namespace = env.NamespaceID()
		} else {
			c.remoteService.Namespace = raw.Host
		}
		c.remoteService.Name = strings.TrimLeft(raw.Path, "/")
		c.opts = append(c.opts, grpc.WithBalancerName(random.Name))
	}
	return nil
}

// GrpcConn exports grpc connection
func (c *Conn) GrpcConn() *grpc.ClientConn {
	return c.ClientConn
}
