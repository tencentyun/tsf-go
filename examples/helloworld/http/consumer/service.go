package main

import (
	"context"
	"os"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/http"
	transhttp "github.com/go-kratos/kratos/v2/transport/http"
	pb "github.com/tencentyun/tsf-go/examples/helloworld/proto"
	"github.com/tencentyun/tsf-go/naming/consul"

	tsf "github.com/tencentyun/tsf-go"
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
	logger := log.NewStdLogger(os.Stdout)
	log := log.NewHelper(logger)

	httpConn, err := transhttp.NewClient(
		context.Background(),
		transhttp.WithMiddleware(
			recovery.Recovery(),
			tsf.ClientMiddleware("provider-http"),
		),
		transhttp.WithScheme("http"),
		transhttp.WithEndpoint("discovery:///provider-http"),
		transhttp.WithDiscovery(c),
		tsf.ClientHTTPOptions(),
	)
	if err != nil {
		log.Errorf("dial http err:%v", err)
		return
	}
	s := &server{
		httpClient: pb.NewGreeterHTTPClient(httpConn),
	}

	httpSrv := http.NewServer(http.Address("0.0.0.0:8080"))
	httpSrv.HandlePrefix("/", pb.NewGreeterHandler(s,
		http.Middleware(
			recovery.Recovery(),
			logging.Server(logger),
			tsf.ServerMiddleware("consumer-http", 8080),
		)),
	)
	app := kratos.New(
		kratos.Name("consumer-http"),
		kratos.Server(
			httpSrv,
		),
		tsf.Metadata(tsf.APIMeta(false)),
		tsf.ID(),
		tsf.Registrar(),
	)

	if err := app.Run(); err != nil {
		log.Errorf("app run failed:%v", err)
	}
}
