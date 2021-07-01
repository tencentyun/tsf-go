链路追踪
tsf-go已自动集成Opentelemery SDK(opentracing协议)，同时会上报给TSF 调用链追踪平台，默认10%采样率
1. 自定义采样率：
```go
import 	"github.com/tencentyun/tsf-go/tracing"
// 设置采样率为100%
tracing.SetProvider(tracing.WithSampleRatio(1.0))
```
2. 自定义trace span输出（tsf默认以zipkin协议格式输出至/data/tsf_apm/trace/log/trace_log.log）
```go
import 	"github.com/tencentyun/tsf-go/tracing"

type exporter struct {
}

func (e exporter) ExportSpans(ctx context.Context, ss []tracesdk.ReadOnlySpan) error {
   //输出至stdout
   fmt.Println(ss)
}

func (e exporter) Shutdown(ctx context.Context) error {
	return nil
}

// 设置span exporter
tracing.SetProvider(tracing.WithTracerExporter(exporter{}))
```
3. 替换Trace Propagator协议（tsf默认使用zipkin b3协议进行Header传播、解析）
```go
import 	"go.opentelemetry.io/otel"
import 	"go.opentelemetry.io/otel/propagation"

// 可以通过SetTextMapPropagator替换
// 比如这里替换成W3C Trace Context标准格式
// https://www.w3.org/TR/trace-context/
otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.Baggage{}, propagation.TraceContext{}))
``` 
4. Redis\Mysql tracing支持
```go
import 	"database/sql"

import 	"github.com/go-sql-driver/mysql"
import 	"github.com/go-redis/redis/v8"
import 	"github.com/luna-duclos/instrumentedsql"
import 	"github.com/tencentyun/tsf-go/tracing/mysqlotel"
import	"github.com/tencentyun/tsf-go/tracing/redisotel"

// 添加redis tracing hook
redisClient.AddHook(redisotel.New("127.0.0.1:6379"))


// 注册mysql的tracing instrument
sql.Register("tracing-mysql",
    instrumentedsql.WrapDriver(mysql.MySQLDriver{},
        instrumentedsql.WithTracer(mysqlotel.NewTracer("127.0.0.1:3306")),
        instrumentedsql.WithOmitArgs(),
    ),
)
db, err := sql.Open("tracing-mysql", "root:123456@tcp(127.0.0.1:3306)/pie")
```
具体使用方式参考[tracing examples](/examples/tracing)中范例代码
