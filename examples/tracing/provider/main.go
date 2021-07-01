package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"time"

	tsf "github.com/tencentyun/tsf-go"
	pb "github.com/tencentyun/tsf-go/examples/helloworld/proto"
	"github.com/tencentyun/tsf-go/log"
	"github.com/tencentyun/tsf-go/tracing"
	"github.com/tencentyun/tsf-go/tracing/mysqlotel"
	"github.com/tencentyun/tsf-go/tracing/redisotel"

	"github.com/go-kratos/kratos/v2"
	klog "github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/go-redis/redis/v8"
	"github.com/go-sql-driver/mysql"
	"github.com/luna-duclos/instrumentedsql"
)

// server is used to implement helloworld.GreeterServer.
type server struct {
	log      *klog.Helper
	redisCli *redis.Client
	db       *sql.DB
}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	err := s.redisCli.Incr(ctx, "hello_count").Err()
	if err != nil {
		s.log.Errorf("set redis incr failed!err:=%v", err)
	}
	row := s.db.QueryRowContext(ctx, "select id from users limit 1")
	var id int64
	err = row.Scan(&id)
	if err != nil {
		s.log.Errorf("get id from mysql failed!err:=%v", err)
	}
	return &pb.HelloReply{Message: fmt.Sprintf("Welcome %+v!", in.Name)}, nil
}

func main() {
	flag.Parse()
	logger := log.DefaultLogger
	log := log.NewHelper(logger)

	// 主动设置trace采样率为30%
	// 如果上游parent span中设置了是否采样，则以上游span为准，忽略采样率设置
	// 在这个example中，由于consumer采样率设置了100%，所以provider实际采样率也为100%
	tracing.SetProvider(tracing.WithSampleRatio(0.3))
	// 初始化redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:         "127.0.0.1:6379",
		Password:     "",
		DB:           int(0),
		DialTimeout:  time.Second * 3,
		WriteTimeout: time.Second * 3,
		ReadTimeout:  time.Second * 10,
	})
	// 给redis添加otel tracing钩子
	redisClient.AddHook(redisotel.New("127.0.0.1:6379"))

	// 注册mysql的tracing instrument
	sql.Register("tracing-mysql",
		instrumentedsql.WrapDriver(mysql.MySQLDriver{},
			instrumentedsql.WithTracer(mysqlotel.NewTracer("127.0.0.1:3306")),
			instrumentedsql.WithOmitArgs(),
		),
	)
	db, err := sql.Open("tracing-mysql", "root:123456@tcp(127.0.0.1:3306)/pie")
	if err != nil {
		panic(err)
	}
	err = db.Ping()
	if err != nil {
		panic(err)
	}
	s := &server{
		redisCli: redisClient,
		log:      log,
		db:       db,
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
