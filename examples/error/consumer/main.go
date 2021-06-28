package main

import (
	"context"

	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/tencentyun/tsf-go"
	pb "github.com/tencentyun/tsf-go/examples/helloworld/proto"
	"github.com/tencentyun/tsf-go/log"
)

func main() {
	callHTTP()
}

func callHTTP() {
	logger := log.DefaultLogger
	log := log.NewHelper(logger)

	clientOpts := []http.ClientOption{http.WithEndpoint("127.0.0.1:8000")}
	clientOpts = append(clientOpts, tsf.ClientHTTPOptions(tsf.WithEnableDiscovery(false))...)
	httpConn, err := http.NewClient(context.Background(), clientOpts...)
	if err != nil {
		log.Fatalf("dial http err:%v", err)
	}
	client := pb.NewGreeterHTTPClient(httpConn)

	reply, err := client.SayHello(context.Background(), &pb.HelloRequest{Name: "empty"})
	if err != nil {
		log.Errorf("[http] SayHello(%s) failed!err:=%v\n", "empty", err)
	} else {
		log.Infof("[http] SayHello %s\n", reply.Message)
	}

	reply, err = client.SayHello(context.Background(), &pb.HelloRequest{Name: "kratos_http"})
	if err != nil {
		log.Errorf("[http] SayHello(%s) failed!err:=%v\n", "kratos_http", err)
	} else {
		log.Infof("[http] SayHello %s\n", reply.Message)
	}

	reply, err = client.SayHello(context.Background(), &pb.HelloRequest{Name: "tsf"})
	if err != nil {
		log.Errorf("[http] SayHello(%s) failed!err:=%v\n", "tsf", err)
	} else {
		log.Infof("[http] SayHello %s\n", reply.Message)
	}
}
