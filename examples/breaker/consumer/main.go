package main

import (
	"context"
	"math/rand"
	"time"

	tsf "github.com/tencentyun/tsf-go"
	"github.com/tencentyun/tsf-go/breaker"
	pb "github.com/tencentyun/tsf-go/examples/helloworld/proto"
	"github.com/tencentyun/tsf-go/log"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/transport/http"
)

func main() {
	callHTTP()
}

func callHTTP() {
	logger := log.DefaultLogger
	log := log.NewHelper(logger)

	clientOpts := []http.ClientOption{http.WithEndpoint("127.0.0.1:8000")}

	// tsf breaker middleware 默认error code大于等于500才认为出错并MarkFailed
	// 这里我们改成大于400就认为出错
	errHook := func(ctx context.Context, operation string, err error) (success bool) {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) || errors.FromError(err).GetCode() > 400 {
			return false
		}
		return true
	}

	cfg := &breaker.Config{
		// 成功率放大系数，即k * success < total时触发熔断
		// 默认1.5
		K: 1.4,
		// 熔断触发临界请求量
		// 统计窗口内请求量低于Request值则不触发熔断
		// 默认值 20
		Request: 10,
	}
	clientOpts = append(clientOpts, tsf.ClientHTTPOptions(tsf.WithMiddlewares(
		// 插入Breaker Middleware
		tsf.BreakerMiddleware(
			tsf.WithBreakerErrorHook(errHook),
			tsf.WithBreakerConfig(cfg),
		)),
	)...)
	httpConn, err := http.NewClient(context.Background(), clientOpts...)
	if err != nil {
		log.Fatalf("dial http err:%v", err)
	}
	client := pb.NewGreeterHTTPClient(httpConn)
	for {
		var name = "ok"
		if rand.Intn(100) >= 40 {
			name = "error"
		}
		reply, err := client.SayHello(context.Background(), &pb.HelloRequest{Name: name})
		if err != nil {
			log.Errorf(" SayHello failed!err:=%v\n", err)
		} else {
			log.Infof("SayHello success!:%s", reply.Message)
		}
		time.Sleep(time.Millisecond * 200)
	}

}
