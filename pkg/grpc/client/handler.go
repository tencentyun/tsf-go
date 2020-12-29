package client

import (
	"context"
	"fmt"
	"strconv"

	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/propagation/b3"
	"github.com/tencentyun/tsf-go/pkg/errCode"
	"github.com/tencentyun/tsf-go/pkg/grpc/status"
	"github.com/tencentyun/tsf-go/pkg/meta"
	"github.com/tencentyun/tsf-go/pkg/sys/env"
	"github.com/tencentyun/tsf-go/pkg/sys/monitor"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func (c *Conn) getStat(ctx context.Context, method string) *monitor.Stat {
	localService, ok := meta.Sys(ctx, meta.ServiceName).(string)
	if !ok {
		localService = env.ServiceName()
	}
	localMethod, ok := meta.Sys(ctx, meta.Interface).(string)
	if !ok {
		localMethod = "/defaultInterface"
	}
	return monitor.NewStat(monitor.CategoryMS, monitor.KindClient, &monitor.Endpoint{ServiceName: localService, InterfaceName: localMethod, Path: localMethod, Method: "POST"}, &monitor.Endpoint{ServiceName: c.remoteService.Name, InterfaceName: method})
}

func (c *Conn) startContext(ctx context.Context, api string) context.Context {
	// 注入远端服务名
	pairs := []meta.SysPair{
		{Key: meta.DestKey(meta.ServiceName), Value: c.remoteService.Name},
		{Key: meta.DestKey(meta.ServiceNamespace), Value: c.remoteService.Namespace},
	}

	// 注入自己的服务名
	serviceName := env.ServiceName()
	if res := meta.Sys(ctx, meta.ServiceName); res == nil {
		pairs = append(pairs, meta.SysPair{Key: meta.ServiceName, Value: serviceName})
	} else {
		serviceName = res.(string)
	}
	pairs = append(pairs, meta.SysPair{Key: meta.DestKey(meta.Interface), Value: api})
	if laneID := c.lane.GetLaneID(ctx); laneID != "" {
		pairs = append(pairs, meta.SysPair{Key: meta.LaneID, Value: laneID})
	}
	ctx = meta.WithSys(ctx, pairs...)

	gmd := metadata.MD{}
	meta.RangeUser(ctx, func(key string, value string) {
		gmd[meta.UserKey(key)] = []string{value}
	})
	meta.RangeSys(ctx, func(key string, value interface{}) {
		if meta.IsOutgoing(key) {
			if str, ok := value.(string); ok {
				gmd[key] = []string{str}
			} else if fmtStr, ok := value.(fmt.Stringer); ok {
				gmd[key] = []string{fmtStr.String()}
			}
		}
	})
	gmd[meta.GroupID] = []string{env.GroupID()}
	gmd[meta.ServiceNamespace] = []string{env.NamespaceID()}
	gmd[meta.ApplicationID] = []string{env.ApplicationID()}
	gmd[meta.ApplicationVersion] = []string{env.ProgVersion()}
	// merge with old matadata if exists
	if oldmd, ok := metadata.FromOutgoingContext(ctx); ok {
		gmd = metadata.Join(gmd, oldmd)
	}
	ctx = metadata.NewOutgoingContext(ctx, gmd)
	return ctx
}

func (c *Conn) handle(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) (err error) {
	api := method
	ctx = c.startContext(ctx, api)
	ctx = c.startSpan(ctx, api)
	stat := c.getStat(ctx, api)
	defer func() {
		var code = 200
		if err = status.FromGrpcStatus(err); err != nil {
			if ec, ok := err.(errCode.ErrCode); ok {
				code = ec.Code()
			} else {
				code = 500
			}
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

	err = invoker(ctx, method, req, reply, cc, opts...)
	return
}

func (c *Conn) handleStream(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (clientStream grpc.ClientStream, err error) {
	api := method
	ctx = c.startContext(ctx, api)
	ctx = c.startSpan(ctx, api)
	stat := c.getStat(ctx, api)
	defer func() {
		var code = 200
		if err = status.FromGrpcStatus(err); err != nil {
			if ec, ok := err.(errCode.ErrCode); ok {
				code = ec.Code()
			} else {
				code = 500
			}
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

	clientStream, err = streamer(ctx, desc, cc, method, opts...)
	return
}

func (c *Conn) startSpan(ctx context.Context, api string) context.Context {
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

	span.Tag("http.method", "POST")
	span.Tag("localInterface", api)
	span.Tag("http.path", api)
	if gmd, ok := metadata.FromOutgoingContext(ctx); ok {
		b3.InjectGRPC(&gmd)(span.Context())
		ctx = metadata.NewOutgoingContext(ctx, gmd)
	}
	return ctx
}
