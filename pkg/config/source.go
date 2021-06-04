package config

import (
	"context"
)

type Spec struct {
	Key  string
	Data Data
}

type Data interface {
	Unmarshal(interface{}) error
	Raw() []byte
}

// Source is config interface
type Source interface {
	// 如果path是以/结尾，则是目录，否则当作key处理
	Subscribe(path string) Watcher
	Get(ctx context.Context, path string) []Spec
}

// Watcher is topic watch
type Watcher interface {
	// Watch 第一次访问的时候如果有值则立马返回spec；后面watch的时候有变更才返回
	// 如果不传key则watch整个文件变动
	// 如果超时或者Watcher被Close应当抛出错误
	Watch(ctx context.Context) ([]Spec, error)
	Close()
}
