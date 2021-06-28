package main

import (
	"flag"
	"fmt"

	"github.com/tencentyun/tsf-go/config"
)

type AppConfig struct {
	Stone string `yaml:"stone"`
	Auth  Auth   `yaml:"auth"`
}
type Auth struct {
	Key string `yaml:"key"`
}

func main() {
	flag.Parse()
	cfg := config.GetConfig()
	value, _ := cfg.GetString("stone")
	fmt.Println("stone:", value)
	// 监听应用配置文件变化
	config.WatchConfig(func(conf *config.Config) {
		var appCfg AppConfig
		err := conf.Unmarshal(&appCfg)
		if err != nil {
			panic(err)
		}
		fmt.Printf("appConfig: %v\n", appCfg)
	})
	// 订阅全局配置（命名空间纬度）
	config.WatchConfig(func(conf *config.Config) {
		var gloablCfg AppConfig
		err := conf.Unmarshal(&gloablCfg)
		if err != nil {
			panic(err)
		}
		fmt.Printf("globalConfig: %v\n", gloablCfg)
	}, config.WithGlobal(true))
}
