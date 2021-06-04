package log

import (
	"context"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tencentyun/tsf-go/pkg/meta"
)

func TestLog(t *testing.T) {
	log := log.NewHelper(NewLogger())
	log.DefaultLog.Infof("2233")
	log.DefaultLog.Info("2233", "niang", "5566")
	log.DefaultLog.Infow("name", "niang")
	log.DefaultLog.Infow("msg", "request", "name", "niang")

	ctx := meta.WithSys(context.Background(), meta.SysPair{
		Key:   meta.ServiceName,
		Value: "provider",
	})
	log.WithContext(ctx).Warn("test trace")
}
