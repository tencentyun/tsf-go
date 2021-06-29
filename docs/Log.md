# TSF日志输出
## Quick Start
#### 1. 初始化log helper
```go
import 	"github.com/tencentyun/tsf-go/log"

// 默认日志配置项，等同于log.NewLogger()
// 当没有显示配置日志输出地址时，运行在tsf平台时默认会输出到/data/logs/root.log
logger := log.DefaultLogger

log := log.NewHelper(logger)
```
#### 2. 打印日志
```go
// 打印format后的日志
log.Infof("app started!date: %v", time.Now())
// 打印key value pair日志
log.Infow("msg", "welcome to tsf world!", "name", "tsf")
// 注入上下文中trace id等信息至日志中
log.WithContext(ctx).Infof("get request message!")
```
#### 3.TSF 控制台日志配置
需要在 TSF [日志服务]-[日志配置] 中新建一个配置，并绑定部署组并发布
配置日志类型为自定义 Logback
日志格式为 `%d{yyyy-MM-dd HH:mm:ss.SSS} %level %thread %trace %msg%n`
采集路径为/data/logs/root.log
> 更多 TSF 日志配置可参考 [日志服务说明](https://cloud.tencent.com/document/product/649/18196)。


## 配置参数说明
1. WithLevel
   配置日志显示等级，默认为Info
   也可通过环境变量tsf_log_level来控制
2. WithTrace
   是否开启Trace信息，默认为true
   如果打印日志时不通过WithContext传递Go的context，会导致日志中不打印traceID。
3. WithPath
   日志输出路径，运行在tsf平台时默认为/data/logs/root.log
   运行在本地环境时默认是stdout
   也可通过环境变量tsf_log_path来控制
4. WithZap
   替换整个logger核心组件