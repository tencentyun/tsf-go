package tracing

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/tencentyun/tsf-go/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// Annotation associates an event that explains latency with a timestamp.
type Annotation struct {
	Timestamp int64
	Value     string
}

// Endpoint holds the network context of a node in the service graph.
type Endpoint struct {
	ServiceName string `json:"serviceName,omitempty"`
	IPv4        string `json:"ipv4,omitempty"`
	IPv6        string `json:"ipv6,omitempty"`
	Port        uint16 `json:"port,omitempty"`
}

// Span is zipkin span model
type Span struct {
	TraceID  string `json:"traceId"`
	ID       string `json:"id"`
	ParentID string `json:"parentId,omitempty"`
	Debug    bool   `json:"debug,omitempty"`

	Kind           string            `json:"kind,omitempty"`
	Name           string            `json:"name,omitempty"`
	Timestamp      int64             `json:"timestamp,omitempty"`
	Duration       int64             `json:"duration,omitempty"`
	Shared         bool              `json:"shared,omitempty"`
	LocalEndpoint  *Endpoint         `json:"localEndpoint,omitempty"`
	RemoteEndpoint *Endpoint         `json:"remoteEndpoint,omitempty"`
	Annotations    []Annotation      `json:"annotations,omitempty"`
	Tags           map[string]string `json:"tags,omitempty"`
}

// Option is tracing option.
type Option func(*options)

type options struct {
	TracerProvider trace.TracerProvider
	Propagators    propagation.TextMapPropagator
}

// WithPropagators with tracer proagators.
func WithPropagators(propagators propagation.TextMapPropagator) Option {
	return func(opts *options) {
		opts.Propagators = propagators
	}
}

// WithTracerProvider with tracer privoder.
func WithTracerProvider(provider trace.TracerProvider) Option {
	return func(opts *options) {
		opts.TracerProvider = provider
	}
}

// Tracer is otel span tracer
type Tracer struct {
	tracer trace.Tracer
	kind   trace.SpanKind
}

// NewTracer create tracer instance
func NewTracer(kind trace.SpanKind, opts ...Option) (*Tracer, error) {
	tp, err := tracerProvider()
	if err != nil {
		log.DefaultLog.Errorf("new tsf tracer failed!err:=%v", err)
		return nil, err
	}
	options := options{
		TracerProvider: tp,
		Propagators:    propagation.NewCompositeTextMapPropagator(propagation.Baggage{}, propagation.TraceContext{}),
	}
	for _, o := range opts {
		o(&options)
	}
	otel.SetTracerProvider(options.TracerProvider)
	otel.SetTextMapPropagator(options.Propagators)
	switch kind {
	case trace.SpanKindClient:
		return &Tracer{tracer: otel.Tracer("client"), kind: kind}, nil
	case trace.SpanKindServer:
		return &Tracer{tracer: otel.Tracer("server"), kind: kind}, nil
	default:
		return nil, fmt.Errorf("unsupported span kind: %v", kind)
	}
}

// Start start tracing span
func (t *Tracer) Start(ctx context.Context, component string, operation string, carrier propagation.TextMapCarrier) (context.Context, trace.Span) {
	if t.kind == trace.SpanKindServer {
		ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)
	}
	ctx, span := t.tracer.Start(ctx,
		operation,
		trace.WithSpanKind(t.kind),
	)
	if t.kind == trace.SpanKindClient {
		otel.GetTextMapPropagator().Inject(ctx, carrier)
	}
	return ctx, span
}

// End finish tracing span
func (t *Tracer) End(ctx context.Context, span trace.Span, err error) {
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

// Get trace provider
func tracerProvider() (*tracesdk.TracerProvider, error) {
	exp := &Exporter{defaultLogger}
	tp := tracesdk.NewTracerProvider(
		tracesdk.WithSampler(tracesdk.AlwaysSample()),
		// Always be sure to batch in production.
		tracesdk.WithBatcher(exp),
		// Record information about this application in an Resource.
		tracesdk.WithResource(resource.NewWithAttributes(
			attribute.Int64("ID", 1),
		)),
	)
	return tp, nil
}

type Exporter struct {
	logger *zap.Logger
}

func (e *Exporter) ExportSpans(ctx context.Context, ss []*tracesdk.SpanSnapshot) error {
	for _, s := range ss {
		attrs := make(map[string]attribute.Value, 0)
		for _, attr := range s.Attributes {
			attrs[string(attr.Key)] = attr.Value
		}
		span := Span{
			TraceID:   s.SpanContext.TraceID().String(),
			ID:        s.SpanContext.SpanID().String(),
			Kind:      s.SpanKind.String(),
			Name:      s.Name,
			Timestamp: s.StartTime.UnixNano() / 1000,
			Duration:  int64(s.EndTime.Sub(s.StartTime)) / 1000,
			LocalEndpoint: &Endpoint{
				ServiceName: attrs["local.service"].AsString(),
				IPv4:        attrs["local.ip"].AsString(),
				Port:        uint16(attrs["local.port"].AsInt64()),
			},
			RemoteEndpoint: &Endpoint{
				ServiceName: attrs["peer.service"].AsString(),
				IPv4:        attrs["peer.ip"].AsString(),
				Port:        uint16(attrs["peer.port"].AsInt64()),
			},
			Tags: map[string]string{},
		}
		if s.Parent.HasSpanID() {
			span.ParentID = s.Parent.SpanID().String()
		}
		if v, ok := attrs["annotations"]; ok {
			span.Annotations = append(span.Annotations, Annotation{Timestamp: s.StartTime.UnixNano() / 1000, Value: v.AsString()})
		}
		delete(attrs, "local.service")
		delete(attrs, "local.ip")
		delete(attrs, "local.port")
		delete(attrs, "peer.service")
		delete(attrs, "peer.ip")
		delete(attrs, "peer.port")
		for k, v := range attrs {
			if v.Type() == attribute.STRING {
				span.Tags[k] = v.AsString()
			} else if v.Type() == attribute.INT64 {
				span.Tags[k] = strconv.FormatInt(v.AsInt64(), 10)
			}
		}
		content, err := json.Marshal(span)
		if err != nil {
			log.DefaultLog.Errorf("tsfReporter Marshal failed!%v %s", span, err)
			continue
		}
		e.logger.Info(string(content))
	}
	return nil
}

func (e *Exporter) Shutdown(ctx context.Context) error {
	return nil
}
