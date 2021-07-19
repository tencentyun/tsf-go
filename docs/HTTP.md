# HTTP 开发
## 服务端开发
#### 1.通过protobuf定义HTTP服务接口
这里使用[gogleapis规范](https://github.com/googleapis/googleapis/blob/master/google/api/http.proto#L46)定义的option来描述http接口，完整的示例可以参考[a_bit_of_everything.proto](https://github.com/grpc-ecosystem/grpc-gateway/blob/master/examples/internal/proto/examplepb/a_bit_of_everything.proto)
```protobuf
syntax = "proto3";

// 定义protobuf包名pacakage_name
package helloworld;

// 如果不使用restful http协议，可以不引入此proto
import "google/api/annotations.proto";

// 这里go_package指定的是protofbu生成文件xxx.pb.go在git上的地址
option go_package = "github.com/tencentyun/tsf-go/examples/helloworld/proto";

// 定义服务名service_name
service Greeter {
  rpc GetHelloGET (HelloRequest) returns (HelloReply)  {       
        // HTTP get 请求，会将HelloRequest中的id作为path parameter
        option (google.api.http) = {
            get: "/helloworld/{id}",
        };
  }
  rpc UpdateHello (HelloRequest) returns (HelloReply)  {       
        // HTTP post请求，会将HelloRequest中的id作为path parameter，
        // 同时也会encode HelloRequest 作为HTTP request body
        option (google.api.http) = {
            post: "/helloworld/{id}",
            body: "*",
        };
  }
}

//  请求参数
message HelloRequest {
  string id = 1;
  string name = 2;
}

// 响应参数
message HelloReply {
  string message = 1;
}
```
#### 2.生成服务端桩代码xxx.pb.go代码
通过protoc命令生成服务代码(http协议)
`protoc --proto_path=. --proto_path=./third_party
--go_out=paths=source_relative:. --go_out=paths=source_relative:. --go-http_out=paths=source_relative:.  *.proto`
- 如果没有定义google.api.http，但仍想生成xxx_http.pb.go代码，则生成时需要加上参数--go-http_opt=omitempty=false
- 注意需要将proto依赖的[third_party](https://github.com/tencentyun/tsf-go/tree/master/third_party)下载至您的项目中，并替换成实际路径
#### 3.编写service实现层代码
```go
import	pb "github.com/tencentyun/tsf-go/examples/helloworld/proto"

// server is used to implement helloworld.GreeterServer.
type server struct {
	pb.UnimplementedGreeterServer
}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	return &pb.HelloReply{Message: fmt.Sprintf("Welcome %+v!", in.Name)}, nil
}
```
#### 4.编写server(http协议)启动入口main.go
```go
import  pb "github.com/tencentyun/tsf-go/examples/helloworld/proto"
import 	tsf "github.com/tencentyun/tsf-go"
import  "github.com/go-kratos/kratos/v2"
import  "github.com/go-kratos/kratos/v2/transport/grpc"

func main() {
  flag.Parse()

  s := &server{}
  httpSrv := http.NewServer(
    http.Address(":8000"),
    http.Middleware(
    recovery.Recovery(),
    tsf.ServerMiddleware(),
    ),
  )
  pb.RegisterGreeterHTTPServer(httpSrv, s)

  opts := []kratos.Option{kratos.Name("provider-http"), kratos.Server(httpSrv)}
  opts = append(opts, tsf.AppOptions()...)
  app := kratos.New(opts...)

  if err := app.Run(); err != nil {
    panic(err) 
  }
}
```
### 5.服务启动
- 参考[腾讯云文档](https://cloud.tencent.com/document/product/649/16618)搭建并启动一个本地轻量级consul注册中心；如果暂时不想启动服务发现，则可以在第4步骤中将`opts = append(opts, tsf.AppOptions()...)`这行代码删除即可
- 执行`go run main.go`即可启动server

## 客户端开发（http协议）
### 1.编写客户端代码
```go
import  pb "github.com/tencentyun/tsf-go/examples/helloworld/proto"
import  tsf "github.com/tencentyun/tsf-go"
import 	"github.com/go-kratos/kratos/v2/transport/grpc"
import  "import  "github.com/go-kratos/kratos/v2"

func main() {
    flag.Parse()
    // 指定被调方服务连接地址:<scheme>://<authority>/<service_name>
    // 如果使用服务发现，此处scheme固定为discovery，authority留空，service_name为定义注册到服务发现中的服务名
    // 如果不使用服务发现，直接填写"<ip>:<port>"即可
    clientOpts := []http.ClientOption{http.WithEndpoint("discovery:///provider-http")}
    // 如果不使用服务发现，此行可以删除
	  clientOpts = append(clientOpts, tsf.ClientHTTPOptions()...)
	  httpConn, err := http.NewClient(context.Background(), clientOpts...)
	  if err != nil {
		  panic(err)
	  }
    client := pb.NewGreeterHTTPClient(httpConn)
    reply, err := client.SayHello(context.Background(), &pb.HelloRequest{Name: "tsf_grpc"})
    if err != nil {
        panic(err)
    }
}
```
#### 2.启动客户端
执行`go run main.go`即可启动client