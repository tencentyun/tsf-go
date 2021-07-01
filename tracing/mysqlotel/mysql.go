package mysqlotel

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	tsf "github.com/tencentyun/tsf-go"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/luna-duclos/instrumentedsql"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Tracer struct {
	tracer trace.Tracer
	ip     string
	port   uint16
}

// NewTracer returns a tracer that will fetch spans using opentracing's SpanFromContext function
// if traceOrphans is set to true, then spans with no parent will be traced anyway, if false, they will not be.
func NewTracer(address string) instrumentedsql.Tracer {
	tracer := otel.Tracer("mysql")
	remoteIP, remotePort := parseAddr(address)

	return Tracer{tracer: tracer, ip: remoteIP, port: remotePort}
}

// GetSpan returns a span
func (t Tracer) GetSpan(ctx context.Context) instrumentedsql.Span {
	return Span{ctx: ctx, t: &t}
}

type Span struct {
	t    *Tracer
	ctx  context.Context
	span trace.Span
}

func (s Span) NewChild(name string) instrumentedsql.Span {
	fmt.Println("name:", name, "span:", s.span)
	if !trace.SpanFromContext(s.ctx).IsRecording() {
		return Span{}
	}
	localEndpoint := tsf.LocalEndpoint(s.ctx)

	ctx, span := s.t.tracer.Start(s.ctx, name, trace.WithSpanKind(trace.SpanKindClient))

	span.SetAttributes(attribute.String("local.ip", localEndpoint.IP))
	span.SetAttributes(attribute.Int64("local.port", int64(localEndpoint.Port)))
	span.SetAttributes(attribute.String("local.service", localEndpoint.Service))

	span.SetAttributes(attribute.String("peer.ip", s.t.ip))
	span.SetAttributes(attribute.Int64("peer.port", int64(s.t.port)))
	span.SetAttributes(attribute.String("peer.service", "mysql-server"))
	span.SetAttributes(attribute.String("remoteComponent", "MYSQL"))

	return Span{ctx: ctx, span: span}
}

func (s Span) SetLabel(k, v string) {
	if s.span == nil {
		return
	}
	s.span.SetAttributes(attribute.String(k, v))
}

func (s Span) SetError(err error) {
	if s.span == nil {
		return
	}
	var code = 200
	if err != nil {
		code = errors.FromError(err).StatusCode()
		s.span.RecordError(err)
		s.span.SetStatus(codes.Error, err.Error())
		s.span.SetAttributes(attribute.String("exception", err.Error()))
	} else {
		s.span.SetStatus(codes.Ok, "OK")
	}

	s.span.SetAttributes(
		attribute.Int("resultStatus", code),
	)
}

func (s Span) Finish() {
	if s.span == nil {
		return
	}
	s.span.End()
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
