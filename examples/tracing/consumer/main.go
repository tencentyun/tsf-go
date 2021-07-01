package main

import (
	"context"
	"flag"
	"log"
	"time"

	transhttp "github.com/go-kratos/kratos/v2/transport/http"
	pb "github.com/tencentyun/tsf-go/examples/helloworld/proto"
	"github.com/tencentyun/tsf-go/naming/consul"
	"github.com/tencentyun/tsf-go/tracing"
)

func main() {
	flag.Parse()
	// 将tracing采样率提升至100%
	// 如果不设置，默认为10%
	tracing.SetProvider(tracing.WithSampleRatio(1.0))
	go func() {
		for {
			time.Sleep(time.Millisecond * 1000)
			callHTTP()
			time.Sleep(time.Second)
		}
	}()

	newService(consul.DefaultConsul())
}

func callHTTP() {
	conn, err := transhttp.NewClient(
		context.Background(),
		transhttp.WithEndpoint("127.0.0.1:8080"),
	)
	if err != nil {
		panic(err)
	}
	client := pb.NewGreeterHTTPClient(conn)
	reply, err := client.SayHello(context.Background(), &pb.HelloRequest{Name: "kratos_http"})
	if err != nil {
		panic(err)
	}
	log.Printf("[http] SayHello %s\n", reply.Message)
}
