package tracing

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/tencentyun/tsf-go/log"

	"go.opentelemetry.io/otel/attribute"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
)

type Exporter struct {
	logger *zap.Logger
}

func (e *Exporter) ExportSpans(ctx context.Context, ss []tracesdk.ReadOnlySpan) error {
	for _, s := range ss {
		attrs := make(map[string]attribute.Value, 0)
		for _, attr := range s.Attributes() {
			attrs[string(attr.Key)] = attr.Value
		}
		span := Span{
			TraceID:   s.SpanContext().TraceID().String(),
			ID:        s.SpanContext().SpanID().String(),
			Kind:      strings.ToUpper(s.SpanKind().String()),
			Name:      s.Name(),
			Timestamp: s.StartTime().UnixNano() / 1000,
			Duration:  int64(s.EndTime().Sub(s.StartTime())) / 1000,
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
		if s.Parent().HasSpanID() {
			span.ParentID = s.Parent().SpanID().String()
		}
		if v, ok := attrs["annotations"]; ok {
			span.Annotations = append(span.Annotations, Annotation{Timestamp: s.StartTime().UnixNano() / 1000, Value: v.AsString()})
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
