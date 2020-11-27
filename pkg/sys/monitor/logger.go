package monitor

import (
	"github.com/natefinch/lumberjack"
	"github.com/tencentyun/tsf-go/pkg/sys/env"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger *zap.Logger

func init() {
	initMonitor()
}

func initMonitor() {
	path := env.MonitorPath()
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
	logger = zap.New(core)
}
