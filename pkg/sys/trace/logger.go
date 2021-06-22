package trace

import (
	"github.com/tencentyun/tsf-go/pkg/sys/env"

	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// DefaultLogger is default trace logger
var DefaultLogger *zap.Logger

func init() {
	DefaultLogger = getLogger()
}

func getLogger() *zap.Logger {
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
