module github.com/tencentyun/tsf-go/examples

go 1.15

require (
	github.com/gin-gonic/gin v1.7.2
	github.com/go-kratos/kratos/v2 v2.0.0
	github.com/go-redis/redis/v8 v8.11.0
	github.com/go-sql-driver/mysql v1.5.0
	github.com/luna-duclos/instrumentedsql v1.1.3
	github.com/tencentyun/tsf-go v1.0.0-rc1
	github.com/tencentyun/tsf-go/tracing/mysqlotel v1.0.0-rc1
	github.com/tencentyun/tsf-go/tracing/redisotel v1.0.0-rc1
	google.golang.org/genproto v0.0.0-20210630183607-d20f26d13c79
	google.golang.org/grpc v1.39.0
	google.golang.org/protobuf v1.27.1
)

replace github.com/tencentyun/tsf-go => ../

replace github.com/tencentyun/tsf-go/tracing/redisotel => ../tracing/redisotel

replace github.com/tencentyun/tsf-go/tracing/mysqlotel => ../tracing/mysqlotel
