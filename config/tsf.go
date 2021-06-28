package config

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/tencentyun/tsf-go/log"
	"github.com/tencentyun/tsf-go/pkg/config"
	"github.com/tencentyun/tsf-go/pkg/config/consul"
	"github.com/tencentyun/tsf-go/pkg/sys/env"
	"github.com/tencentyun/tsf-go/pkg/util"
)

var mu sync.RWMutex
var once sync.Once

var global *Config
var app *Config

var globalFunc []func(conf *Config)
var appFunc []func(conf *Config)

// Init 需要提前初始化，否则可能获取不到数据
func Init() {
	util.ParseFlag()
	source := consul.DefaultConsul()
	appWatcher := source.Subscribe(fmt.Sprintf("config/application/%s/%s/data", env.ApplicationID(), env.GroupID()))
	globalWatcher := source.Subscribe(fmt.Sprintf("config/application/%s/data", env.NamespaceID()))

	appSpecs, err := appWatcher.Watch(context.Background())
	if err != nil {
		log.DefaultLog.Errorf("config watch failed!err:=%v", err)
		return
	}
	gloablSpecs, err := globalWatcher.Watch(context.Background())
	if err != nil {
		log.DefaultLog.Errorf("config watch failed!err:=%v", err)
		return
	}
	if len(appSpecs) > 0 {
		app = newTsfConfig(appSpecs[0].Data)
	}
	if len(gloablSpecs) > 0 {
		global = newTsfConfig(gloablSpecs[0].Data)
	}

	go refreshApp(appWatcher)
	go refreshGlobal(globalWatcher)
}

func refreshGlobal(globalWatcher config.Watcher) {
	ctx := context.Background()
	for {
		specs, err := globalWatcher.Watch(ctx)
		if err != nil {
			log.DefaultLog.Errorw("msg", "refreshGlobal Watch failed!", "err", err)
			return
		}
		var conf *Config
		if len(specs) > 0 {
			conf = newTsfConfig(specs[0].Data)
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

func refreshApp(appWatcher config.Watcher) {
	ctx := context.Background()
	for {
		specs, err := appWatcher.Watch(ctx)
		if err != nil {
			log.DefaultLog.Errorw("msg", "refreshApp Watch failed!", "err", err)
			return
		}
		var conf *Config
		if len(specs) > 0 {
			conf = newTsfConfig(specs[0].Data)
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

// GetConfig 获取配置文件
func GetConfig(opts ...Option) *Config {
	return getCfg(opts...)
}

// WatchConfig 订阅配置文件的变化，如果非空则第一次必推送
// 可能推送nil config
func WatchConfig(f func(conf *Config), opts ...Option) {
	once.Do(Init)
	var opt options
	for _, o := range opts {
		o(&opt)
	}
	mu.Lock()
	defer mu.Unlock()
	if opt.isGlobal {
		if global != nil {
			f(global)
		}
		globalFunc = append(globalFunc, f)
		return
	}
	if app != nil {
		f(app)
	}
	appFunc = append(appFunc, f)
	return
}

func getCfg(opts ...Option) *Config {
	once.Do(Init)
	var cfg *Config
	var opt options
	for _, o := range opts {
		o(&opt)
	}
	mu.RLock()
	defer mu.RUnlock()
	if opt.isGlobal {
		cfg = global
	} else {
		cfg = app
	}
	return cfg
}

// Get 非阻塞获取配置的key value
func Get(key string, opts ...Option) (res interface{}, ok bool) {
	cfg := getCfg(opts...)
	if cfg != nil {
		return cfg.Get(key)
	}
	return
}

func GetString(key string, opts ...Option) (v string, ok bool) {
	cfg := getCfg(opts...)
	if cfg != nil {
		return cfg.GetString(key)
	}
	return
}

func GetBool(key string, opts ...Option) (v bool, ok bool) {
	cfg := getCfg(opts...)
	if cfg != nil {
		return cfg.GetBool(key)
	}
	return
}

func GetInt(key string, opts ...Option) (v int64, ok bool) {
	cfg := getCfg(opts...)
	if cfg != nil {
		return cfg.GetInt(key)
	}
	return
}

func GetFloat(key string, opts ...Option) (v float64, ok bool) {
	cfg := getCfg(opts...)
	if cfg != nil {
		return cfg.GetFloat(key)
	}
	return
}

func GetDuration(key string, opts ...Option) (v time.Duration, ok bool) {
	cfg := getCfg(opts...)
	if cfg != nil {
		return cfg.GetDuration(key)
	}
	return
}

func GetTime(key string, opts ...Option) (v time.Time, ok bool) {
	cfg := getCfg(opts...)
	if cfg != nil {
		return cfg.GetTime(key)
	}
	return
}

type options struct {
	isGlobal bool
}

// WithGlobal is with global config
func WithGlobal(isGlobal bool) Option {
	return func(o *options) {
		o.isGlobal = isGlobal
	}
}

// Option is config client option.
type Option func(*options)
