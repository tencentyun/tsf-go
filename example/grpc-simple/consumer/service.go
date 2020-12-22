package main

import (
	"context"
	"io"
	"os"
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
	greeter := pb.NewGreeterClient(cc.GrpcConn())

	server := server.NewServer(&server.Config{ServerName: "client-grpc", Port: 8082})
	// Add stop hook
	server.OnStop(func(ctx context.Context) error {
		os.Exit(0)
		return nil
	})
	pb.RegisterGreeterServer(server.Server, &Service{client: greeter})
	go func() {
		err := server.Start()
		if err != nil {
			panic(err)
		}
	}()
}

func (s *Service) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	//注入tsf的用户标签，会传递给下游
	ctx = meta.WithUser(ctx, meta.UserPair{Key: "user", Value: "test2233"})
	return s.client.SayHello(ctx, req)
}

// SayHello is service method of SayHelloStream
func (s *Service) SayHelloStream(stream pb.Greeter_SayHelloStreamServer) error {
	cliStream, err := s.client.SayHelloStream(stream.Context())
	if err != nil {
		return err
	}
	for {
		r, err := stream.Recv()
		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}
		err = cliStream.Send(r)
		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}
		resp, err := cliStream.Recv()
		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}
		err = stream.Send(resp)
		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}
	}
}
