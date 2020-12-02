package tconf

// tconf is tsf remote config

import (
	"context"
	"fmt"
	"sync"

	"github.com/tencentyun/tsf-go/pkg/config"
	"github.com/tencentyun/tsf-go/pkg/config/consul"
	"github.com/tencentyun/tsf-go/pkg/log"
	"github.com/tencentyun/tsf-go/pkg/sys/env"
	"github.com/tencentyun/tsf-go/pkg/util"
	"go.uber.org/zap"
)

var mu sync.RWMutex
var inited bool

var global *Config
var app *Config

var globalFunc []func(conf *Config)
var appFunc []func(conf *Config)

var globalWatcher config.Watcher
var appWatcher config.Watcher

type Config struct {
	v map[string]interface{}
	config.Data
}

func (c *Config) Get(key string) (res interface{}, ok bool) {
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
		log.Error(context.Background(), "config refill failed!", zap.Error(err), zap.String("raw", string(c.Raw())))
	}
}

// 需要提前初始化，否则可能获取不到数据
func Init(ctx context.Context) error {
	util.ParseFlag()
	mu.Lock()
	defer mu.Unlock()
	if inited {
		return nil
	}
	appWatcher = consul.DefaultConsul().Subscribe(fmt.Sprintf("config/application/%s/%s/data", env.ApplicationID(), env.GroupID()))
	globalWatcher = consul.DefaultConsul().Subscribe(fmt.Sprintf("config/application/%s/data", env.NamespaceID()))

	appSpecs, err := appWatcher.Watch(ctx)
	if err != nil {
		return err
	}
	gloablSpecs, err := globalWatcher.Watch(ctx)
	if err != nil {
		return err
	}
	inited = true
	if len(appSpecs) > 0 {
		app = &Config{v: map[string]interface{}{}, Data: appSpecs[0].Data}
		app.refill()
	}
	if len(gloablSpecs) > 0 {
		global = &Config{v: map[string]interface{}{}, Data: gloablSpecs[0].Data}
		global.refill()
	}

	go refreshApp()
	go refreshGlobal()
	return nil
}

func refreshGlobal() {
	ctx := context.Background()
	for {
		specs, err := globalWatcher.Watch(ctx)
		if err != nil {
			log.Error(ctx, "refreshGlobal Watch failed!", zap.Error(err))
			return
		}
		var conf *Config
		if len(specs) > 0 {
			conf = &Config{v: map[string]interface{}{}, Data: specs[0].Data}
			conf.refill()
		}
		mu.Lock()
		global = conf
		mu.Unlock()
		go func() {
			for _, f := range globalFunc {
				f(conf)
			}
		}()
	}
}

func refreshApp() {
	ctx := context.Background()
	for {
		specs, err := appWatcher.Watch(ctx)
		if err != nil {
			log.Error(ctx, "refreshApp Watch failed!", zap.Error(err))
			return
		}
		var conf *Config
		if len(specs) > 0 {
			conf = &Config{v: map[string]interface{}{}, Data: specs[0].Data}
			conf.refill()
		}
		mu.Lock()
		app = conf
		mu.Unlock()
		go func() {
			for _, f := range appFunc {
				f(conf)
			}
		}()
	}
}

// GlobalConfig 订阅全局配置文件的变化
func GlobalConfig(f func(conf *Config)) {
	f(global)
	mu.Lock()
	defer mu.Unlock()
	globalFunc = append(globalFunc, f)
}

// AppConfig 订阅应用级别配置文件的变化
func AppConfig(f func(conf *Config)) {
	f(app)
	mu.Lock()
	defer mu.Unlock()
	appFunc = append(appFunc, f)
}

// GetApp 非阻塞获取应用配置的key value
func GetApp(key string) (res interface{}, ok bool) {
	mu.RLock()
	defer mu.RUnlock()
	if app != nil {
		res, ok = app.Get(key)
	}
	return
}

// GetGlobal 非阻塞获取全局配置的key value
func GetGlobal(key string) (res interface{}, ok bool) {
	mu.RLock()
	defer mu.RUnlock()
	if global != nil {
		res, ok = global.Get(key)
	}
	return
}
