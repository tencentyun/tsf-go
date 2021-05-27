package main

import (
	"context"
	"log"
	"time"

	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	transhttp "github.com/go-kratos/kratos/v2/transport/http"
	pb "github.com/tencentyun/tsf-go/examples/helloworld/proto"
	"github.com/tencentyun/tsf-go/naming/consul"
)

func main() {

	c := consul.DefaultConsul()
	callHTTP(c)
	callGRPC(c)
}

func callGRPC(c *consul.Consul) {
	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint("discovery:///helloworld"),
		grpc.WithDiscovery(c),
	)
	if err != nil {
		log.Fatal(err)
	}
	time.Sleep(time.Second)
	client := pb.NewGreeterClient(conn)
	reply, err := client.SayHello(context.Background(), &pb.HelloRequest{Name: "kratos_grpc"})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("[grpc] SayHello %+v\n", reply)
}

func callHTTP(c *consul.Consul) {
	conn, err := transhttp.NewClient(
		context.Background(),
		transhttp.WithMiddleware(
			recovery.Recovery(),
		),
		transhttp.WithScheme("http"),
		transhttp.WithEndpoint("discovery:///helloworld"),
		transhttp.WithDiscovery(c),
	)
	if err != nil {
		log.Fatal(err)
	}
	time.Sleep(time.Millisecond * 250)
	client := pb.NewGreeterHTTPClient(conn)
	reply, err := client.SayHello(context.Background(), &pb.HelloRequest{Name: "kratos_http"})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("[http] SayHello %s\n", reply.Message)

}
