package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/http"
	tsf "github.com/tencentyun/tsf-go"
	pb "github.com/tencentyun/tsf-go/examples/atom/proto"
	"github.com/tencentyun/tsf-go/log"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// server is used to implement helloworld.GreeterServer.
type server struct {
}

// SayHello implements helloworld.GreeterServer
func (s *server) Hello(ctx context.Context, in *emptypb.Empty) (*pb.HelloReply, error) {
	return &pb.HelloReply{Message: fmt.Sprintf("Hello!")}, nil
}

// SayHello implements helloworld.GreeterServer
func (s *server) Echo(ctx context.Context, in *pb.EchoRequest) (*pb.EchoReply, error) {
	return &pb.EchoReply{Param: in.Param, ApplicationName: "provider-demo"}, nil
}

// SayHello implements helloworld.GreeterServer
func (s *server) EchoError(ctx context.Context, in *pb.EchoRequest) (*pb.EchoReply, error) {
	return nil, errors.ServiceUnavailable(errors.UnknownReason, "provider-demo not available")
}

// SayHello implements helloworld.GreeterServer
func (s *server) EchoSlow(ctx context.Context, in *pb.EchoRequest) (*pb.EchoReply, error) {
	time.Sleep(time.Second)
	return &pb.EchoReply{Param: in.Param, ApplicationName: "provider-demo", SleepTime: "1000ms"}, nil
}

func main() {
	flag.Parse()
	logger := log.DefaultLogger
	log := log.NewHelper(logger)

	s := &server{}
	httpSrv := http.NewServer(
		http.Address("0.0.0.0:8000"),
		http.Middleware(
			recovery.Recovery(),
			tsf.ServerMiddleware(),
			logging.Server(logger),
		),
	)
	pb.RegisterGreeterHTTPServer(httpSrv, s)

	opts := []kratos.Option{kratos.Name("provider-demo"), kratos.Server(httpSrv)}
	opts = append(opts, tsf.AppOptions(tsf.Medata(AtomMetadata()))...)
	app := kratos.New(opts...)

	if err := app.Run(); err != nil {
		log.Errorf("app run failed:%v", err)
	}
}
