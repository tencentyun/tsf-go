package server

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strconv"

	"github.com/tencentyun/tsf-go/pkg/errCode"
	"github.com/tencentyun/tsf-go/pkg/grpc/status"
	tsfHttp "github.com/tencentyun/tsf-go/pkg/http"
	"github.com/tencentyun/tsf-go/pkg/internal/env"
	"github.com/tencentyun/tsf-go/pkg/internal/monitor"
	"github.com/tencentyun/tsf-go/pkg/log"
	"github.com/tencentyun/tsf-go/pkg/meta"
	"github.com/tencentyun/tsf-go/pkg/util"

	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

func (s *Server) getStat(method string) *monitor.Stat {
	return monitor.NewStat(monitor.CategoryMS, monitor.KindServer, &monitor.Endpoint{ServiceName: s.conf.ServerName, InterfaceName: method, Path: method, Method: "POST"}, nil)
}

func (s *Server) handle(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
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
				v := vals[0]
				var tsfMeta tsfHttp.Metadata
				err := json.Unmarshal([]byte(v), &tsfMeta)
				if err != nil {
					log.L().Info(ctx, "grpc http parse header TSF-Metadata failed!", zap.String("meta", v), zap.Error(err))
				}
				sysPairs = append(sysPairs, meta.SysPair{Key: meta.SourceKey(meta.ApplicationID), Value: tsfMeta.ApplicationID})
				sysPairs = append(sysPairs, meta.SysPair{Key: meta.SourceKey(meta.ApplicationVersion), Value: tsfMeta.ApplicationVersion})
				sysPairs = append(sysPairs, meta.SysPair{Key: meta.SourceKey(meta.ServiceName), Value: tsfMeta.ServiceName})
				sysPairs = append(sysPairs, meta.SysPair{Key: meta.SourceKey(meta.GroupID), Value: tsfMeta.GroupID})
				sysPairs = append(sysPairs, meta.SysPair{Key: meta.SourceKey(meta.ConnnectionIP), Value: tsfMeta.LocalIP})
				sysPairs = append(sysPairs, meta.SysPair{Key: meta.SourceKey(meta.Namespace), Value: tsfMeta.NamespaceID})
			} else if key == "tsf-tags" {
				var tags []map[string]interface{} = make([]map[string]interface{}, 0)
				err = json.Unmarshal([]byte(vals[0]), &tags)
				if err != nil {
					log.L().Info(ctx, "grpc http parse header TSF-Tags failed!", zap.String("tags", vals[0]), zap.Error(err))
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
	api := info.FullMethod

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
	stat := s.getStat(api)
	span := zipkin.SpanFromContext(ctx)
	span.Tag("http.method", "POST")
	span.Tag("localInterface", api)
	span.Tag("http.path", api)
	span.SetRemoteEndpoint(remoteEndpointFromContext(ctx))
	defer func() {
		if rerr := recover(); rerr != nil {
			const size = 64 << 10
			buf := make([]byte, size)
			rs := runtime.Stack(buf, false)
			if rs > size {
				rs = size
			}
			buf = buf[:rs]
			pl := fmt.Sprintf("grpc server panic: %v\n%v\n%s\n", req, rerr, buf)
			fmt.Fprintf(os.Stderr, pl)
			log.L().Error(ctx, pl)
			err = errCode.Internal
		}

		var code = 200
		if err != nil {
			if ec, ok := err.(errCode.ErrCode); ok {
				code = ec.Code()
			} else {
				code = 500
			}
			span.Tag("exception", err.Error())
			err = status.ToGrpcStatus(err)
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
