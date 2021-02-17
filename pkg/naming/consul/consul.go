package consul

import (
	"context"
	"fmt"
	"math/rand"
	xhttp "net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tencentyun/tsf-go/pkg/http"
	"github.com/tencentyun/tsf-go/pkg/log"
	"github.com/tencentyun/tsf-go/pkg/naming"
	"github.com/tencentyun/tsf-go/pkg/statusError"
	"github.com/tencentyun/tsf-go/pkg/sys/env"
	"github.com/tencentyun/tsf-go/pkg/util"

	"go.uber.org/zap"
)

var (
	_ naming.Discovery = &Consul{}
	_ naming.Registry  = &Consul{}
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

type insInfo struct {
	ins    *naming.Instance
	cancel context.CancelFunc
}

type svcInfo struct {
	info    naming.Service
	nodes   atomic.Value
	watcher map[*Watcher]struct{}
	cancel  context.CancelFunc
	consul  *Consul
}

var defaultConsul *Consul
var mu sync.Mutex

func DefaultConsul() *Consul {
	mu.Lock()
	defer mu.Unlock()
	if defaultConsul == nil {
		defaultConsul = New(&Config{Address: env.ConsulAddressList(), Token: env.Token()})
	}
	return defaultConsul
}

func New(conf *Config) *Consul {
	c := &Consul{
		queryCli:  http.NewClient(http.WithTimeout(time.Second * 120)),
		setCli:    http.NewClient(http.WithTimeout(time.Second * 30)),
		registry:  make(map[string]*insInfo),
		discovery: make(map[naming.Service]*svcInfo),
		bc: &util.BackoffConfig{
			MaxDelay:  25 * time.Second,
			BaseDelay: 500 * time.Millisecond,
			Factor:    1.5,
			Jitter:    0.2,
		},
		conf: conf,
	}
	if conf != nil && conf.Catalog {
		go c.Catalog()
	}
	return c
}

func (c *Consul) Scheme() string {
	return "consul"
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
		if svc.Name == naming.NsGlobal {
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

func (c *Consul) register(ins *naming.Instance) (err error) {
	sd := &ServiceDefinition{
		ID:      ins.ID,
		Name:    ins.Service.Name,
		Address: ins.Host,
		Meta:    ins.Metadata,
		Port:    ins.Port,
		Check: CheckType{
			CheckID: checkID(ins),
			TTL:     time.Second * 40,
		},
		Tags: ins.Tags,
	}
	url := fmt.Sprintf("http://%s/v1/agent/service/register?token=%s", c.addr(), c.conf.Token)
	if c.conf.NamespaceID != "" {
		url += "&nid=" + c.conf.NamespaceID
	}
	if c.conf.AppID != "" {
		url += "&uid=" + c.conf.AppID
	}
	err = c.setCli.Put(url, sd, nil)
	if err != nil {
		log.Error(context.Background(), "[naming] register instance to consul failed!", zap.Any("instance", sd), zap.String("url", url), zap.Error(err))
	} else {
		log.Info(context.Background(), "[naming] register instance to consul success!", zap.Any("instance", sd), zap.String("url", url))
	}
	return
}

func (c *Consul) heartBeat(ins *naming.Instance) (err error) {
	url := fmt.Sprintf("http://%s/v1/agent/check/pass/%s?token=%s", c.addr(), checkID(ins), c.conf.Token)
	if c.conf.NamespaceID != "" {
		url += "&nid=" + c.conf.NamespaceID
	}
	if c.conf.AppID != "" {
		url += "&uid=" + c.conf.AppID
	}
	err = c.setCli.Put(url, nil, nil)
	if err != nil {
		log.Error(context.Background(), "[naming] send heartbeat to consul failed!", zap.String("id", ins.ID), zap.String("url", url), zap.Error(err))
	}
	return
}

func (c *Consul) deregister(ins *naming.Instance) (err error) {
	url := fmt.Sprintf("http://%s/v1/agent/service/deregister/%s?token=%s", c.addr(), ins.ID, c.conf.Token)
	if c.conf.NamespaceID != "" {
		url += "&nid=" + c.conf.NamespaceID
	}
	if c.conf.AppID != "" {
		url += "&uid=" + c.conf.AppID
	}
	err = c.setCli.Put(url, nil, nil)
	if err != nil {
		log.Error(context.Background(), "[naming] deregister ins to consul failed!", zap.String("id", ins.ID), zap.String("url", url), zap.Error(err))
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

func (c *Consul) Register(ins *naming.Instance) (err error) {
	c.lock.Lock()
	if _, ok := c.registry[ins.ID]; ok {
		c.lock.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	c.registry[ins.ID] = &insInfo{
		ins:    ins,
		cancel: cancel,
	}
	c.lock.Unlock()

	err = c.register(ins)
	if err != nil {
		return
	}
	go func() {
		timer := time.NewTimer(time.Second * 20)
		defer timer.Stop()
		retries := 0
		for {
			select {
			case <-ctx.Done():
				log.Debug(ctx, "[naming] recevie exit signal,quit register!", zap.String("id", ins.ID), zap.String("name", ins.Service.Name))
				return
			case <-timer.C:
				err = c.heartBeat(ins)
				if err != nil {
					if statusError.IsNotFound(err) || statusError.IsInternal(err) {
						time.Sleep(time.Millisecond * 500)
						// 如果注册中心报错500或者404，则重新注册
						err = c.register(ins)
					}
					if err != nil {
						timer.Reset(c.bc.Backoff(retries))
						retries++
						continue
					}
				}
				timer.Reset(time.Second * 20)
				retries = 0
			}
		}
	}()
	return
}

func (c *Consul) Deregister(ins *naming.Instance) (err error) {
	log.Info(context.Background(), "deregister service!", zap.String("svc", ins.Service.Name))
	c.lock.RLock()
	v, ok := c.registry[ins.ID]
	c.lock.RUnlock()
	if ok && v != nil {
		v.cancel()
	}
	c.deregister(ins)
	return
}

func (c *Consul) Subscribe(svc naming.Service) naming.Watcher {
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
		nodes, _ := v.nodes.Load().([]naming.Instance)
		if len(nodes) > 0 {
			// watcher初始化的时候至少一个slot，所以肯定可以非阻塞推送成功
			w.event <- struct{}{}
		}
	}
	w.svc = v
	v.watcher[w] = struct{}{}
	return w
}

func (c *Consul) newService(svc naming.Service) *svcInfo {
	v := &svcInfo{watcher: make(map[*Watcher]struct{}, 0), consul: c, info: svc}
	var ctx context.Context
	ctx, v.cancel = context.WithCancel(context.Background())
	c.discovery[svc] = v
	go v.subscribe(ctx, svc)
	return v
}

func (c *Consul) Fetch(svc naming.Service) (nodes []naming.Instance, initialized bool) {
	c.lock.RLock()
	v, ok := c.discovery[svc]
	c.lock.RUnlock()
	initialized = ok
	if !ok {
		c.lock.Lock()
		if v, ok = c.discovery[svc]; !ok {
			c.newService(svc)
		}
		c.lock.Unlock()
		return
	}
	nodes, _ = v.nodes.Load().([]naming.Instance)
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
	var inss []naming.Instance
	for _, node := range nodes {
		svc := naming.NewService(s.info.Namespace, s.info.Name)
		var ins = naming.Instance{
			Service:  &svc,
			ID:       node.Service.ID,
			Host:     node.Service.Address,
			Port:     node.Service.Port,
			Metadata: node.Service.Meta,
			Status:   naming.StatusUp,
		}
		inss = append(inss, ins)
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

func (w *Watcher) Watch(ctx context.Context) (nodes []naming.Instance, err error) {
	select {
	case <-ctx.Done():
		err = statusError.Deadline("")
		return
	case <-w.ctx.Done():
		err = statusError.ClientClosed("")
		return
	case <-w.event:
		nodes = w.svc.nodes.Load().([]naming.Instance)
	}
	return
}

func (w *Watcher) Close() {
	select {
	case <-w.ctx.Done():
		return
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
}

func uniName(svc naming.Service) string {
	return fmt.Sprintf("%s/%s", svc.Namespace, svc.Name)
}

func checkID(ins *naming.Instance) string {
	return fmt.Sprintf("service:%s", ins.ID)
}
