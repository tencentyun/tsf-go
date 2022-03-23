module github.com/tencentyun/tsf-go/examples

go 1.15

require (
	github.com/gin-gonic/gin v1.7.3
	github.com/go-kratos/kratos/v2 v2.0.5
	github.com/go-redis/redis/v8 v8.11.5
	github.com/go-sql-driver/mysql v1.6.0
	github.com/luna-duclos/instrumentedsql v1.1.3
	github.com/tencentyun/tsf-go v0.0.0-20220323120705-9f1a22c7d03b
	google.golang.org/genproto v0.0.0-20210811021853-ddbe55d93216
	google.golang.org/grpc v1.40.0
	google.golang.org/protobuf v1.27.1
)

replace github.com/tencentyun/tsf-go => ../
