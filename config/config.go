package config

import (
	"time"

	"github.com/tencentyun/tsf-go/log"
	"github.com/tencentyun/tsf-go/pkg/config"
)

var _ config.Config = &Config{}

// Config is tsf config
type Config struct {
	v map[string]interface{}
	config.Data
}

func newTsfConfig(d config.Data) *Config {
	c := &Config{v: map[string]interface{}{}, Data: d}
	c.refill()
	return c
}

func (c *Config) Get(key string) (v interface{}, ok bool) {
	return c.get(key)
}

func (c *Config) GetString(key string) (v string, ok bool) {
	res, ok := c.get(key)
	if ok {
		v, ok = res.(string)
	}
	return
}

func (c *Config) GetBool(key string) (v bool, ok bool) {
	res, ok := c.get(key)
	if ok {
		v, ok = res.(bool)
	}
	return
}

func (c *Config) GetInt(key string) (v int64, ok bool) {
	res, ok := c.get(key)
	if ok {
		v, ok = res.(int64)
	}
	return
}

func (c *Config) GetFloat(key string) (v float64, ok bool) {
	res, ok := c.get(key)
	if ok {
		v, ok = res.(float64)
	}
	return
}

func (c *Config) GetDuration(key string) (v time.Duration, ok bool) {
	res, ok := c.get(key)
	if ok {
		v, ok = res.(time.Duration)
	}
	return
}

func (c *Config) GetTime(key string) (v time.Time, ok bool) {
	res, ok := c.get(key)
	if ok {
		v, ok = res.(time.Time)
	}
	return
}

func (c *Config) get(key string) (res interface{}, ok bool) {
	if c == nil {
		return
	}
	res, ok = c.v[key]
	return
}

func (c *Config) Unmarshal(v interface{}) error {
	if c == nil || c.Data == nil {
		return nil
	}
	return c.Data.Unmarshal(v)
}

func (c *Config) Raw() []byte {
	if c == nil || c.Data == nil {
		return nil
	}
	return c.Data.Raw()
}

func (c *Config) refill() {
	err := c.Data.Unmarshal(c.v)
	if err != nil {
		log.DefaultLog.Errorw("msg", "config refill failed!", "err", err, "raw", string(c.Raw()))
	}
}
