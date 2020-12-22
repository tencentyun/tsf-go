package client

import (
	"context"

	"google.golang.org/grpc"
)

// Use attachs a global inteceptor to the Client.
// For example, this is the right place for a circuit breaker or error management inteceptor.
// This function is not concurrency safe.
func (c *ClientConn) Use(interceptors ...grpc.UnaryClientInterceptor) *ClientConn {
	c.interceptors = append(c.interceptors, interceptors...)
	return c
}

// UseStream attachs a global inteceptor to the Client.
// For example, this is the right place for a circuit breaker or error management inteceptor.
// This function is not concurrency safe.
func (c *ClientConn) UseStream(interceptors ...grpc.StreamClientInterceptor) *ClientConn {
	c.streamInterceptors = append(c.streamInterceptors, interceptors...)
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

// chainStreamClient creates a single interceptor out of a chain of many interceptors.
//
// Execution is done in left-to-right order, including passing of context.
// For example ChainStreamClient(one, two, three) will execute one before two before three.
func (c *ClientConn) chainStreamClient() grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		n := len(c.streamInterceptors)

		chainer := func(currentInter grpc.StreamClientInterceptor, currentStreamer grpc.Streamer) grpc.Streamer {
			return func(currentCtx context.Context, currentDesc *grpc.StreamDesc, currentConn *grpc.ClientConn, currentMethod string, currentOpts ...grpc.CallOption) (grpc.ClientStream, error) {
				return currentInter(currentCtx, currentDesc, currentConn, currentMethod, currentStreamer, currentOpts...)
			}
		}

		chainedStreamer := streamer
		for i := n - 1; i >= 0; i-- {
			chainedStreamer = chainer(c.streamInterceptors[i], chainedStreamer)
		}

		return chainedStreamer(ctx, desc, cc, method, opts...)
	}
}
