package main

import (
	"context"
	"os"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	pb "github.com/tencentyun/tsf-go/examples/helloworld/proto"
	"github.com/tencentyun/tsf-go/naming/consul"

	tsf "github.com/tencentyun/tsf-go"
)

// server is used to implement helloworld.GreeterServer.
type server struct {
	pb.UnimplementedGreeterServer

	client pb.GreeterClient
}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
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
			tsf.ClientMiddleware(),
		),
		tsf.ClientGrpcOptions(),
	)
	if err != nil {
		log.Errorf("dial grpc err:%v", err)
		return
	}

	s := &server{
		client: pb.NewGreeterClient(conn),
	}

	grpcSrv := grpc.NewServer(
		grpc.Address("0.0.0.0:9090"),
		grpc.Middleware(
			logging.Server(logger),
			tsf.ServerMiddleware(),
		),
	)
	pb.RegisterGreeterServer(grpcSrv, s)

	app := kratos.New(
		kratos.Name("consumer-go"),
		kratos.Server(
			grpcSrv,
		),
		tsf.Metadata(tsf.APIMeta(false)),
		tsf.ID(),
		tsf.Registrar(),
	)

	if err := app.Run(); err != nil {
		log.Errorf("app run failed:%v", err)
	}
}
