package tracing

import (
	"context"

	"github.com/tencentyun/tsf-go/util"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/stats"
)

type ClientHandler struct {
}

// HandleConn exists to satisfy gRPC stats.Handler.
func (c *ClientHandler) HandleConn(ctx context.Context, cs stats.ConnStats) {
}

// TagConn exists to satisfy gRPC stats.Handler.
func (c *ClientHandler) TagConn(ctx context.Context, cti *stats.ConnTagInfo) context.Context {
	if span := trace.SpanFromContext(ctx); span.SpanContext().HasTraceID() {
		remoteIP, remotePort := util.ParseAddr(cti.RemoteAddr.String())
		span.SetAttributes(attribute.String("peer.ip", remoteIP))
		span.SetAttributes(attribute.Int64("peer.port", int64(remotePort)))
	}
	return ctx
}

// HandleRPC implements per-RPC tracing and stats instrumentation.
func (c *ClientHandler) HandleRPC(ctx context.Context, rs stats.RPCStats) {

}

// TagRPC implements per-RPC context management.
func (c *ClientHandler) TagRPC(ctx context.Context, rti *stats.RPCTagInfo) context.Context {
	return ctx
}
