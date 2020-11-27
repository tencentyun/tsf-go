package balancer

import (
	"context"

	"github.com/tencentyun/tsf-go/pkg/naming"
)

type Balancer interface {
	Pick(ctx context.Context, nodes []naming.Instance) (node *naming.Instance)
}
