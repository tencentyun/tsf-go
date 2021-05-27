package main

import (
	"context"
	"fmt"
	"os"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	pb "github.com/tencentyun/tsf-go/examples/helloworld/proto"
	"github.com/tencentyun/tsf-go/naming/consul"
	"github.com/tencentyun/tsf-go/pkg/sys/env"
)

// server is used to implement helloworld.GreeterServer.
type server struct {
	pb.UnimplementedGreeterServer
}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	return &pb.HelloReply{Message: fmt.Sprintf("Welcome %+v!", in.Name)}, nil
}

func main() {
	logger := log.NewStdLogger(os.Stdout)
	log := log.NewHelper(logger)

	grpcSrv := grpc.NewServer(
		grpc.Address(":9000"),
		grpc.Middleware(
			recovery.Recovery(),
			logging.Server(logger),
		),
	)
	s := &server{}
	pb.RegisterGreeterServer(grpcSrv, s)

	httpSrv := http.NewServer(http.Address(":8000"))
	httpSrv.HandlePrefix("/", pb.NewGreeterHandler(s,
		http.Middleware(
			recovery.Recovery(),
		)),
	)

	app := kratos.New(
		kratos.Name("helloworld"),
		kratos.Server(
			grpcSrv,
			httpSrv,
		),
		kratos.Registrar(consul.DefaultConsul()),
		kratos.ID(env.InstanceId()),
	)

	if err := app.Run(); err != nil {
		log.Errorf("app run failed:%v", err)
	}
}
