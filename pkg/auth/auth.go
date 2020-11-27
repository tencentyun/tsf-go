package auth

import (
	"context"

	"github.com/tencentyun/tsf-go/pkg/config"
	"github.com/tencentyun/tsf-go/pkg/naming"
)

type Builder interface {
	Build(cfg config.Config, svc naming.Service) Auth
}

type Auth interface {
	// api为被访问的接口名
	Verify(ctx context.Context, api string) error
}
