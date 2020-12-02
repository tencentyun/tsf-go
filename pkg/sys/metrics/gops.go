package metrics

import (
	"context"
	"fmt"

	"github.com/google/gops/agent"
	"github.com/tencentyun/tsf-go/pkg/log"
	"github.com/tencentyun/tsf-go/pkg/sys/env"
)

func StartAgent() {
	if !env.DisableDisableGops() {
		addr := fmt.Sprintf(":%d", env.GopsPort())
		if err := agent.Listen(agent.Options{Addr: addr}); err != nil {
			log.L().Errorf(context.Background(), "gops agent.Listen %s err: %v", addr, err)
		}
	}
}
