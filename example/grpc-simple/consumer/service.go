package main

import (
	"context"
	"time"

	"github.com/tencentyun/tsf-go/pkg/grpc/client"
	"github.com/tencentyun/tsf-go/pkg/grpc/server"
	"github.com/tencentyun/tsf-go/pkg/meta"
	pb "github.com/tencentyun/tsf-go/testdata"
)

type Service struct {
	client pb.GreeterClient
}

func newService() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	cc, err := client.DialWithBlock(ctx, "consul://local/provider-demo")
	if err != nil {
		panic(err)
	}
	greeter := pb.NewGreeterClient(cc.ClientConn)

	server := server.NewServer(&server.Config{ServerName: "client-grpc", Port: 8082})
	pb.RegisterGreeterServer(server.Server, &Service{client: greeter})
	go func() {
		err := server.Start()
		if err != nil {
			panic(err)
		}
	}()
}

func (s *Service) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	ctx = meta.WithUser(ctx, meta.UserPair{Key: "user", Value: "test2233"})
	return s.client.SayHello(ctx, req)
}
