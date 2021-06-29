# 负载均衡
TSF 默认提供了三种负载均衡算法：Random 、P2C 、 Consistent Hashing
默认算法是 P2C

#### 1.Random
随机调度策略
```go
import "github.com/tencentyun/tsf-go/balancer/random"

clientOpts = append(clientOpts, tsf.ClientGrpcOptions(random.New())...)
```

#### 2.P2C （默认）
基于[Power of Two choices](http://www.eecs.harvard.edu/~michaelm/NEWWORK/postscripts/twosurvey.pdf)算法，同时结合请求延迟、错误率、并发数数据实时调整权重的负载均衡策略，从而降低请求响应延迟和后端负载
```go
import "github.com/tencentyun/tsf-go/balancer/p2c"

clientOpts = append(clientOpts, tsf.ClientGrpcOptions(p2c.New())...)
```
#### 3.Consistent Hashing
一致性Hash算法
```go
import "github.com/tencentyun/tsf-go/balancer/hash"

clientOpts = append(clientOpts, tsf.ClientGrpcOptions(hash.New())...)
//将 hash key注入至context中，一致性Hash负载均衡会根据这个key的值进行hash
ctx = hash.NewContext(ctx,"test_key")
client.SayHello(ctx, in)
```