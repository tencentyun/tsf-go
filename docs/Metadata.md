# 用户自定义标签
自定义标签的作用：[系统和业务自定义标签](https://cloud.tencent.com/document/product/649/34136)

在链路中传递自定义标签：
```go
import "github.com/tencentyun/tsf-go/pkg/meta"

ctx = meta.WithUser(ctx, meta.UserPair{Key: "user", Value: "test2233"})
s.client.SayHello(ctx, req)
```
在下游server中获取自定义标签：
```go
import "github.com/tencentyun/tsf-go/pkg/meta"

fmt.Println(meta.User(ctx,"user"))
```