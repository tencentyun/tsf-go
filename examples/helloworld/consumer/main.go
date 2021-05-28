package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/go-kratos/kratos/v2/transport/grpc"
	transhttp "github.com/go-kratos/kratos/v2/transport/http"
	pb "github.com/tencentyun/tsf-go/examples/helloworld/proto"
	"github.com/tencentyun/tsf-go/naming/consul"
)

func main() {
	flag.Parse()

	c := consul.DefaultConsul()

	go func() {
		for {
			time.Sleep(time.Millisecond * 500)
			callGRPC()
			//callHTTP(c)
			time.Sleep(time.Second)
		}
	}()

	newService(c)
}

func callGRPC() {
	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint("127.0.0.1:9090"),
	)
	if err != nil {
		log.Fatal(err)
	}
	client := pb.NewGreeterClient(conn)
	reply, err := client.SayHello(context.Background(), &pb.HelloRequest{Name: "kratos_grpc"})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("[grpc] SayHello %+v\n", reply)
}

func callHTTP() {
	conn, err := transhttp.NewClient(
		context.Background(),
		transhttp.WithScheme("http"),
		transhttp.WithEndpoint("127.0.0.1:8080"),
	)
	if err != nil {
		log.Fatal(err)
	}
	client := pb.NewGreeterHTTPClient(conn)
	reply, err := client.SayHello(context.Background(), &pb.HelloRequest{Name: "kratos_http"})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("[http] SayHello %s\n", reply.Message)
}
