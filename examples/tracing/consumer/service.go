package main

import (
	"context"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/http"
	tsf "github.com/tencentyun/tsf-go"
	pb "github.com/tencentyun/tsf-go/examples/helloworld/proto"
	"github.com/tencentyun/tsf-go/log"
	"github.com/tencentyun/tsf-go/naming/consul"
)

// server is used to implement helloworld.GreeterServer.
type server struct {
	pb.UnimplementedGreeterServer

	httpClient pb.GreeterHTTPClient
}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	return s.httpClient.SayHello(ctx, in)
}

func newService(c *consul.Consul) {
	logger := log.DefaultLogger
	log := log.NewHelper(logger)

	clientOpts := []http.ClientOption{http.WithEndpoint("discovery:///provider-http")}
	clientOpts = append(clientOpts, tsf.ClientHTTPOptions()...)
	httpConn, err := http.NewClient(context.Background(), clientOpts...)
	if err != nil {
		log.Errorf("dial http err:%v", err)
		return
	}
	s := &server{
		httpClient: pb.NewGreeterHTTPClient(httpConn),
	}

	httpSrv := http.NewServer(
		http.Address("0.0.0.0:8080"),
		http.Middleware(
			recovery.Recovery(),
			// 将tracing采样率提升至100%
			tsf.ServerMiddleware(),
			logging.Server(logger),
		),
	)
	pb.RegisterGreeterHTTPServer(httpSrv, s)

	opts := []kratos.Option{kratos.Name("consumer-http"), kratos.Server(httpSrv)}
	opts = append(opts, tsf.AppOptions()...)
	app := kratos.New(opts...)

	if err := app.Run(); err != nil {
		log.Errorf("app run failed:%v", err)
	}
}
