package trace

import (
	"context"
	"encoding/json"

	"github.com/tencentyun/tsf-go/pkg/log"
	"go.uber.org/zap"

	"github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/reporter"
)

var (
	_ reporter.Reporter = &tsfReporter{}
)

type Span struct {
	model.SpanContext
	Kind           model.Kind         `json:"kind,omitempty"`
	Name           string             `json:"name,omitempty"`
	Timestamp      int64              `json:"timestamp,omitempty"`
	Duration       int64              `json:"duration,omitempty"`
	Shared         bool               `json:"shared,omitempty"`
	LocalEndpoint  *model.Endpoint    `json:"localEndpoint,omitempty"`
	RemoteEndpoint *model.Endpoint    `json:"remoteEndpoint,omitempty"`
	Annotations    []model.Annotation `json:"annotations,omitempty"`
	Tags           map[string]string  `json:"tags,omitempty"`
}

type tsfReporter struct {
	logger *zap.Logger
}

// Send Span data to the reporter
func (r *tsfReporter) Send(s model.SpanModel) {
	span := Span{
		SpanContext:    s.SpanContext,
		Name:           s.Name,
		Kind:           s.Kind,
		Timestamp:      s.Timestamp.UnixNano() / 1000,
		Duration:       int64(s.Duration) / 1000,
		LocalEndpoint:  s.LocalEndpoint,
		RemoteEndpoint: s.RemoteEndpoint,
		Annotations:    s.Annotations,
		Tags:           s.Tags,
		Shared:         s.Shared,
	}
	content, err := json.Marshal(span)
	if err != nil {
		log.Error(context.Background(), "tsfReporter Marshal failed!", zap.Any("span", span))
		return
	}
	r.logger.Info(string(content))
}

// Close the reporter
func (r *tsfReporter) Close() error {
	return nil
}
