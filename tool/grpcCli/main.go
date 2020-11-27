package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/gogo/protobuf/jsonpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/encoding"
)

// Reply for test
type Reply struct {
	res []byte
}

var (
	data          string
	file          string
	method        string
	addr          string
	tlsCert       string
	tlsServerName string
	appID         string
	env           string
)

//Reference https://jbrandhorst.com/post/grpc-json/
func init() {
	encoding.RegisterCodec(JSON{
		Marshaler: jsonpb.Marshaler{
			EmitDefaults: true,
			OrigName:     true,
		},
	})
	flag.StringVar(&data, "data", `{"name":"longxia}`, `-data '{"name":"longxia"}'`)
	flag.StringVar(&file, "file", ``, `./data.json`)
	flag.StringVar(&method, "method", "/tsf.test.helloworld.Greeter/SayHello", `-method /testproto.Greeter/SayHello`)
	flag.StringVar(&addr, "addr", "127.0.0.1:8080", `127.0.0.1:8080`)
	flag.StringVar(&tlsCert, "cert", "", `./cert.pem`)
	flag.StringVar(&tlsServerName, "server_name", "", `hello_server`)
}

// 该example因为使用的是json传输格式所以只能用于调试或测试，用于线上会导致性能下降
// 使用方法：
//  ./grpcCli -data='{"name":"xia","age":19}' -addr=127.0.0.1:8080 -method=/testproto.Greeter/SayHello
//  ./grpcCli -file=data.json -addr=127.0.0.1:8080 -method=/testproto.Greeter/SayHello
func main() {
	flag.Parse()
	opts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(grpc.CallContentSubtype(JSON{}.Name())),
	}
	if tlsCert != "" {
		creds, err := credentials.NewClientTLSFromFile(tlsCert, tlsServerName)
		if err != nil {
			panic(err)
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	}
	if file != "" {
		content, err := ioutil.ReadFile(file)
		if err != nil {
			fmt.Printf("ioutil.ReadFile(%s) error(%v)\n", file, err)
			os.Exit(1)
		}
		if len(content) > 0 {
			data = string(content)
		}
	}
	conn, err := grpc.Dial(addr, opts...)
	if err != nil {
		panic(err)
	}
	var reply Reply
	err = grpc.Invoke(context.Background(), method, []byte(data), &reply, conn)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(reply.res))
}

// JSON is impl of encoding.Codec
type JSON struct {
	jsonpb.Marshaler
	jsonpb.Unmarshaler
}

// Name is name of JSON
func (j JSON) Name() string {
	return "json"
}

// Marshal is json marshal
func (j JSON) Marshal(v interface{}) (out []byte, err error) {
	return v.([]byte), nil
}

// Unmarshal is json unmarshal
func (j JSON) Unmarshal(data []byte, v interface{}) (err error) {
	v.(*Reply).res = data
	return nil
}
