package consul

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/tencentyun/tsf-go/log"
	"github.com/tencentyun/tsf-go/naming"
)

func (c *Consul) Register(ctx context.Context, ki *registry.ServiceInstance) (err error) {
	for _, ins := range naming.FromKratosInstance(ki) {
		err := c.registerIns(ins)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Consul) registerIns(ins *naming.Instance) (err error) {
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
		c.heartBeat(ins)
		timer := time.NewTimer(time.Second * 20)
		defer timer.Stop()
		retries := 0
		for {
			select {
			case <-ctx.Done():
				log.DefaultLog.Debugw("msg", "[naming] recevie exit signal,quit register!", "id", ins.ID, "name", ins.Service.Name)
				return
			case <-timer.C:
				err = c.heartBeat(ins)
				if err != nil {
					if errors.IsNotFound(err) || errors.IsInternalServer(err) {
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

func (c *Consul) Deregister(ctx context.Context, ki *registry.ServiceInstance) (err error) {
	for _, ins := range naming.FromKratosInstance(ki) {
		err := c.deregister(ins)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Consul) deregisterIns(ins *naming.Instance) (err error) {
	log.DefaultLog.Infow("msg", "deregister service!", "svc", ins.Service.Name)
	c.lock.RLock()
	v, ok := c.registry[ins.ID]
	c.lock.RUnlock()
	if ok && v != nil {
		v.cancel()
	}
	c.deregister(ins)
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
		log.DefaultLog.Errorw("msg", "[naming] register instance to consul failed!", "instance", sd, "url", url, "err", err)
	} else {
		log.DefaultLog.Infow("msg", "[naming] register instance to consul success!", "instance", sd, "url", url)
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
		log.DefaultLog.Errorw("msg", "[naming] send heartbeat to consul failed!", "id", ins.ID, "url", url, "err", err)
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
		log.DefaultLog.Errorw("msg", "[naming] deregister ins to consul failed!", "id", ins.ID, "url", url, "err", err)
	}
	return
}
