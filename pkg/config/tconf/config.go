package tconf

import (
	"time"

	"github.com/tencentyun/tsf-go/log"
	"github.com/tencentyun/tsf-go/pkg/config"
)

var _ config.Config = &tsfConfig{}

type tsfConfig struct {
	v map[string]interface{}
	config.Data
}

func newTsfConfig(d config.Data) *tsfConfig {
	c := &tsfConfig{v: map[string]interface{}{}, Data: d}
	c.refill()
	return c
}

func (c *tsfConfig) Get(key string) (v interface{}, ok bool) {
	return c.get(key)
}

func (c *tsfConfig) GetString(key string) (v string, ok bool) {
	res, ok := c.get(key)
	if ok {
		v, ok = res.(string)
	}
	return
}

func (c *tsfConfig) GetBool(key string) (v bool, ok bool) {
	res, ok := c.get(key)
	if ok {
		v, ok = res.(bool)
	}
	return
}

func (c *tsfConfig) GetInt(key string) (v int64, ok bool) {
	res, ok := c.get(key)
	if ok {
		v, ok = res.(int64)
	}
	return
}

func (c *tsfConfig) GetFloat(key string) (v float64, ok bool) {
	res, ok := c.get(key)
	if ok {
		v, ok = res.(float64)
	}
	return
}

func (c *tsfConfig) GetDuration(key string) (v time.Duration, ok bool) {
	res, ok := c.get(key)
	if ok {
		v, ok = res.(time.Duration)
	}
	return
}

func (c *tsfConfig) GetTime(key string) (v time.Time, ok bool) {
	res, ok := c.get(key)
	if ok {
		v, ok = res.(time.Time)
	}
	return
}

func (c *tsfConfig) get(key string) (res interface{}, ok bool) {
	if c == nil {
		return
	}
	res, ok = c.v[key]
	return
}

func (c *tsfConfig) Unmarshal(v interface{}) error {
	if c == nil || c.Data == nil {
		return nil
	}
	return c.Data.Unmarshal(v)
}

func (c *tsfConfig) Raw() []byte {
	if c == nil || c.Data == nil {
		return nil
	}
	return c.Data.Raw()
}

func (c *tsfConfig) refill() {
	err := c.Data.Unmarshal(c.v)
	if err != nil {
		log.DefaultLog.Errorw("msg", "config refill failed!", "err", err, "raw", string(c.Raw()))
	}
}
