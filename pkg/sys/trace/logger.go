package trace

import (
	"github.com/tencentyun/tsf-go/pkg/internal/env"

	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger *zap.Logger

func init() {
	logger = getLogger()
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
