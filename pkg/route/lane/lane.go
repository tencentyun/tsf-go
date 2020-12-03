package lane

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/tencentyun/tsf-go/pkg/config"
	"github.com/tencentyun/tsf-go/pkg/config/consul"
	"github.com/tencentyun/tsf-go/pkg/errCode"
	"github.com/tencentyun/tsf-go/pkg/log"
	"github.com/tencentyun/tsf-go/pkg/meta"
	"github.com/tencentyun/tsf-go/pkg/naming"
	"github.com/tencentyun/tsf-go/pkg/route"
	"github.com/tencentyun/tsf-go/pkg/sys/env"
	"go.uber.org/zap"
)

var (
	_ route.Router = &Lane{}

	mu          sync.Mutex
	defaultLane *Lane
)

type Lane struct {
	ruleWatcher config.Watcher
	laneWathcer config.Watcher

	allRules   []LaneRule
	allLanes   map[string]LaneInfo
	namespaces map[string]map[string]struct{}
	groups     map[string]map[string]struct{}
	rules      []LaneRule          // EFFECTIVE LANE RULES
	lanes      map[string]LaneInfo // EFFECTIVE LANE INFOS
	services   map[string]map[naming.Service]bool
	mu         sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc
}

func DefaultLane() *Lane {
	mu.Lock()
	defer mu.Unlock()
	if defaultLane == nil {
		defaultLane = New(
			consul.DefaultConsul(),
		)
	}
	return defaultLane
}

func New(cfg config.Config) (lane *Lane) {
	ruleWatcher := cfg.Subscribe("lane/rule/")
	laneWatcher := cfg.Subscribe("lane/info/")
	lane = &Lane{
		ruleWatcher: ruleWatcher,
		laneWathcer: laneWatcher,
		allLanes:    map[string]LaneInfo{},
		rules:       []LaneRule{},
		lanes:       map[string]LaneInfo{},
		namespaces:  map[string]map[string]struct{}{},
		groups:      map[string]map[string]struct{}{},
		services:    map[string]map[naming.Service]bool{},
	}
	lane.ctx, lane.cancel = context.WithCancel(context.Background())
	go lane.refreshAllRule()
	go lane.refreshAllLane()
	return
}

func (l *Lane) GetLaneID(ctx context.Context) string {
	l.mu.RLock()
	rules := l.rules
	lanes := l.allLanes
	l.mu.RUnlock()

	for _, rule := range rules {
		if lane, ok := lanes[rule.LaneID]; ok {
			tagRule := rule.toCommonTagRule()
			if tagRule.Hit(ctx) {
				return lane.ID
			}
		}
	}
	return ""
}

func (l *Lane) Select(ctx context.Context, svc naming.Service, nodes []naming.Instance) []naming.Instance {
	if len(nodes) == 0 {
		return nodes
	}
	laneID, ok := meta.Sys(ctx, meta.LaneID).(string)
	if !ok || laneID == "" {
		return l.selectNormal(ctx, svc, nodes)
	}
	l.mu.RLock()
	lane, ok := l.allLanes[laneID]
	if !ok {
		l.mu.RUnlock()
		log.Error(ctx, "[lane.Select] no lane info found in allLanes!", zap.String("laneID", laneID))
		return nodes
	}
	serviceHit := l.services[laneID]
	if serviceHit == nil {
		serviceHit = make(map[naming.Service]bool)
	}
	hit, ok := serviceHit[svc]
	l.mu.RUnlock()

	if !ok {
		for _, node := range nodes {
			appID := node.Metadata[naming.ApplicationID]
			nID := node.Metadata[naming.NamespaceID]
			for _, group := range lane.GroupList {
				if group.ApplicationID == appID && group.NamespaceID == nID {
					hit = true
					break
				}
			}
			if hit {
				break
			}
		}
		l.mu.Lock()
		serviceHit[svc] = hit
		l.services[laneID] = serviceHit
		l.mu.Unlock()
	}
	if hit {
		return l.selectColor(ctx, nodes, lane)
	}
	return l.selectNormal(ctx, svc, nodes)
}

func (l *Lane) selectColor(ctx context.Context, nodes []naming.Instance, lane LaneInfo) []naming.Instance {
	var colors []naming.Instance
	l.mu.RLock()
	groups := l.groups
	l.mu.RUnlock()
	for _, node := range nodes {
		groupID := node.Metadata[naming.GroupID]
		if ids, ok := groups[groupID]; ok && len(ids) > 0 {
			if _, ok := ids[lane.ID]; ok {
				colors = append(colors, node)
			}
		}
	}
	log.Debug(ctx, "lane take effect, choose color instance!", zap.Any("color_nodes", colors))
	return colors
}

