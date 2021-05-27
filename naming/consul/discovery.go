package consul

import (
	"context"
	"fmt"
	"math/rand"
	xhttp "net/http"
	"strconv"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/registry"
	"github.com/tencentyun/tsf-go/naming"
	"github.com/tencentyun/tsf-go/pkg/http"
	"github.com/tencentyun/tsf-go/pkg/log"
	"github.com/tencentyun/tsf-go/pkg/statusError"
	"github.com/tencentyun/tsf-go/pkg/sys/env"
	"github.com/tencentyun/tsf-go/pkg/util"

	"go.uber.org/zap"
)

// Used to return information about a node
type Node struct {
	ID              string
	Node            string
	Address         string
	Datacenter      string
	TaggedAddresses map[string]string
	Meta            map[string]string

	RaftIndex
}

// RaftIndex is used to track the index used while creating
// or modifying a given struct type.
type RaftIndex struct {
	CreateIndex uint64
	ModifyIndex uint64
}

// NodeService is a service provided by a node
type NodeService struct {
	ID                string
	Service           string
	Tags              []string
	Address           string
	Meta              map[string]string
	Port              int
	EnableTagOverride bool

	RaftIndex
}

type CheckServiceNode struct {
	Node    *Node
	Service *NodeService
	Checks  HealthChecks
}

func (self *CheckServiceNode) compare(node *CheckServiceNode) (euqal bool) {
	if self.Service.Address != node.Service.Address {
		return
	} else if self.Service.ID != node.Service.ID {
		return
	} else if self.Service.Port != node.Service.Port {
		return
	}
	euqal = true
	return
}

func compareNodes(old, new []CheckServiceNode) (euqal bool) {
	for len(old) != len(new) {
		return
	}
	for _, node := range new {
		ok := false
		for _, oldNode := range old {
			if oldNode.compare(&node) {
				ok = true
			}
		}
		if !ok {
			return
		}
	}
	euqal = true
	return
}

type HealthChecks []*HealthCheck

// HealthCheck represents a single check on a given node
type HealthCheck struct {
	Node        string
	CheckID     string   // Unique per-node ID
	Name        string   // Check name
	Status      string   // The current check status
	Notes       string   // Additional notes with the status
	Output      string   // Holds output of script runs
	ServiceID   string   // optional associated service
	ServiceName string   // optional service name
	ServiceTags []string // optional service tags

	RaftIndex
}

type CheckType struct {
	// fields already embedded in CheckDefinition
	// Note: CheckType.CheckID == CheckDefinition.ID

	CheckID string
	Name    string
	Status  string
	Notes   string

	// fields copied to CheckDefinition
	// Update CheckDefinition when adding fields here

	ScriptArgs        []string
	HTTP              string
	Header            map[string][]string
	Method            string
	TCP               string
	Interval          time.Duration
	DockerContainerID string
	Shell             string
	TLSSkipVerify     bool
	Timeout           time.Duration
	TTL               time.Duration

	// DeregisterCriticalServiceAfter, if >0, will cause the associated
	// service, if any, to be deregistered if this check is critical for
	// longer than this duration.
	DeregisterCriticalServiceAfter time.Duration
}

type CheckTypes []*CheckType

type ServiceDefinition struct {
	ID                string
	Name              string
	Tags              []string
	Address           string
	Meta              map[string]string
	Port              int
	Check             CheckType
	Checks            CheckTypes
	Token             string
	EnableTagOverride bool
}

type Config struct {
	Address []string
	Token   string
	// additional message:tsf namespaceid and tencent appid if exsist
	AppID       string
	NamespaceID string

	Catalog bool
}

type Consul struct {
	queryCli  *http.Client
	setCli    *http.Client
	bc        *util.BackoffConfig
	registry  map[string]*insInfo
	discovery map[naming.Service]*svcInfo
	lock      sync.RWMutex

	conf *Config
}

