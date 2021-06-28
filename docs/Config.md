### 分布式配置
#### 1. 引入配置模块
`"github.com/tencentyun/tsf-go/config"`

#### 2. 非阻塞获取某一个配置值
```go
if prefix, ok := config.GetString("prefix");ok {
	fmt.Println(prefix)
}
```
#### 3. 订阅某一个配置文件的变化
```go
type AppConfig struct {
	Stone string `yaml:"stone"`
	Auth  Auth   `yaml:"auth"`
}
type Auth struct {
	Key string `yaml:"key"`
}
config.WatchConfig(func(conf *config.Config) {
	var appCfg AppConfig
	err := conf.Unmarshal(&appCfg)
	if err != nil {
		panic(err)
	}
	fmt.Printf("appConfig: %v\n", appCfg)
})
```
> 更多 TSF 分布式配置的说明请参考 [配置管理概述](https://cloud.tencent.com/document/product/649/17956)。 