func (l *Lane) selectNormal(ctx context.Context, svc naming.Service, nodes []naming.Instance) []naming.Instance {
	l.mu.RLock()
	if lanes, ok := l.namespaces[svc.Namespace]; !ok || len(lanes) == 0 {
		l.mu.RUnlock()
		return nodes
	}
	groups := l.groups
	l.mu.RUnlock()

	var color []naming.Instance
	var normal []naming.Instance
	for _, node := range nodes {
		groupID := node.Metadata[naming.GroupID]
		if groupID != "" {
			if lanes, ok := groups[groupID]; ok && len(lanes) > 0 {
				color = append(color, node)
				continue
			}
		}
		normal = append(normal, node)
	}
	if len(color) > 0 {
		log.Debug(ctx, "lane take effect, filter color instance!", zap.Any("color_nodes", color))
	}
	return normal
}

func (l *Lane) refreshAllRule() {
	for {
		specs, err := l.ruleWatcher.Watch(l.ctx)
		if err != nil {
			if errCode.Deadline.Equal(err) || errCode.ClientClosed.Equal(err) {
				log.Error(context.Background(), "watch lane config deadline or clsoe!exit now!", zap.Error(err))
				return
			}
			log.Error(context.Background(), "watch lane config failed!", zap.Error(err))
			continue
		}
		var allRules []LaneRule
		for _, spec := range specs {
			var rule LaneRule
			err = spec.Data.Unmarshal(&rule)
			if err != nil {
				log.Error(context.Background(), "unmarshal lane rule config failed!", zap.Error(err), zap.String("raw", string(spec.Data.Raw())))
				continue
			}
			allRules = append(allRules, rule)
		}
		if len(allRules) == 0 && err != nil {
			log.Error(context.Background(), "get lane rule config failed,not override old data!")
			continue
		}
		log.Info(context.Background(), "[lane] found new lane rule,replace now!", zap.Any("rules", allRules))
		l.mu.Lock()
		l.allRules = allRules
		l.mu.Unlock()

		l.refreshRules()
	}
}

func (l *Lane) refreshAllLane() {
	for {
		specs, err := l.laneWathcer.Watch(l.ctx)
		if err != nil {
			if errCode.Deadline.Equal(err) || errCode.ClientClosed.Equal(err) {
				log.Error(context.Background(), "watch lane config deadline or clsoe!exit now!", zap.Error(err))
				return
			}
			log.Error(context.Background(), "watch lane config failed!", zap.Error(err))
			time.Sleep(time.Second)
			continue
		}
		allLanes := make(map[string]LaneInfo, 0)
		for _, spec := range specs {
			var lane LaneInfo
			err = spec.Data.Unmarshal(&lane)
			if err != nil {
				log.Error(context.Background(), "unmarshal lane config failed!", zap.Error(err), zap.String("raw", string(spec.Data.Raw())))
				time.Sleep(time.Second)
				continue
			}
			allLanes[lane.ID] = lane
		}
		if len(allLanes) == 0 && err != nil {
			log.Error(context.Background(), "get lane info config failed,not override old data!")
			continue
		}
		log.Info(context.Background(), "[lane] found new lane info,replace now!", zap.Any("rules", allLanes))
		l.mu.Lock()
		l.allLanes = allLanes
		l.mu.Unlock()

		l.refreshLanes()
		l.refreshRules()
	}
}

func (l *Lane) refreshLanes() {
	effectiveLanes := make(map[string]LaneInfo)
	namespaces := make(map[string]map[string]struct{})
	groups := make(map[string]map[string]struct{})
	for _, lane := range l.allLanes {
		for _, group := range lane.GroupList {
			if group.GroupID == env.GroupID() && group.Entrance {
				effectiveLanes[lane.ID] = lane
			}
			if !group.Entrance {
				sets := namespaces[group.NamespaceID]
				if sets == nil {
					sets = make(map[string]struct{})
				}
				sets[lane.ID] = struct{}{}
				namespaces[group.NamespaceID] = sets
			}
			gSets := groups[group.GroupID]
			if gSets == nil {
				gSets = make(map[string]struct{})
			}
			gSets[lane.ID] = struct{}{}
			groups[group.GroupID] = gSets
		}
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.lanes = effectiveLanes
	l.namespaces = namespaces
	l.groups = groups
	l.services = make(map[string]map[naming.Service]bool)
}

func (l *Lane) refreshRules() {
	var rules []LaneRule
	for _, lane := range l.lanes {
		for _, rule := range l.allRules {
			if rule.LaneID == lane.ID {
				rules = append(rules, rule)
			}
		}
	}
	sort.Slice(rules, func(i, j int) bool {
		if rules[i].Priority == rules[j].Priority {
			return rules[i].CreateTime.Before(rules[j].CreateTime)
		}
		return rules[i].Priority < rules[j].Priority
	})
	l.mu.Lock()
	defer l.mu.Unlock()
	l.rules = rules
}
