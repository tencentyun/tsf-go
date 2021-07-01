package tracing

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v2/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
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
	sampleRatio float64
	exporter    tracesdk.SpanExporter
	r           *resource.Resource
}

// WithTracerExporter with tracer exporter.
func WithTracerExporter(exporter tracesdk.SpanExporter) Option {
	return func(opts *options) {
		opts.exporter = exporter
	}
}

// WithResource with tracer resource.
func WithResource(r *resource.Resource) Option {
	return func(opts *options) {
		opts.r = r
	}
}

// WithSampleRatio samples a given fraction of traces. Fractions >= 1 will
// always sample. Fractions < 0 are treated as zero. To respect the
// parent trace's `SampledFlag`, the `TraceIDRatioBased` sampler should be used
// as a delegate of a `Parent` sampler.
func WithSampleRatio(sampleRatio float64) Option {
	return func(opts *options) {
		opts.sampleRatio = sampleRatio
	}
}

// SetProvider set otel global provider
func SetProvider(opts ...Option) {
	options := options{
		sampleRatio: 0.1,
		exporter:    &Exporter{defaultLogger},
	}
	for _, o := range opts {
		o(&options)
	}
	tp := tracerProvider(options.sampleRatio, options.exporter, options.r)
	otel.SetTracerProvider(tp)
}

func init() {
	SetProvider()
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.Baggage{}, propagator{}))
}

// Tracer is otel span tracer
type Tracer struct {
	tracer trace.Tracer
	kind   trace.SpanKind
}

// NewTracer create tracer instance
func NewTracer(kind trace.SpanKind) (*Tracer, error) {
	switch kind {
	case trace.SpanKindClient:
		return &Tracer{tracer: otel.Tracer("CLIENT"), kind: kind}, nil
	case trace.SpanKindServer:
		return &Tracer{tracer: otel.Tracer("SERVER"), kind: kind}, nil
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
func tracerProvider(ratio float64, exporter tracesdk.SpanExporter, r *resource.Resource) *tracesdk.TracerProvider {
	opts := []tracesdk.TracerProviderOption{
		// 默认采样率10%
		tracesdk.WithSampler(tracesdk.ParentBased(tracesdk.TraceIDRatioBased(ratio))),
		// Always be sure to batch in production.
		tracesdk.WithBatcher(exporter),
	}
	if r != nil {
		opts = append(opts, tracesdk.WithResource(r))
	}
	tp := tracesdk.NewTracerProvider(opts...)
	return tp
}
