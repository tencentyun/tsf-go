package client

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/tencentyun/tsf-go/pkg/grpc/balancer/wrr"
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

// ClientConn is the framework's client side instance, it contains the ctx, opt and interceptors.
// Create an instance of Client, by using NewClient().
type ClientConn struct {
	*grpc.ClientConn
	remoteService naming.Service
	interceptors  []grpc.UnaryClientInterceptor
	opts          []grpc.DialOption
	lane          *lane.Lane
}

// DialWithBlock create a grpc client conn with context
// It will block until create connection successfully
// 核心依赖建议使用DialWithBlock，确保能够拉到服务提供者节点再进行后续的启动操作
func DialWithBlock(ctx context.Context, target string, opts ...grpc.DialOption) (c *ClientConn, err error) {
	c = &ClientConn{}
	c.setup(target, true, opts...)
	if c.ClientConn, err = grpc.DialContext(ctx, target, c.opts...); err != nil {
		return
	}
	return
}

// Dial create a grpc client conn
// It will return immediately without any blocking
func Dial(target string, opts ...grpc.DialOption) (c *ClientConn, err error) {
	c = &ClientConn{}
	c.setup(target, false, opts...)
	if c.ClientConn, err = grpc.Dial(target, c.opts...); err != nil {
		return
	}
	return
}

func (c *ClientConn) setup(target string, block bool, o ...grpc.DialOption) error {
	util.ParseFlag()
	// 将consul服务发现模块注入至grpc
	resolver.Register(consul.DefaultConsul())
	// 将wrr负载均衡模块注入至grpc
	balancer := composite.DefaultComposite()
	wrr.Register(balancer)
	// 加载框架自带的middleware
	c.Use(c.handle)
	c.lane = balancer.Lane()

	c.opts = append(c.opts,
		grpc.WithInsecure(),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                time.Second * 30,
			Timeout:             time.Second * 10,
			PermitWithoutStream: true,
		}),
		grpc.WithUnaryInterceptor(c.chainUnaryClient()),
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
		c.opts = append(c.opts, grpc.WithBalancerName(wrr.Name))
	}
	return nil
}

// Use attachs a global inteceptor to the Client.
// For example, this is the right place for a circuit breaker or error management inteceptor.
// This function is not concurrency safe.
func (c *ClientConn) Use(interceptors ...grpc.UnaryClientInterceptor) *ClientConn {
	c.interceptors = append(c.interceptors, interceptors...)
	return c
}

// ChainUnaryClient creates a single interceptor out of a chain of many interceptors.
//
// Execution is done in left-to-right order, including passing of context.
// For example ChainUnaryClient(one, two, three) will execute one before two before three.
func (c *ClientConn) chainUnaryClient() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		n := len(c.interceptors)

		chainer := func(currentInter grpc.UnaryClientInterceptor, currentInvoker grpc.UnaryInvoker) grpc.UnaryInvoker {
			return func(currentCtx context.Context, currentMethod string, currentReq, currentRepl interface{}, currentConn *grpc.ClientConn, currentOpts ...grpc.CallOption) error {
				return currentInter(currentCtx, currentMethod, currentReq, currentRepl, currentConn, currentInvoker, currentOpts...)
			}
		}

		chainedInvoker := invoker
		for i := n - 1; i >= 0; i-- {
			chainedInvoker = chainer(c.interceptors[i], chainedInvoker)
		}

		return chainedInvoker(ctx, method, req, reply, cc, opts...)
	}
}
