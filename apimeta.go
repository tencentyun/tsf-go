package tsf

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"time"

	"github.com/go-kratos/kratos/v2/api/metadata"
	"github.com/go-kratos/swagger-api/openapiv2"
)

func genAPIMeta(md map[string]string, srv *openapiv2.Service, serviceName string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	var httpAPIMeta string
	var rpcAPIMeta string
	if serviceName != "" {
		httpAPIMeta, _ = srv.GetServiceOpenAPI(ctx, &metadata.GetServiceDescRequest{Name: serviceName}, false)
		rpcAPIMeta, _ = srv.GetServiceOpenAPI(ctx, &metadata.GetServiceDescRequest{Name: serviceName}, true)
	} else {
		reply, err := srv.ListServices(ctx, &metadata.ListServicesRequest{})
		if err == nil {
			for _, service := range reply.Services {
				if service != "grpc.health.v1.Health" && service != "grpc.reflection.v1alpha.ServerReflection" && service != "kratos.api.Metadata" {
					httpAPIMeta, _ = srv.GetServiceOpenAPI(ctx, &metadata.GetServiceDescRequest{Name: service}, false)
					rpcAPIMeta, _ = srv.GetServiceOpenAPI(ctx, &metadata.GetServiceDescRequest{Name: service}, true)
					break
				}
			}
		}
	}
	if httpAPIMeta != "" {
		var buf bytes.Buffer
		zw := gzip.NewWriter(&buf)
		_, err := zw.Write([]byte(httpAPIMeta))
		if err == nil {
			err = zw.Close()
			if err == nil {
				res := base64.StdEncoding.EncodeToString(buf.Bytes())
				md["TSF_API_METAS_HTTP"] = res
			}
		}
	}
	if rpcAPIMeta != "" {
		var buf bytes.Buffer
		zw := gzip.NewWriter(&buf)
		_, err := zw.Write([]byte(rpcAPIMeta))
		if err == nil {
			err = zw.Close()
			if err == nil {
				res := base64.StdEncoding.EncodeToString(buf.Bytes())
				md["TSF_API_METAS_GRPC"] = res
			}
		}
	}
}
