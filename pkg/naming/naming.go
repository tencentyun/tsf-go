package naming

import (
	"context"
	"strconv"

	"github.com/tencentyun/tsf-go/pkg/sys/env"
)

const (
	StatusUp   = 0
	StatusDown = 1

	GroupID       = "TSF_GROUP_ID"
	NamespaceID   = "TSF_NAMESPACE_ID"
	ApplicationID = "TSF_APPLICATION_ID"

	NsLocal  = "local"
	NsGlobal = "global"
)

// Service 服务信息
type Service struct {
	Namespace string
	Name      string
}

func NewService(namespace string, name string) Service {
	if namespace == "" || namespace == NsLocal {
		namespace = env.NamespaceID()
	}
	return Service{Namespace: namespace, Name: name}
}

// Instance 服务实例信息
type Instance struct {
	// 服务信息
	Service *Service `json:"service,omitempty"`
	// namespace下全局唯一的实例ID
	ID string `json:"id"`
	// 服务实例所属地域
	Region string `json:"region"`
	// 服务实例可访问的ip地址
	Host string `json:"addrs"`
	// 协议端口
	Port int `json:"port"`
	// 服务实例标签元信息,比如appVersion、group、weight等
	Metadata map[string]string `json:"metadata"`
	// 实例运行状态: up/down
	Status int64 `json:"status"`
	// 过滤用的标签
	Tags []string
}

func (i Instance) Addr() string {
	return i.Host + ":" + strconv.FormatInt(int64(i.Port), 10)
}

// Discovery 服务发现
type Discovery interface {
	// 根据namespace,service name非阻塞获取服务信息，并返回是否初始化过
	Fetch(svc Service) ([]Instance, bool)
	// 根据namespace,service name订阅服务信息，直到服务有更新或超时返回(如果超时则success=false)
	Subscribe(svc Service) Watcher
	// discovery Scheme
	Scheme() string
}

// Watcher 消息订阅
type Watcher interface {
	//第一次访问的时候如果有值立马返回cfg，后面watch的时候有变更才返回
	//如果超时或者被Close应当抛出错误
	Watch(ctx context.Context) ([]Instance, error)
	//如果不需要使用则关闭订阅
	Close()
}

// Registry 注册中心
type Registry interface {
	// 注册实例
	Register(ins *Instance) error
	// 注销实例
	Deregister(ins *Instance) error
}

// Builder resolver builder.
type Builder interface {
	Build(id string, options ...BuildOpt) Watcher
	Scheme() string
}

// BuildOptions build options.
type BuildOptions struct {
}

// BuildOpt build option interface.
type BuildOpt interface {
	Apply(*BuildOptions)
}

type funcOpt struct {
	f func(*BuildOptions)
}

func (f *funcOpt) Apply(opt *BuildOptions) {
	f.f(opt)
}
