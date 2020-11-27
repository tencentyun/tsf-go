package server

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/longXboy/go-grpc-http1/server"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
)

var (
	magic    = []byte("PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n")
	magicLen = len(magic)
	tls10    = []byte{0x16, 0x03, 0x01}
	tls11    = []byte{0x16, 0x03, 0x02}
	tls12    = []byte{0x16, 0x03, 0x03}
	tls13    = []byte{0x16, 0x03, 0x04}
)

func serveHttp(grpcSrv *grpc.Server, lis net.Listener) {
	opts := []server.Option{server.PreferGRPCWeb(true)}

	downgradingSrv := &http.Server{}
	var h2Srv http2.Server
	err := http2.ConfigureServer(downgradingSrv, &h2Srv)
	if err != nil {
		panic(err)
	}
	downgradingSrv.Handler = h2c.NewHandler(
		server.CreateDowngradingHandler(grpcSrv, http.NotFoundHandler(), opts...),
		&h2Srv)

	r := NewRunner(grpcSrv, lis)

	go downgradingSrv.Serve(r.lisHttp1)
	grpcSrv.Serve(r.lisHttp2)
}

type wrapConn struct {
	net.Conn
	err    error
	header *bytes.Buffer
}

func (c *wrapConn) Read(p []byte) (n int, err error) {
	if c.header.Len() != 0 {
		n, err = c.header.Read(p)
		if err != nil {
			return
		}
		err = c.err
		c.err = nil
		return
	}
	return c.Conn.Read(p)
}

type acceptCh struct {
	conn net.Conn
	err  error
}

type runner struct {
	lis     net.Listener
	ctx     context.Context
	cancel  context.CancelFunc
	grpcSrv *grpc.Server

	lisHttp1 *listener
	lisHttp2 *listener
}

type listener struct {
	runner *runner
	accept chan acceptCh
}

func NewRunner(grpcSrv *grpc.Server, lis net.Listener) *runner {
	r := &runner{grpcSrv: grpcSrv, lis: lis}
	lisHttp1 := &listener{runner: r, accept: make(chan acceptCh)}
	lisHttp2 := &listener{runner: r, accept: make(chan acceptCh)}
	r.lisHttp1 = lisHttp1
	r.lisHttp2 = lisHttp2
	r.ctx, r.cancel = context.WithCancel(context.Background())
	go r.run()
	return r
}

func (l *runner) isH1(conn net.Conn) (net.Conn, bool, error) {
	buf := make([]byte, magicLen)
	n, err := io.ReadAtLeast(conn, buf, magicLen)
	wrap := &wrapConn{conn, err, bytes.NewBuffer(buf[:n])}
	if err != nil {
		return wrap, false, err
	}
	if bytes.Compare(buf, magic) == 0 {
		return wrap, false, err
	} else if bytes.Compare(buf[:3], tls10) == 0 {
		return wrap, false, err
	} else if bytes.Compare(buf[:3], tls11) == 0 {
		return wrap, false, err
	} else if bytes.Compare(buf[:3], tls12) == 0 {
		return wrap, false, err
	} else if bytes.Compare(buf[:3], tls13) == 0 {
		return wrap, false, err
	} else if strings.HasPrefix(string(buf), "POST /") {
		return wrap, true, err
	} else if strings.HasPrefix(string(buf), "GET /") {
		return wrap, true, err
	} else if strings.HasPrefix(string(buf), "HEAD /") {
		return wrap, true, err
	} else if strings.HasPrefix(string(buf), "PUT /") {
		return wrap, true, err
	} else if strings.HasPrefix(string(buf), "DELETE /") {
		return wrap, true, err
	} else if strings.HasPrefix(string(buf), "CONNECT /") {
		return wrap, true, err
	} else if strings.HasPrefix(string(buf), "OPTIONS /") {
		return wrap, true, err
	} else if strings.HasPrefix(string(buf), "TRACE /") {
		return wrap, true, err
	} else if strings.HasPrefix(string(buf), "PATCH	/") {
		return wrap, true, err
	} else {
		return wrap, false, err
	}
}

func (r *runner) run() {
	for {
		conn, err := r.lis.Accept()
		if err != nil {
			item := acceptCh{conn: conn, err: err}
			select {
			case r.lisHttp2.accept <- item:
			case <-r.ctx.Done():
				return
			}
		} else {
			go func(netConn net.Conn) {
				wrap, ok, err := r.isH1(netConn)
				if !ok || err != nil {
					item := acceptCh{conn: wrap}
					select {
					case r.lisHttp2.accept <- item:
					case <-r.ctx.Done():
						return
					}
				} else {
					item := acceptCh{conn: wrap}
					select {
					case r.lisHttp1.accept <- item:
					case <-r.ctx.Done():
						return
					}
				}
			}(conn)
		}
	}
}

func (l *listener) Accept() (net.Conn, error) {
	select {
	case conn := <-l.accept:
		return conn.conn, conn.err
	case <-l.runner.ctx.Done():
		return nil, l.runner.ctx.Err()
	}
}

func (l *listener) Close() error {
	err := l.runner.lis.Close()
	l.runner.cancel()
	return err
}

func (l *listener) Addr() net.Addr {
	return l.runner.lis.Addr()
}
