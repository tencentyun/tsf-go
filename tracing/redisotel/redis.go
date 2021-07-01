package redisotel

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-redis/redis/extra/rediscmd"
	"github.com/go-redis/redis/v8"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func New(address string) RedisHook {
	tracer := otel.Tracer("redis")

	remoteIP, remotePort := parseAddr(address)
	return RedisHook{ip: remoteIP, port: remotePort, tracer: tracer}
}

type RedisHook struct {
	ip     string
	port   uint16
	tracer trace.Tracer
}

var _ redis.Hook = RedisHook{}

func (rh RedisHook) BeforeProcess(ctx context.Context, cmd redis.Cmder) (context.Context, error) {
	if !trace.SpanFromContext(ctx).IsRecording() {
		fmt.Println("redis not recoding!")

		return ctx, nil
	}
	fmt.Println("redis in!")

	ctx, span := rh.tracer.Start(ctx, cmd.FullName(), trace.WithSpanKind(trace.SpanKindClient))
	span.SetAttributes(attribute.String("peer.ip", rh.ip))
	span.SetAttributes(attribute.Int64("peer.port", int64(rh.port)))
	span.SetAttributes(attribute.String("peer.service", "redis-server"))
	span.SetAttributes(attribute.String("remoteComponent", "REDIS"))

	span.SetAttributes(
		attribute.String("db.system", "redis"),
		attribute.String("db.statement", rediscmd.CmdString(cmd)),
	)

	return ctx, nil
}

func (RedisHook) AfterProcess(ctx context.Context, cmd redis.Cmder) error {
	recordError(ctx, cmd.Err())
	return nil
}

func (rh RedisHook) BeforeProcessPipeline(ctx context.Context, cmds []redis.Cmder) (context.Context, error) {
	if !trace.SpanFromContext(ctx).IsRecording() {
		return ctx, nil
	}

	summary, cmdsString := rediscmd.CmdsString(cmds)

	ctx, span := rh.tracer.Start(ctx, "pipeline "+summary)
	span.SetAttributes(attribute.String("peer.ip", rh.ip))
	span.SetAttributes(attribute.Int64("peer.port", int64(rh.port)))
	span.SetAttributes(attribute.String("peer.service", "redis-server"))
	span.SetAttributes(attribute.String("remoteComponent", "REDIS"))

	span.SetAttributes(
		attribute.String("db.system", "redis"),
		attribute.Int("db.redis.num_cmd", len(cmds)),
		attribute.String("db.statement", cmdsString),
	)

	return ctx, nil
}

func (RedisHook) AfterProcessPipeline(ctx context.Context, cmds []redis.Cmder) error {
	recordError(ctx, cmds[0].Err())
	return nil
}

func recordError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	var code = 200
	if err != nil {
		code = errors.FromError(err).StatusCode()
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		span.SetAttributes(attribute.String("exception", err.Error()))
	} else {
		span.SetStatus(codes.Ok, "OK")
	}

	span.SetAttributes(
		attribute.Int("resultStatus", code),
	)
	span.End()
}

func parseAddr(addr string) (ip string, port uint16) {
	strs := strings.Split(addr, ":")
	if len(strs) > 0 {
		ip = strs[0]
	}
	if len(strs) > 1 {
		uport, _ := strconv.ParseUint(strs[1], 10, 16)
		port = uint16(uport)
	}
	return
}
