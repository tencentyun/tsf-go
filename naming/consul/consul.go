package consul

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-kratos/kratos/v2/registry"
	"github.com/tencentyun/tsf-go/naming"
	"github.com/tencentyun/tsf-go/pkg/http"
	"github.com/tencentyun/tsf-go/pkg/sys/env"
	"github.com/tencentyun/tsf-go/pkg/util"
)

var _ registry.Discovery = &Consul{}
var _ registry.Registrar = &Consul{}

var defaultConsul *Consul
var mu sync.Mutex

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
