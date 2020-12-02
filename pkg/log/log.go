package log

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/tencentyun/tsf-go/pkg/log/logger"
	tsfZap "github.com/tencentyun/tsf-go/pkg/log/zap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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

// L return default logger
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
		factory = &tsfZap.Builder{}
	}
	return factory.Build()
}

func Error(ctx context.Context, msg string, fields ...zap.Field) {
	L().Error(ctx, msg, fields...)
}

func Info(ctx context.Context, msg string, fields ...zap.Field) {
	L().Info(ctx, msg, fields...)
}

func Debug(ctx context.Context, msg string, fields ...zap.Field) {
	L().Debug(ctx, msg, fields...)
}

func Warn(ctx context.Context, msg string, fields ...zap.Field) {
	L().Warn(ctx, msg, fields...)
}

func Fatal(ctx context.Context, msg string, fields ...zap.Field) {
	L().Fatal(ctx, msg, fields...)
}

func Errorf(ctx context.Context, msg string, args ...interface{}) {
	L().Errorf(ctx, msg, args...)
}

func Infof(ctx context.Context, msg string, args ...interface{}) {
	L().Infof(ctx, msg, args...)
}

func Debugf(ctx context.Context, msg string, args ...interface{}) {
	L().Debugf(ctx, msg, args...)
}

func Warnf(ctx context.Context, msg string, args ...interface{}) {
	L().Warnf(ctx, msg, args...)
}

func Fatalf(ctx context.Context, msg string, args ...interface{}) {
	L().Fatalf(ctx, msg, args...)
}

// Sync calls the underlying Core's Sync method, flushing any buffered log entries.
// Applications should take care to call Sync before exiting
func Sync() error { return L().Sync() }

// SetLevel 设置输出端日志级别
func SetLevel(level zapcore.Level, output ...string) {
	L().SetLevel(level, output...)
}

// GetLevel 获取输出端日志级别
func GetLevel(output ...string) zapcore.Level {
	return L().GetLevel(output...)
}

// WithFields 设置一些业务自定义数据到每条log里:比如uid，imei等
func WithFields(fields ...zapcore.Field) logger.Logger {
	return L().WithFields(fields...)
}
