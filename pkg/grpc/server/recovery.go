package server

import (
	"context"
	"fmt"
	"os"
	"runtime"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/tencentyun/tsf-go/pkg/log"
	"google.golang.org/grpc"
)

func (s *Server) recovery(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	defer func() {
		if rerr := recover(); rerr != nil {
			const size = 64 << 10
			buf := make([]byte, size)
			rs := runtime.Stack(buf, false)
			if rs > size {
				rs = size
			}
			buf = buf[:rs]
			pl := fmt.Sprintf("grpc server panic: %v\n%v\n%s\n", req, rerr, buf)
			fmt.Fprintf(os.Stderr, pl)
			log.Error(ctx, pl)
			err = errors.InternalServer(errors.UnknownReason, "")
		}
	}()
	resp, err = handler(ctx, req)
	return
}

func (s *Server) recoveryStream(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
	defer func() {
		if rerr := recover(); rerr != nil {
			const size = 64 << 10
			buf := make([]byte, size)
			rs := runtime.Stack(buf, false)
			if rs > size {
				rs = size
			}
			buf = buf[:rs]
			pl := fmt.Sprintf("grpc server panic: %v\n%v\n%s\n", srv, rerr, buf)
			fmt.Fprintf(os.Stderr, pl)
			log.Error(stream.Context(), pl)
			err = errors.InternalServer(errors.UnknownReason, "")
		}
	}()
	err = handler(srv, stream)
	return
}
