package router

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"

	"github.com/tencentyun/tsf-go/naming"
	"github.com/tencentyun/tsf-go/pkg/config"
	"github.com/tencentyun/tsf-go/pkg/config/consul"
	"github.com/tencentyun/tsf-go/pkg/log"
	"github.com/tencentyun/tsf-go/pkg/statusError"
	"github.com/tencentyun/tsf-go/pkg/sys/env"
	"github.com/tencentyun/tsf-go/route"

	"go.uber.org/zap"
)

var (
	_ route.Router = &Router{}

	mu           sync.Mutex
	defaultRoute *Router
)

type Config struct {
	NamespaceID string
}

type Router struct {
	conf     *Config
	watcher  config.Watcher
	services atomic.Value

	ctx    context.Context
	cancel context.CancelFunc
}

func DefaultRouter() *Router {
	mu.Lock()
	defer mu.Unlock()
	if defaultRoute == nil {
		defaultRoute = New(
			&Config{
				NamespaceID: env.NamespaceID(),
			},
			consul.DefaultConsul(),
		)
	}
	return defaultRoute
}

func New(conf *Config, cfg config.Config) *Router {
	watcher := cfg.Subscribe(fmt.Sprintf("route/%s/", conf.NamespaceID))
	r := &Router{
		conf:     conf,
		watcher:  watcher,
		services: atomic.Value{},
	}
	r.ctx, r.cancel = context.WithCancel(context.Background())
	go r.refresh()
	return r
}

func (r *Router) Select(ctx context.Context, svc naming.Service, nodes []naming.Instance) (selects []naming.Instance) {
	if len(nodes) == 0 {
		selects = nodes
		return
	}
	if svc.Namespace == "" || svc.Namespace == "local" {
		svc.Namespace = env.NamespaceID()
	}
	services, ok := r.services.Load().(map[naming.Service]RuleGroup)
	if !ok {
		selects = nodes
		return
	}
	ruleGroup, ok := services[svc]

	if !ok || len(ruleGroup.RuleList) == 0 {
		selects = nodes
		return
	}
	var hit bool
	for _, rule := range ruleGroup.RuleList {
		t := rule.toCommonTagRule()
		if t.Hit(ctx) {
			log.Debug(ctx, "[route]: hit rule", zap.Any("svc", svc), zap.Any("rule", rule))
			hit = true
			selects = r.matchByRule(rule, nodes)
			if len(selects) != 0 {
				break
			}
		} else {
			log.Debug(ctx, "[route]: not hit rule", zap.Any("svc", svc), zap.Any("rule", rule))
		}
	}
	if !hit {
		selects = nodes
	} else if len(selects) == 0 && ruleGroup.FallbackStatus {
		selects = nodes
	}
	return selects
}

func (r *Router) matchByRule(rule Rule, nodes []naming.Instance) []naming.Instance {
	var sum int64
	candidates := make(map[string]struct {
		inss   []naming.Instance
		weight int64
	})
	for _, node := range nodes {
		for _, dest := range rule.DestList {
			match := true
			for _, item := range dest.DestItemList {
				if node.Metadata[item.DestItemField] != item.DestItemValue {
					match = false
				}
			}
			if match {
				selects, ok := candidates[dest.DestId]
				if !ok {
					sum += dest.DestWeight
				}
				selects.inss = append(selects.inss, node)
				selects.weight = dest.DestWeight
				candidates[dest.DestId] = selects
			}
		}
	}
	if sum == 0 {
		return nil
	}
	cur := rand.Int63n(sum)
	for _, dest := range candidates {
		sum = sum - dest.weight
		if sum <= cur {
			return dest.inss
		}
	}
	panic(fmt.Errorf("Route.matchByRule impossible code reached! sum:%d candidates:%v", sum, candidates))
}

func (r *Router) refresh() {
	for {
		specs, err := r.watcher.Watch(r.ctx)
		if err != nil {
			if statusError.IsDeadline(err) || statusError.IsClientClosed(err) {
				log.Error(context.Background(), "watch route config deadline or clsoe!exit now!", zap.Error(err))
				return
			}
			log.Error(context.Background(), "watch route config failed!", zap.Error(err))
			continue
		}
		services := make(map[naming.Service]RuleGroup)
		for _, spec := range specs {
			var ruleGroup []RuleGroup
			err = spec.Data.Unmarshal(&ruleGroup)
			if err != nil || len(ruleGroup) == 0 {
				log.Error(context.Background(), "unmarshal route config failed!", zap.Error(err), zap.String("raw", string(spec.Data.Raw())))
				continue
			}
			svc := *naming.NewService(ruleGroup[0].NamespaceId, ruleGroup[0].MicroserviceName)
			services[svc] = ruleGroup[0]
			if ruleGroup[0].NamespaceId != "" && ruleGroup[0].NamespaceId != env.NamespaceID() {
				// 拉到的不是本命名空间的，则说明是全局命名空间的
				svc := *naming.NewService(naming.NsGlobal, ruleGroup[0].MicroserviceName)
				services[svc] = ruleGroup[0]
			}
		}
		if len(services) == 0 && err != nil {
			log.Error(context.Background(), "get route config failed,not override old data!")
			continue
		}
		log.Infof(context.Background(), "[route] found new route,replace now! services: %v", services)
		r.services.Store(services)
	}
}

func (r *Router) Close() {
	r.cancel()
}
