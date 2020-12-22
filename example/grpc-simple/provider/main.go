package main

import (
	"context"
	"io"
	"time"

	"github.com/tencentyun/tsf-go/pkg/grpc/server"
	"github.com/tencentyun/tsf-go/pkg/log"
	"github.com/tencentyun/tsf-go/pkg/util"
	pb "github.com/tencentyun/tsf-go/testdata"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	util.ParseFlag()

	server := server.NewServer(&server.Config{ServerName: "provider-demo"})
	pb.RegisterGreeterServer(server.GrpcServer(), &Service{})
	server.Use(func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		start := time.Now()
		resp, err = handler(ctx, req)
		log.Info(ctx, "enter grpc handler!", zap.String("method", info.FullMethod), zap.Duration("dur", time.Since(start)))
		return
	})

	err := server.Start()
	if err != nil {
		panic(err)
	}
}

// Service is gRPC service
type Service struct {
}

// SayHello is service method of SayHello
func (s *Service) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	return &pb.HelloReply{Message: "hi " + req.Name}, nil
}

// SayHello is service method of SayHelloStream
func (s *Service) SayHelloStream(stream pb.Greeter_SayHelloStreamServer) error {
	for {
		r, err := stream.Recv()
		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}
		err = stream.Send(&pb.HelloReply{Message: "welcome :" + r.Name})
		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}
	}
}
