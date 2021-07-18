module github.com/tencentyun/tsf-go/tracing/redisotel

go 1.15

require (
	github.com/go-kratos/kratos/v2 v2.0.0
	github.com/go-redis/redis/extra/rediscmd v0.2.0
	github.com/go-redis/redis/v8 v8.11.0
	github.com/tencentyun/tsf-go v0.1.13
	go.opentelemetry.io/otel v1.0.0-RC1
	go.opentelemetry.io/otel/trace v1.0.0-RC1

)

replace github.com/go-kratos/kratos/v2 v2.0.0-rc7 => github.com/go-kratos/kratos/v2 v2.0.0-20210701014935-bdb51d26969e

replace github.com/tencentyun/tsf-go => ../../
