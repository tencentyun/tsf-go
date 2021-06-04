package authenticator

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/tencentyun/tsf-go/log"
	"github.com/tencentyun/tsf-go/pkg/auth"
	"github.com/tencentyun/tsf-go/pkg/config"
	"github.com/tencentyun/tsf-go/pkg/naming"
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
			log.DefaultLog.Debugw("msg", "Authenticator.Verify hit blacklist,access blocked!", "rule", rule.tagRule)
			return errors.Forbidden(errors.UnknownReason, "")
		}
	}
	if authConfig.Type == "W" {
		log.DefaultLog.Debug("Authenticator.Verify not hit whitelist,access blocked!")
		return errors.Forbidden(errors.UnknownReason, "")
	}
	return nil
}

func (a *Authenticator) refreshRule() {
	for {
		specs, err := a.watcher.Watch(a.ctx)
		if err != nil {
			if errors.IsGatewayTimeout(err) || errors.IsClientClosed(err) {
				log.DefaultLog.Errorw("msg", "watch auth config deadline or clsoe!exit now!", "err", err)
				return
			}
			log.DefaultLog.Errorw("msg", "watch auth config failed!", "err", err)
			continue
		}
		var authConfigs []AuthConfig
		for _, spec := range specs {
			if spec.Key != fmt.Sprintf("authority/%s/%s/data", a.svc.Namespace, a.svc.Name) {
				err = fmt.Errorf("found invalid auth config key!")
				log.DefaultLog.Errorw("msg", "found invalid auth config key!", "key", spec.Key, "expect", fmt.Sprintf("authority/%s/%s/data", a.svc.Namespace, a.svc.Name))
				continue
			}
			err = spec.Data.Unmarshal(&authConfigs)
			if err != nil {
				log.DefaultLog.Errorw("msg", "unmarshal auth config failed!", "err", err, "raw", string(spec.Data.Raw()))
				continue
			}
		}
		if len(authConfigs) == 0 && err != nil {
			log.DefaultLog.Error("get auth config failed,not override old data!")
			continue
		}
		var authConfig *AuthConfig
		if len(authConfigs) > 0 {
			authConfig = &authConfigs[0]
			for _, rule := range authConfig.Rules {
				rule.genTagRules()
			}
		}
		log.DefaultLog.Infof("[auth] found new auth rules,replace now!config: %v", authConfig)
		a.mu.Lock()
		a.authConfig = authConfig
		a.mu.Unlock()
	}
}