func (c *Consul) addr() string {
	if len(c.conf.Address) == 0 {
		return ""
	} else if len(c.conf.Address) == 1 {
		return c.conf.Address[0]
	}

	return c.conf.Address[rand.Intn(len(c.conf.Address))]
}

func (c *Consul) catalog(index int64) (services map[string]interface{}, consulIndex int64, err error) {
	url := fmt.Sprintf("http://%s/v1/catalog/services?token=%s&wait=55s&index=%d", c.addr(), c.conf.Token, index)
	if c.conf.NamespaceID != "" {
		url += "&nid=" + c.conf.NamespaceID
	}
	if c.conf.AppID != "" {
		url += "&uid=" + c.conf.AppID
	}
	defer func() {
		if err != nil {
			log.Error(context.Background(), "[naming] get catalog failed!", zap.String("url", url), zap.Error(err))
		}
	}()
	var header xhttp.Header
	services = map[string]interface{}{}
	header, err = c.queryCli.Get(url, &services)
	if err != nil {
		if statusError.IsNotFound(err) {
			err = nil
		} else {
			return
		}
	}
	if header != nil {
		str := header.Get("X-Consul-Index")
		consulIndex, err = strconv.ParseInt(str, 10, 64)
		if err != nil {
			err = statusError.New(500, "consul index invalid: %s", str)
			return
		}
	} else {
		err = statusError.New(500, "consul index invalid,no http header found!")
		return
	}
	return
}

func (c *Consul) healthService(svc naming.Service, index int64) (nodes []CheckServiceNode, consulIndex int64, err error) {
	url := fmt.Sprintf("http://%s/v1/health/service/%s?token=%s&passing&wait=55s&index=%d", c.addr(), svc.Name, c.conf.Token, index)
	/*if svc.NameSpace == "global" {
		url += "&nsType=GLOBAL"
	} else if svc.NameSpace == "all" {
		// not supported now!
		//url += "&nsType=DEF_AND_GLOBAL"
	} else if svc.NameSpace == "local" {
		url += "&nsType=DEF"
	}*/
	if svc.Namespace != "" && svc.Namespace != env.NamespaceID() {
		if svc.Namespace == naming.NsGlobal {
			url += "&nsType=GLOBAL"
		} else {
			url += "&nid=" + svc.Namespace
		}
	} else if c.conf.NamespaceID != "" {
		url += "&nid=" + c.conf.NamespaceID
	}
	if c.conf.AppID != "" {
		url += "&uid=" + c.conf.AppID
	}
	defer func() {
		if err != nil {
			log.Error(context.Background(), "[naming] get healthService failed!", zap.String("name", svc.Name), zap.String("url", url), zap.Error(err))
		}
	}()
	var header xhttp.Header
	header, err = c.queryCli.Get(url, &nodes)
	if err != nil {
		if statusError.IsNotFound(err) {
			err = nil
		} else {
			return
		}
	}
	if header != nil {
		str := header.Get("X-Consul-Index")
		consulIndex, err = strconv.ParseInt(str, 10, 64)
		if err != nil {
			err = statusError.New(500, "consul index invalid: %s", str)
			return
		}
	} else {
		err = statusError.New(500, "consul index invalid,no http header found!")
		return
	}
	return
}

func (c *Consul) Catalog() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	var lastIndex int64
	retries := 0
	for {
		<-ticker.C
		_, index, err := c.catalog(lastIndex)
		if err != nil {
			time.Sleep(c.bc.Backoff(retries))
			retries++
			continue
		}
		retries = 0
		lastIndex = index
	}
}

func (c *Consul) Watch(ctx context.Context, service string) (registry.Watcher, error) {
	svc := naming.Service{Name: service}
	w := &Watcher{
		event: make(chan struct{}, 1),
	}
	w.ctx, w.cancel = context.WithCancel(context.Background())
	c.lock.Lock()
	defer c.lock.Unlock()
	v, ok := c.discovery[svc]
	if !ok {
		v = c.newService(svc)
	} else {
		nodes, _ := v.nodes.Load().([]*registry.ServiceInstance)
		if len(nodes) > 0 {
			// watcher初始化的时候至少一个slot，所以肯定可以非阻塞推送成功
			w.event <- struct{}{}
		}
	}
	w.svc = v
	v.watcher[w] = struct{}{}
	return w, nil
}

