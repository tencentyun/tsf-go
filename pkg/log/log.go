package log

import (
	"sync"
	"sync/atomic"

	"github.com/tencentyun/tsf-go/pkg/log/logger"
	"github.com/tencentyun/tsf-go/pkg/log/zap"
)

var factory logger.LoggerFactory
var gLogger logger.Logger
var lock sync.Mutex
var initialized int32

func Register(f logger.LoggerFactory) {
	lock.Lock()
	defer lock.Unlock()
	factory = f
}

func L() logger.Logger {
	if atomic.LoadInt32(&initialized) == 0 {
		lock.Lock()
		defer lock.Unlock()
		if atomic.LoadInt32(&initialized) == 0 {
			gLogger = getLogger()
		}
		atomic.StoreInt32(&initialized, 1)
	}
	return gLogger
}

func getLogger() logger.Logger {
	if factory == nil {
		// default logger: uber.go/zap
		factory = &zap.Builder{}
	}
	return factory.Build()
}
