package tracing

import (
	"context"
	"strings"

	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// Default B3 Header keys
const (
	TraceID      = "x-b3-traceid"
	SpanID       = "x-b3-spanid"
	ParentSpanID = "x-b3-parentspanid"
	Sampled      = "x-b3-sampled"
	Flags        = "x-b3-flags"
	Context      = "b3"
)

type propagator struct {
}

// Inject set cross-cutting concerns from the Context into the carrier.
func (propagator) Inject(ctx context.Context, c propagation.TextMapCarrier) {
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return
	}
	c.Set(TraceID, sc.TraceID().String())
	c.Set(SpanID, sc.SpanID().String())
	c.Set(ParentSpanID, sc.SpanID().String())
	if sc.TraceFlags().IsSampled() {
		c.Set(Sampled, "1")
	} else {
		c.Set(Sampled, "0")
	}
}

// DO NOT CHANGE: any modification will not be backwards compatible and
// must never be done outside of a new major release.

// Extract reads cross-cutting concerns from the carrier into a Context.
func (propagator) Extract(ctx context.Context, c propagation.TextMapCarrier) context.Context {
	var err error
	var scc trace.SpanContextConfig
	scc.TraceID, err = trace.TraceIDFromHex(c.Get(TraceID))
	if err != nil {
		return ctx
	}
	scc.SpanID, err = trace.SpanIDFromHex(c.Get(SpanID))
	if err != nil {
		return ctx
	}

	switch strings.ToLower(c.Get(Sampled)) {
	case "0", "false":
	case "1", "true":
		scc.TraceFlags = trace.FlagsSampled
	case "":
		// sc.Sampled = nil
	default:
		return ctx
	}
	scc.Remote = true
	sc := trace.NewSpanContext(scc)
	if !sc.IsValid() {
		return ctx
	}
	return trace.ContextWithRemoteSpanContext(ctx, sc)
}

// DO NOT CHANGE: any modification will not be backwards compatible and
// must never be done outside of a new major release.

// Fields returns the keys who's values are set with Inject.
func (propagator) Fields() []string {
	return []string{TraceID, SpanID, ParentSpanID, Sampled}
}