func (c *Consul) newService(svc naming.Service) *svcInfo {
	v := &svcInfo{watcher: make(map[*Watcher]struct{}, 0), consul: c, info: svc}
	var ctx context.Context
	ctx, v.cancel = context.WithCancel(context.Background())
	c.discovery[svc] = v
	go v.subscribe(ctx, svc)
	return v
}

// GetService is get service
func (c *Consul) GetService(ctx context.Context, service string) (nodes []*registry.ServiceInstance, err error) {
	svc := naming.Service{Name: service}
	c.lock.RLock()
	v, ok := c.discovery[svc]
	c.lock.RUnlock()
	if !ok {
		c.lock.Lock()
		if v, ok = c.discovery[svc]; !ok {
			c.newService(svc)
		}
		c.lock.Unlock()
		return
	}
	nodes, ok = v.nodes.Load().([]*registry.ServiceInstance)
	if !ok {
		return nil, fmt.Errorf("not found ")
	}
	return
}

func (s *svcInfo) subscribe(ctx context.Context, svc naming.Service) {
	var (
		lastNodes []CheckServiceNode
		lastIndex int64
		err       error
	)

	lastNodes, lastIndex, err = s.consul.healthService(svc, lastIndex)
	if err == nil && len(lastNodes) > 0 {
		s.broadcast(lastNodes)
	}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	retries := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			nodes, index, err := s.
				consul.healthService(svc, lastIndex)
			if err != nil {
				time.Sleep(s.consul.bc.Backoff(retries))
				retries++
				continue
			}
			if len(nodes) != 0 {
				if index != lastIndex || !compareNodes(lastNodes, nodes) {
					lastNodes = nodes
					s.broadcast(nodes)
				}
			}
			retries = 0
			lastIndex = index
		}
	}
}

func (s *svcInfo) broadcast(nodes []CheckServiceNode) {
	s.store(nodes)
	s.consul.lock.RLock()
	defer s.consul.lock.RUnlock()
	for k := range s.watcher {
		select {
		case k.event <- struct{}{}:
		default:
		}
	}
}

func (s *svcInfo) store(nodes []CheckServiceNode) {
	var inss []*registry.ServiceInstance
	for _, node := range nodes {
		var ins = naming.Instance{
			Service:  &naming.Service{Namespace: node.Service.Meta[naming.NamespaceID], Name: node.Service.Service},
			ID:       node.Service.ID,
			Host:     node.Service.Address,
			Port:     node.Service.Port,
			Metadata: node.Service.Meta,
			Status:   naming.StatusUp,
		}

		inss = append(inss, ins.ToKratosInstance())
	}
	s.nodes.Store(inss)
}

type Watcher struct {
	event chan struct{}
	svc   *svcInfo
	// for cancel
	ctx    context.Context
	cancel context.CancelFunc
}

func (w *Watcher) Next() (nodes []*registry.ServiceInstance, err error) {
	select {
	case <-w.ctx.Done():
		err = statusError.ClientClosed("")
		return
	case <-w.event:
		nodes = w.svc.nodes.Load().([]*registry.ServiceInstance)
	}
	return
}

func (w *Watcher) Stop() error {
	select {
	case <-w.ctx.Done():
		return nil
	default:
	}
	w.cancel()
	w.svc.consul.lock.Lock()
	defer w.svc.consul.lock.Unlock()
	delete(w.svc.watcher, w)
	if len(w.svc.watcher) == 0 {
		delete(w.svc.consul.discovery, w.svc.info)
		w.svc.cancel()
	}
	return nil
}

func uniName(svc naming.Service) string {
	return fmt.Sprintf("%s/%s", svc.Namespace, svc.Name)
}

func checkID(ins *naming.Instance) string {
	return fmt.Sprintf("service:%s", ins.ID)
}
