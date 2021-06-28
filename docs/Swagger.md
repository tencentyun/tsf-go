# Swagger API
### 集成
tsf-go自动集成了swagger api
在应用的启动入口`kratos.New()`中加了`tsf.AppOptions()`这个Option就会自动生成服务的swagger json并上报至TSF治理平台

### 限制
1.现在一个微服务只支持注册一个proto service的swagger json，如果引入的proto文件中定义了了多个Proto Service，那么需要通过`tsf.ProtoServiceName`这个Option手动指定上报哪个service的swagger json:
`tsf.AppOptions(tsf.ProtoServiceName("<package_name.service_name>"))`
`<package_name.service_name>` 需要替换成实际的Proto Service名字(比如`helloworld.Greeter`)

2.如果启动的时候日志出现报错`failed to decompress enc: bad gzipped descriptor: EOF`的报错说明被依赖的proto生成时传入的路径不对，
比如:
- api/basedata/tag/v1/tag.proto
- api/basedata/article/v1/article.proto

（其中 **api/basedata/article/v1/article.proto** 依赖 **api/basedata/tag/v1/tag.proto**;
service定义在**api/basedata/article/v1/article.proto** 文件中）


这种情况是由于生成tag.pb.go时没有传入完整的路径，导致生成的source变成了`tag.proto`
那我们只要将完整的依赖路径传给protoc就可以修复了(当然此时需要改成在父目录中执行protoc了)： protoc --proto_path=. --proto_path=./third_party --go_out=paths=source_relative:. api/basedata/tag/v1/tag.proto 
这样生成的tag.pb.go文件中source就是正确的：`api/basedata/tag/v1/tag.proto`

3.如果需要在http server中集成swagger调试页面请参考[OpenAPI Swagger 使用
](https://go-kratos.dev/docs/guide/openapi)