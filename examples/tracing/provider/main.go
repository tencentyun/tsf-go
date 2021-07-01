package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2"
	klog "github.com/go-kratos/kratos/v2/log"

	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/go-redis/redis/v8"
	tsf "github.com/tencentyun/tsf-go"
	pb "github.com/tencentyun/tsf-go/examples/helloworld/proto"
	"github.com/tencentyun/tsf-go/log"
	"github.com/tencentyun/tsf-go/tracing"
	"github.com/tencentyun/tsf-go/tracing/redisotel"
)

// server is used to implement helloworld.GreeterServer.
type server struct {
	log      *klog.Helper
	redisCli *redis.Client
}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	err := s.redisCli.Incr(ctx, "hello_count").Err()
	if err != nil {
		s.log.Errorf("set redis incr failed!err:=%v", err)
	}
	return &pb.HelloReply{Message: fmt.Sprintf("Welcome %+v!", in.Name)}, nil
}

func main() {
	flag.Parse()
	logger := log.DefaultLogger
	log := log.NewHelper(logger)

	// 主动设置trace采样率为100%，如果不设置默认为10%
	tracing.SetProvider(tracing.WithSampleRatio(1.0))
	// 初始化redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:         "127.0.0.1:6379",
		Password:     "",
		DB:           int(0),
		DialTimeout:  time.Second * 3,
		WriteTimeout: time.Second * 3,
		ReadTimeout:  time.Second * 10,
	})
	redisClient.AddHook(redisotel.New("127.0.0.1:6379"))

	s := &server{
		redisCli: redisClient,
		log:      log,
	}
	httpSrv := http.NewServer(
		http.Address("0.0.0.0:8000"),
		http.Middleware(
			recovery.Recovery(),
			// 将tracing采样率提升至100%
			tsf.ServerMiddleware(),
			logging.Server(logger),
		),
	)
	pb.RegisterGreeterHTTPServer(httpSrv, s)

	opts := []kratos.Option{kratos.Name("provider-http"), kratos.Server(httpSrv)}
	opts = append(opts, tsf.AppOptions()...)
	app := kratos.New(opts...)

	if err := app.Run(); err != nil {
		log.Errorf("app run failed:%v", err)
	}
}
