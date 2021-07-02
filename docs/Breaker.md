# 熔断
tsf-go支持[google sre 自适应熔断算法](https://pandaychen.github.io/2020/05/10/A-GOOGLE-SRE-BREAKER/),但是默认不开启，需要用户插入Middleware
1. 插入Breaker Middleware：
```go
clientOpts = append(clientOpts, tsf.ClientHTTPOptions(tsf.WithMiddlewares(
    // 插入Breaker Middleware
    tsf.BreakerMiddleware()),
)...)
```
2. 自定义error hook
```go
// tsf breaker middleware 默认error code大于等于500才认为出错并MarkFailed
	// 这里我们改成大于400就认为出错
errHook := func(ctx context.Context, operation string, err error) (success bool) {
    if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) || errors.FromError(err).StatusCode() > 400 {
        return false
    }
    return true
}
```
3. 自定义breaker配置
```go
cfg := &breaker.Config{
	// 成功率放大系数，即k * success < total时触发熔断
	// 默认1.5
	K: 1.4,
	// 熔断触发临界请求量
	// 统计窗口内请求量低于Request值则不触发熔断
	// 默认值 20
	Request: 10,
}
```
具体使用方法参考[breaker examples](/examples/breaker)