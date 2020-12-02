package authenticator

import (
	"context"
	"fmt"
	"sync"

	"github.com/tencentyun/tsf-go/pkg/auth"
	"github.com/tencentyun/tsf-go/pkg/config"
	"github.com/tencentyun/tsf-go/pkg/errCode"
	"github.com/tencentyun/tsf-go/pkg/log"
	"github.com/tencentyun/tsf-go/pkg/naming"
	"go.uber.org/zap"
)

var (
	_ auth.Builder = &Builder{}
	_ auth.Auth    = &Authenticator{}
)

type Builder struct {
}

func (b *Builder) Build(cfg config.Config, svc naming.Service) auth.Auth {
	watcher := cfg.Subscribe(fmt.Sprintf("authority/%s/%s/data", svc.Namespace, svc.Name))
	a := &Authenticator{watcher: watcher, svc: svc}
	a.ctx, a.cancel = context.WithCancel(context.Background())
	go a.refreshRule()
	return a
}

type Authenticator struct {
	watcher    config.Watcher
	svc        naming.Service
	authConfig *AuthConfig

	mu sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc
}

func (a *Authenticator) Verify(ctx context.Context, method string) error {
	a.mu.RLock()
	authConfig := a.authConfig
	a.mu.RUnlock()
	if authConfig == nil || len(authConfig.Rules) == 0 {
		return nil
	}

	for _, rule := range a.authConfig.Rules {
		rule.genTagRules()
		if rule.tagRule.Hit(ctx) {
			if authConfig.Type == "W" {
				return nil
			}
			log.Debug(ctx, "Authenticator.Verify hit blacklist,access blocked!", zap.Any("rule", rule.tagRule))
			return errCode.Forbidden
		}
	}
	if authConfig.Type == "W" {
		log.Debug(ctx, "Authenticator.Verify not hit whitelist,access blocked!")
		return errCode.Forbidden
	}
	return nil
}

func (a *Authenticator) refreshRule() {
	for {
		specs, err := a.watcher.Watch(a.ctx)
		if err != nil {
			if errCode.Deadline.Equal(err) || errCode.ClientClosed.Equal(err) {
				log.Error(context.Background(), "watch auth config deadline or clsoe!exit now!", zap.Error(err))
				return
			}
			log.Error(context.Background(), "watch auth config failed!", zap.Error(err))
			continue
		}
		var authConfigs []AuthConfig
		for _, spec := range specs {
			if spec.Key != fmt.Sprintf("authority/%s/%s/data", a.svc.Namespace, a.svc.Name) {
				err = fmt.Errorf("found invalid auth config key!")
				log.Error(context.Background(), "found invalid auth config key!", zap.String("key", spec.Key), zap.String("expect", fmt.Sprintf("authority/%s/%s/data", a.svc.Namespace, a.svc.Name)))
				continue
			}
			err = spec.Data.Unmarshal(&authConfigs)
			if err != nil {
				log.Error(context.Background(), "unmarshal auth config failed!", zap.Error(err), zap.String("raw", string(spec.Data.Raw())))
				continue
			}
		}
		if len(authConfigs) == 0 && err != nil {
			log.Error(context.Background(), "get auth config failed,not override old data!")
			continue
		}
		var authConfig *AuthConfig
		if len(authConfigs) > 0 {
			authConfig = &authConfigs[0]
			for _, rule := range authConfig.Rules {
				rule.genTagRules()
			}
		}
		a.mu.Lock()
		a.authConfig = authConfig
		a.mu.Unlock()
	}
}
