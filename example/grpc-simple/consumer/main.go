package main

import (
	"context"
	"flag"
	"time"

	"github.com/tencentyun/tsf-go/pkg/grpc/client"
	"github.com/tencentyun/tsf-go/pkg/log"
	"github.com/tencentyun/tsf-go/pkg/meta"
	pb "github.com/tencentyun/tsf-go/testdata"
	"google.golang.org/grpc"
)

func main() {
	flag.Parse()
	newService()
	doWork()
}

func doWork() {
	time.Sleep(time.Second * 2)
	cc, err := client.Dial("127.0.0.1:8082", grpc.WithInsecure())
	if err != nil {
		panic(err)
	}

	// get client stub
	greeter := pb.NewGreeterClient(cc.GrpcConn())
	for {
		time.Sleep(time.Second * 2)
		ctx, _ := context.WithTimeout(context.Background(), time.Second*2)
		ctx = meta.WithUser(ctx, meta.UserPair{"user", "test2233"})
		resp, err := greeter.SayHello(ctx, &pb.HelloRequest{Name: "lobser!"})
		if err != nil {
			log.Errorf(context.Background(), "got err: %v", err)
			continue
		}
		log.Infof(context.Background(), "unary SayHello resp: %v", resp)

		ctx, _ = context.WithTimeout(context.Background(), time.Second*2)
		stream, err := greeter.SayHelloStream(ctx)
		if err != nil {
			log.Errorf(context.Background(), "stream got err: %v", err)
			continue
		}
		err = stream.Send(&pb.HelloRequest{Name: "stream lobser"})
		if err != nil {
			log.Errorf(context.Background(), "stream got err: %v", err)
			continue
		}
		resp, err = stream.Recv()
		if err != nil {
			log.Errorf(context.Background(), "stream got err: %v", err)
			continue
		}
		stream.CloseSend()
		log.Infof(context.Background(), "steam SayHello resp: %v", resp)
	}
}
