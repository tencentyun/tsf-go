module github.com/tencentyun/tsf-go

go 1.14

require (
	github.com/elazarl/goproxy v0.0.0-20210110162100-a92cc753f88e
	github.com/fullstorydev/grpcurl v1.8.1
	github.com/gin-gonic/gin v1.7.2
	github.com/go-kratos/kratos/v2 v2.0.0-rc6
	github.com/go-kratos/swagger-api v0.1.4
	github.com/go-playground/validator/v10 v10.6.1 // indirect
	github.com/gobwas/pool v0.2.1 // indirect
	github.com/gobwas/ws v1.0.3 // indirect
	github.com/gogo/protobuf v1.3.2
	github.com/google/gops v0.3.18
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/jhump/protoreflect v1.8.2
	github.com/json-iterator/go v1.1.11 // indirect
	github.com/klauspost/compress v1.13.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/leodido/go-urn v1.2.1 // indirect
	github.com/longXboy/go-grpc-http1 v0.0.0-20201202084506-0a6dbcb9e0f7
	github.com/mattn/go-isatty v0.0.13 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/natefinch/lumberjack v2.0.0+incompatible
	github.com/onsi/ginkgo v1.15.0 // indirect
	github.com/onsi/gomega v1.10.5 // indirect
	github.com/openzipkin/zipkin-go v0.2.5
	github.com/stretchr/testify v1.7.0
	github.com/ugorji/go v1.2.6 // indirect
	go.opentelemetry.io/otel v0.20.0
	go.opentelemetry.io/otel/sdk v0.20.0
	go.opentelemetry.io/otel/trace v0.20.0
	go.uber.org/atomic v1.8.0 // indirect
	go.uber.org/multierr v1.7.0 // indirect
	go.uber.org/zap v1.17.0
	golang.org/x/crypto v0.0.0-20210616213533-5ff15b29337e
	golang.org/x/net v0.0.0-20210614182718-04defd469f4e
	google.golang.org/genproto v0.0.0-20210617175327-b9e0b3197ced
	google.golang.org/grpc v1.38.0
	google.golang.org/protobuf v1.26.0
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
	nhooyr.io/websocket v1.8.7 // indirect
)

replace github.com/go-kratos/kratos/v2 v2.0.0-rc6 => github.com/go-kratos/kratos/v2 v2.0.0-20210624100455-07f9fa3e91db
