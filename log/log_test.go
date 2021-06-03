package log

import (
	"context"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tencentyun/tsf-go/pkg/meta"
)

func TestLog(t *testing.T) {
	log := log.NewHelper(NewLogger())
	log.Infof("2233")
	log.Info("2233", "niang")
	ctx := meta.WithSys(context.Background(), meta.SysPair{
		Key:   meta.ServiceName,
		Value: "provider",
	})
	log.WithContext(ctx).Warn("test trace")
}
