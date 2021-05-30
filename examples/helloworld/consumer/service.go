package main

import (
	"context"
	"os"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	transhttp "github.com/go-kratos/kratos/v2/transport/http"
	pb "github.com/tencentyun/tsf-go/examples/helloworld/proto"
	"github.com/tencentyun/tsf-go/naming/consul"

	tsf "github.com/tencentyun/tsf-go"
)

// server is used to implement helloworld.GreeterServer.
type server struct {
	pb.UnimplementedGreeterServer

	client     pb.GreeterClient
	httpClient pb.GreeterHTTPClient
}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	if in.Name == "kratos_http" {
		return s.httpClient.SayHello(ctx, in)
	}
	return s.client.SayHello(ctx, in)
}

func newService(c *consul.Consul) {
	logger := log.NewStdLogger(os.Stdout)
	log := log.NewHelper(logger)

	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint("discovery:///provider-go"),
		grpc.WithDiscovery(c),
		grpc.WithMiddleware(
			tsf.GRPCClientMiddleware("provider-go"),
		),
		tsf.ClientGrpcOptions(),
	)
	if err != nil {
		log.Errorf("dial grpc err:%v", err)
		return
	}
	httpConn, err := transhttp.NewClient(
		context.Background(),
		transhttp.WithMiddleware(
			recovery.Recovery(),
		),
		transhttp.WithScheme("http"),
		transhttp.WithEndpoint("discovery:///provider-go"),
		transhttp.WithDiscovery(c),
	)
	if err != nil {
		log.Errorf("dial http err:%v", err)
		return
	}
	s := &server{
		client:     pb.NewGreeterClient(conn),
		httpClient: pb.NewGreeterHTTPClient(httpConn),
	}

	grpcSrv := grpc.NewServer(
		grpc.Address(":9090"),
		grpc.Middleware(
			logging.Server(logger),
			tsf.GRPCServerMiddleware("consumer-go", 9090),
		),
	)
	pb.RegisterGreeterServer(grpcSrv, s)

	httpSrv := http.NewServer(http.Address(":8080"))
	httpSrv.HandlePrefix("/", pb.NewGreeterHandler(s,
		http.Middleware(
			recovery.Recovery(),
			logging.Server(logger),
		)),
	)
	app := kratos.New(
		kratos.Name("consumer-go"),
		kratos.Server(
			grpcSrv,
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
