package route

import (
	"context"

	"github.com/tencentyun/tsf-go/pkg/naming"
)

type Router interface {
	Select(ctx context.Context, svc naming.Service, nodes []naming.Instance) (selects []naming.Instance)
}
