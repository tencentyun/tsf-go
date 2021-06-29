package main

import (
	"context"

	"github.com/tencentyun/tsf-go/log"
	"github.com/tencentyun/tsf-go/pkg/meta"
)

func main() {

	log := log.NewHelper(
		log.NewLogger(
			log.WithLevel(log.LevelDebug),
			log.WithPath("stdout"),
			log.WithTrace(true),
		),
	)
	log.Infof("app start: %d", 1)
	log.Info("2233", "niang", "5566")
	log.Infow("name", "niang")
	log.Infow("msg", "request", "name", "niang")

	ctx := meta.WithSys(context.Background(), meta.SysPair{
		Key:   meta.ServiceName,
		Value: "provider",
	})
	log.WithContext(ctx).Warn("test trace")
}
