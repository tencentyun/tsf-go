package tracing

import (
	"github.com/natefinch/lumberjack"
	"github.com/tencentyun/tsf-go/pkg/sys/env"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// defaultLogger is default trace logger
var defaultLogger *zap.Logger

func newLogger() *zap.Logger {
	path := env.TracePath()
	encoding := zapcore.EncoderConfig{
		TimeKey:        "",
		LevelKey:       "",
		NameKey:        "",
		CallerKey:      "",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.EpochTimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	w := zapcore.AddSync(&lumberjack.Logger{
		Filename:   path,
		MaxSize:    100, // megabytes
		MaxBackups: 3,
		MaxAge:     10, // days
	})
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoding),
		w,
		zapcore.Level(zap.InfoLevel),
	)
	return zap.New(core)
}

func init() {
	defaultLogger = newLogger()
}
