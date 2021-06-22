package server

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"
	"github.com/tencentyun/tsf-go/log"
	tsfHttp "github.com/tencentyun/tsf-go/pkg/http"
	"github.com/tencentyun/tsf-go/pkg/meta"
	"github.com/tencentyun/tsf-go/pkg/sys/env"
	"github.com/tencentyun/tsf-go/pkg/sys/monitor"
	"github.com/tencentyun/tsf-go/pkg/util"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

func (s *Server) startContext(ctx context.Context, api string) context.Context {
	// add system metadata into ctx
	var sysPairs []meta.SysPair
	var userPairs []meta.UserPair
	if gmd, ok := metadata.FromIncomingContext(ctx); ok {
		for key, vals := range gmd {
			if vals[0] == "" {
				continue
			}
			if meta.IsIncomming(key) {
				sysPairs = append(sysPairs, meta.SysPair{Key: meta.SourceKey(key), Value: vals[0]})
			} else if meta.IsUserKey(key) {
				userPairs = append(userPairs, meta.UserPair{Key: meta.GetUserKey(key), Value: vals[0]})
			} else if meta.IsLinkKey(key) {
				sysPairs = append(sysPairs, meta.SysPair{Key: key, Value: vals[0]})
			} else if key == "tsf-metadata" {
				var tsfMeta tsfHttp.Metadata
				e := json.Unmarshal([]byte(vals[0]), &tsfMeta)
				if e != nil {
					v, e := url.QueryUnescape(vals[0])
					if e == nil {
						e = json.Unmarshal([]byte(v), &tsfMeta)
					} else {
						log.DefaultLog.WithContext(ctx).Infow("msg", "grpc http parse header TSF-Metadata failed!", "meta", v, "err", e)
					}
				}
				sysPairs = append(sysPairs, meta.SysPair{Key: meta.SourceKey(meta.ApplicationID), Value: tsfMeta.ApplicationID})
				sysPairs = append(sysPairs, meta.SysPair{Key: meta.SourceKey(meta.ApplicationVersion), Value: tsfMeta.ApplicationVersion})
				sysPairs = append(sysPairs, meta.SysPair{Key: meta.SourceKey(meta.ServiceName), Value: tsfMeta.ServiceName})
				sysPairs = append(sysPairs, meta.SysPair{Key: meta.SourceKey(meta.GroupID), Value: tsfMeta.GroupID})
				sysPairs = append(sysPairs, meta.SysPair{Key: meta.SourceKey(meta.ConnnectionIP), Value: tsfMeta.LocalIP})
				sysPairs = append(sysPairs, meta.SysPair{Key: meta.SourceKey(meta.Namespace), Value: tsfMeta.NamespaceID})
			} else if key == "tsf-tags" {
				var tags []map[string]interface{} = make([]map[string]interface{}, 0)
				e := json.Unmarshal([]byte(vals[0]), &tags)
				if e != nil {
					v, e := url.QueryUnescape(vals[0])
					if e == nil {
						e = json.Unmarshal([]byte(v), &tags)
					} else {
						log.DefaultLog.WithContext(ctx).Info("mg", "grpc http parse header TSF-Tags failed!", "tags", vals[0], "err", e)
					}
				}
				for _, tag := range tags {
					for k, v := range tag {
						if value, ok := v.(string); ok {
							userPairs = append(userPairs, meta.UserPair{Key: k, Value: value})
						}
					}
				}
			}
		}
	}
	if pr, ok := peer.FromContext(ctx); ok {
		sysPairs = append(sysPairs, meta.SysPair{Key: meta.SourceKey(meta.ConnnectionIP), Value: util.IPFromAddr(pr.Addr)})
	}

	sysPairs = append(sysPairs, meta.SysPair{Key: meta.ServiceName, Value: s.conf.ServerName})
	sysPairs = append(sysPairs, meta.SysPair{Key: meta.Namespace, Value: env.NamespaceID()})
	sysPairs = append(sysPairs, meta.SysPair{Key: meta.Interface, Value: api})
	sysPairs = append(sysPairs, meta.SysPair{Key: meta.Tracer, Value: s.tracer})
	sysPairs = append(sysPairs, meta.SysPair{Key: meta.GroupID, Value: env.GroupID()})
	sysPairs = append(sysPairs, meta.SysPair{Key: meta.ApplicationID, Value: env.ApplicationID()})
	sysPairs = append(sysPairs, meta.SysPair{Key: meta.ApplicationVersion, Value: env.ProgVersion()})
	sysPairs = append(sysPairs, meta.SysPair{Key: meta.ConnnectionIP, Value: env.LocalIP()})
	ctx = meta.WithSys(ctx, sysPairs...)
	ctx = meta.WithUser(ctx, userPairs...)
	return ctx
}

func remoteEndpointFromContext(ctx context.Context) *model.Endpoint {
	remoteAddr := ""

	p, ok := peer.FromContext(ctx)
	if ok {
		remoteAddr = p.Addr.String()
	}
	var name string
	name, _ = meta.Sys(ctx, meta.SourceKey(meta.ServiceName)).(string)
	ep, _ := zipkin.NewEndpoint(name, remoteAddr)
	return ep
}

func (s *Server) getStat(method string) *monitor.Stat {
	return monitor.NewStat(monitor.CategoryMS, monitor.KindServer, &monitor.Endpoint{ServiceName: s.conf.ServerName, InterfaceName: method, Path: method, Method: "POST"}, nil)
}

func (s *Server) handle(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	api := info.FullMethod
	ctx = s.startContext(ctx, api)
	stat := s.getStat(api)
	span := zipkin.SpanFromContext(ctx)
	span.Tag("http.method", "POST")
	span.Tag("localInterface", api)
	span.Tag("http.path", api)
	span.SetRemoteEndpoint(remoteEndpointFromContext(ctx))
	defer func() {
		var code = 200
		if err != nil {
			code = errors.FromError(err).StatusCode()
			span.Tag("exception", err.Error())
		}
		span.Tag("resultStatus", strconv.FormatInt(int64(code), 10))
		stat.Record(code)
	}()

	// 鉴权
	err = s.authen.Verify(ctx, info.FullMethod)
	if err != nil {
		return
	}

	resp, err = handler(ctx, req)
	return
}

// StreamServerInterceptor returns a new unary server interceptors that performs per-request auth.
func (s *Server) handleStream(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
	api := info.FullMethod
	ctx := s.startContext(stream.Context(), api)
	stat := s.getStat(api)
	span := zipkin.SpanFromContext(ctx)
	span.Tag("http.method", "POST")
	span.Tag("localInterface", api)
	span.Tag("http.path", api)
	span.SetRemoteEndpoint(remoteEndpointFromContext(ctx))
	defer func() {
		var code = 200
		if err != nil {
			code = errors.FromError(err).StatusCode()
			span.Tag("exception", err.Error())
		}
		span.Tag("resultStatus", strconv.FormatInt(int64(code), 10))
		stat.Record(code)
	}()
	// 鉴权
	err = s.authen.Verify(ctx, info.FullMethod)
	if err != nil {
		return
	}
	wrapped := WrapServerStream(stream)
	wrapped.WrappedContext = ctx
	err = handler(srv, wrapped)
	return
}
