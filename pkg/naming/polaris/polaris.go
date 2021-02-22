package polaris

import (
	"context"
	"sync"
	"time"

	"github.com/tencentyun/tsf-go/pkg/log"
	"github.com/tencentyun/tsf-go/pkg/naming"
	"github.com/tencentyun/tsf-go/pkg/sys/env"

	"git.code.oa.com/polaris/polaris-go/api"
	"go.uber.org/zap"
)

type Config struct {
	ServiceToken string
}

type Polaris struct {
	cfg *Config
	api api.ProviderAPI
}

var defaultPolaris *Polaris
var mu sync.Mutex

func Default() *Polaris {
	mu.Lock()
	defer mu.Unlock()
	if defaultPolaris == nil {
		defaultPolaris = New(&Config{ServiceToken: env.ServiceToken()})
	}
	return defaultPolaris
}

func New(conf *Config) *Polaris {
	p := &Polaris{
		cfg: conf,
	}
	var err error
	p.api, err = api.NewProviderAPI()
	if err != nil {
		panic(err)
	}
	return p
}

func (p *Polaris) Register(ins *naming.Instance) (err error) {
	if ins.Metadata == nil {
		ins.Metadata = make(map[string]string)
	}
	ins.Metadata["ID"] = ins.ID

	request := &api.InstanceRegisterRequest{}
	request.Namespace = "Test"
	request.Service = ins.Service.Name
	request.ServiceToken = p.cfg.ServiceToken
	request.Host = ins.Host
	request.Port = ins.Port
	request.Metadata = ins.Metadata
	request.SetTTL(30)
	resp, err := p.api.Register(request)
	if nil != err {
		if err != nil {
			log.Error(context.Background(), "[polaris] register failed!", zap.String("id", ins.ID), zap.String("name", ins.Service.Name), zap.Error(err))
		}
		return
	}
	go func() {
		ticker := time.NewTicker(time.Second * 15)
		defer ticker.Stop()

		for {
			<-ticker.C
			beatReq := &api.InstanceHeartbeatRequest{}
			beatReq.Service = ins.Service.Name
			beatReq.Namespace = "Test"
			beatReq.Host = ins.Host
			beatReq.Port = 8080
			beatReq.ServiceToken = p.cfg.ServiceToken
			beatReq.InstanceID = resp.InstanceID
			err = p.api.Heartbeat(beatReq)
			if err != nil {
				log.Error(context.Background(), "[polaris] heartbeat failed!", zap.String("id", ins.ID), zap.String("p_id", resp.InstanceID), zap.String("name", ins.Service.Name), zap.Error(err))
				continue
			}
		}
	}()

	return
}

func (p *Polaris) Deregister(ins *naming.Instance) (err error) {
	return
}
