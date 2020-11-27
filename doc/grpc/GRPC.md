# 使用gRPC协议
### Server
1. 使用grpc-go的protoc IDL插件protoc --go_out=plugins=grpc:. *.proto来生成xxx.pb.go代码
2. `import "github.com/tencentyun/tsf-go/pkg/grpc/server"`	
3. 加入启动代码:：
```
server := server.NewServer(&server.Config{ServerName: "provider-demo"})
pb.RegisterGreeterServer(server.Server, &Service{})
err := server.Start()
if err != nil {
	panic(err)
}
````
如果配置文件中不指定端口则默认为8080，也可以通过tsf_service_port环境或者启动参数来指定，其他的可指定的配置参考`pkg/internal/env/env.go`中定义的Key

### Client
1. 使用grpc-go的protoc IDL插件protoc --go_out=plugins=grpc:. *.proto来生成xxx.pb.go代码
2. `import "github.com/tencentyun/tsf-go/pkg/grpc/client"`
3. 加入new client stub的代码：
```
ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
defer cancel()
cc, err := client.DialWithBlock(ctx, "consul://local/provider-demo")
if err != nil {
	panic(err)
}
greeter := pb.NewGreeterClient(cc.ClientConn)
```
注意上面代码中的`tsf.provider-demo`需要替换成实际被访问的服务提供者的serviceName
`local`的含义是本地命名空间,如果需要发现全局命名空间服务填写`global`


### TSF日志
1.`import 	"github.com/tencentyun/tsf-go/pkg/log"`
2.打印日志
```
log.L().Infof(ctx, "got resp: %v", resp)
log.L().Info(context.Background(), "got message", zap.String("resp",resp))
```
3. 可以通过注入环境变量tsf_log_path或者启动参数tsf_log_path来指定日志输出路径
注意如果不传递go的ctx，会导致日志中不打印traceID

### 分布式配置
1.import配置模块:`"github.com/tencentyun/tsf-go/pkg/config/tsf"`
2.初始化配置模块：`if err := tsf.Init(context.Background());err != nil {
		panic(err)
	}`
3.初始化完毕后直接非阻塞get某一个配置值：
```
if temp, ok := tsf.GetApp("prefix");ok {
	prefix, _ = temp.(string)
}
```
4.订阅某一个配置文件的变化:
```
type AppConfig struct {
	Prefix string `yaml:"prefix"`
}
tsf.AppConfig(func(cfg *tsf.Config) {
		if cfg == nil {
			// 配置文件不存在（被删除）
			return
		}
		var appCfg AppConfig
		err := cfg.Unmarshal(&appCfg)
		if err != nil {
			log.L().Info(context.Background(), "reload remote config failed!", zap.String("raw", string(cfg.Raw())))
			return
		}
		service.prefix.Store(appCfg.Prefix)
})
```
