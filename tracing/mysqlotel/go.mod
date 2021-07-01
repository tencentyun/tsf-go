module github.com/tencentyun/tsf-go/tracing/mysqlotel

go 1.15

require (
	github.com/go-kratos/kratos/v2 v2.0.0-rc7
	github.com/luna-duclos/instrumentedsql v1.1.3
	go.opentelemetry.io/otel v1.0.0-RC1
	go.opentelemetry.io/otel/metric v0.20.0 // indirect
	go.opentelemetry.io/otel/trace v1.0.0-RC1
	github.com/tencentyun/tsf-go v0.1.13
)

replace github.com/go-kratos/kratos/v2 v2.0.0-rc7 => github.com/go-kratos/kratos/v2 v2.0.0-20210701014935-bdb51d26969e
replace github.com/tencentyun/tsf-go => ../../
