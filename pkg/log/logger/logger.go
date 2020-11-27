package logger

import (
	"context"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger interface {
	Error(ctx context.Context, msg string, fields ...zap.Field)
	Info(ctx context.Context, msg string, fields ...zap.Field)
	Debug(ctx context.Context, msg string, fields ...zap.Field)
	Warn(ctx context.Context, msg string, fields ...zap.Field)
	Fatal(ctx context.Context, msg string, fields ...zap.Field)

	Errorf(ctx context.Context, msg string, args ...interface{})
	Infof(ctx context.Context, msg string, args ...interface{})
	Debugf(ctx context.Context, msg string, args ...interface{})
	Warnf(ctx context.Context, msg string, args ...interface{})
	Fatalf(ctx context.Context, msg string, args ...interface{})

	// Sync calls the underlying Core's Sync method, flushing any buffered log entries.
	// Applications should take care to call Sync before exiting
	Sync() error

	// SetLevel 设置输出端日志级别
	SetLevel(level zapcore.Level, output ...string)

	// GetLevel 获取输出端日志级别
	GetLevel(output ...string) zapcore.Level

	// WithFields 设置一些业务自定义数据到每条log里:比如uid，imei等
	WithFields(...zapcore.Field) Logger
}

type LoggerFactory interface {
	Build() Logger
	Schema() string
}
