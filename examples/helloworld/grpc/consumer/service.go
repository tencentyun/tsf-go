package main

import (
	"context"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	pb "github.com/tencentyun/tsf-go/examples/helloworld/proto"
	"github.com/tencentyun/tsf-go/log"
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
	logger := log.DefaultLogger
	log := log.NewHelper(logger)

	clientOpts := []grpc.ClientOption{grpc.WithEndpoint("discovery:///provider-grpc")}
	clientOpts = append(clientOpts, tsf.ClientGrpcOptions()...)
	conn, err := grpc.DialInsecure(context.Background(), clientOpts...)
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
			recovery.Recovery(),
			logging.Server(logger),
			tsf.ServerMiddleware(),
		),
	)
	pb.RegisterGreeterServer(grpcSrv, s)

	opts := []kratos.Option{kratos.Name("consumer-grpc"), kratos.Server(grpcSrv)}
	opts = append(opts, tsf.DefaultOptions()...)
	app := kratos.New(opts...)

	if err := app.Run(); err != nil {
		log.Errorf("app run failed:%v", err)
	}
}
