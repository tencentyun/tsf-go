package metrics

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"

	"github.com/google/gops/agent"
	"github.com/tencentyun/tsf-go/pkg/log"
	"github.com/tencentyun/tsf-go/pkg/sys/env"
	"go.uber.org/zap"
)

func StartAgent() {
	go startGops()
	go startPprof()
}

func startPprof() {
	if !env.DisableDisablePprof() {
		mux := http.NewServeMux()
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Symbol)

		addr := fmt.Sprintf(":%d", env.PprofPort())

		lis, err := net.Listen("tcp", addr)
		if err != nil {
			log.Errorf(context.Background(), "pprof server listen %s err: %v", addr, err)
			return
		}
		server := http.Server{
			Handler: mux,
			Addr:    addr,
		}
		log.Debug(context.Background(), "pprof http server start serve. To disable it,set tsf_disable_pprof=true", zap.String("addr", addr))
		if err = server.Serve(lis); err != nil {
			log.Errorf(context.Background(), "pprof server serve  err: %v", err)
			return
		}
	}
}

func startGops() {
	if !env.DisableDisableGops() {
		addr := fmt.Sprintf(":%d", env.GopsPort())
		log.Debug(context.Background(), "gops agent start serve.  To disable it,set tsf_disable_gops=true", zap.String("addr", addr))
		if err := agent.Listen(agent.Options{Addr: addr}); err != nil {
			log.Errorf(context.Background(), "gops agent.Listen %s err: %v", addr, err)
			return
		}
	}
}
