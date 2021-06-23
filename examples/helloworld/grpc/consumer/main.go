package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/go-kratos/kratos/v2/transport/grpc"
	pb "github.com/tencentyun/tsf-go/examples/helloworld/proto"
	"github.com/tencentyun/tsf-go/naming/consul"
)

func main() {
	flag.Parse()

	c := consul.DefaultConsul()

	go func() {
		for {
			time.Sleep(time.Millisecond * 1000)
			callGRPC()
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
		log.Fatal("say hello failed!", err)
	}
	log.Printf("[grpc] SayHello %+v\n", reply)
}
