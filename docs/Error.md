# 业务错误处理
### 设计理念
在 API 中，业务错误主要通过 proto 进行定义，并且通过工具自动生成辅助代码。
在 errors.Error 中，主要实现了 HTTP 和 gRPC 的接口：
```go
// 渲染http status code
StatusCode() int
// grpc status
GRPCStatus() *grpc.Status
```
可以看下一个error的基本定义
```go
type Error struct {
    // 错误码，跟 http-status 一致，并且在 grpc 中可以转换成 grpc-status。
    Code     int32
    // 错误原因，定义为业务判定错误码。            
    Reason   string
    // 错误信息，为用户可读的信息，可作为用户提示内容。      
    Message  string    
    // 错误元信息，为错误添加附加可扩展信息。       
    Metadata map[string]string 
}
```
### 定义Error
1.安装 errors 辅助代码生成工具：
`go get github.com/go-kratos/kratos/cmd/protoc-gen-go-errors`
2.定义错误码Reason
```protobuf
syntax = "proto3";

package helloworld;

// import third party proto
import "errors/errors.proto";

option go_package = "github.com/tencentyun/tsf-go/examples/error/errors";

enum ErrorReason {
  // 设置缺省错误码
  option (errors.default_code) = 500;
  
  // 为某个枚举单独设置错误码
  USER_NOT_FOUND = 0 [(errors.code) = 404];

  CONTENT_MISSING = 1 [(errors.code) = 400];;
}
```
3.通过 proto 生成对应的代码：
```bash
protoc --proto_path=. \
--proto_path=./third_party \
--go_out=paths=source_relative:. \
--go-errors_out=paths=source_relative:. \
*.proto
```
注意需要将proto依赖的[third_party](https://github.com/tencentyun/tsf-go/tree/master/third_party)下载至您的项目中，并替换成实际路径
4.使用生成的 errors 辅助代码：
```go
// server return error
pb.ErrorUserNotFound("user %s not found", "kratos")
// client get and compare error
pb.IsUserNotFound(err)
```