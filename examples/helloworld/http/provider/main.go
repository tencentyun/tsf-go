package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/http"
	pb "github.com/tencentyun/tsf-go/examples/helloworld/proto"

	tsf "github.com/tencentyun/tsf-go"
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
	flag.Parse()
	logger := log.NewStdLogger(os.Stdout)
	log := log.NewHelper(logger)

	s := &server{}
	httpSrv := http.NewServer(http.Address(":8000"))
	httpSrv.HandlePrefix("/", pb.NewGreeterHandler(s,
		http.Middleware(
			recovery.Recovery(),
			logging.Server(logger),
			tsf.ServerMiddleware("provider-http", 8000),
		)),
	)
	app := kratos.New(
		kratos.Name("provider-http"),
		kratos.Server(
			httpSrv,
		),
		tsf.Metadata(),
		tsf.ID(),
		tsf.Registrar(),
	)

	if err := app.Run(); err != nil {
		log.Errorf("app run failed:%v", err)
	}
}
