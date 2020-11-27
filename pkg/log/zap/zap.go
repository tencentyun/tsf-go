package zap

import (
	"context"
	"fmt"

	"github.com/tencentyun/tsf-go/pkg/internal/env"
	"github.com/tencentyun/tsf-go/pkg/log/logger"

	"github.com/natefinch/lumberjack"
	"github.com/openzipkin/zipkin-go"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	_ logger.Logger        = &Logger{}
	_ logger.LoggerFactory = &Builder{}
)

type Builder struct {
}

func (b *Builder) Build() logger.Logger {
	var zapLogger *zap.Logger
	level := zap.NewAtomicLevelAt(zapcore.Level(env.LogLevel()))
	if env.LogPath() == "stdout" || env.LogPath() == "stderr" || env.LogPath() == "std" {
		var err error
		config := &zap.Config{
			Level:            level,
			Development:      false,
			Encoding:         "console",
			EncoderConfig:    zap.NewDevelopmentEncoderConfig(),
			OutputPaths:      []string{"stderr"},
			ErrorOutputPaths: []string{"stderr"},
		}
		zapLogger, err = config.Build(zap.AddCallerSkip(1))
		if err != nil {
			panic(fmt.Errorf("logger build failed!err:=%v", err))
		}
	} else {
		w := zapcore.AddSync(&lumberjack.Logger{
			Filename:   env.LogPath(),
			MaxSize:    20, // megabytes
			MaxBackups: 10,
			MaxAge:     10, // days
		})
		core := zapcore.NewCore(
			zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
				// Keys can be anything except the empty string.
				TimeKey:        "ts",
				LevelKey:       "level",
				NameKey:        "logger",
				CallerKey:      "caller",
				FunctionKey:    zapcore.OmitKey,
				MessageKey:     "msg",
				StacktraceKey:  "stacktrace",
				LineEnding:     zapcore.DefaultLineEnding,
				EncodeLevel:    zapcore.CapitalLevelEncoder,
				EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.999"),
				EncodeDuration: zapcore.SecondsDurationEncoder,
				EncodeCaller:   zapcore.ShortCallerEncoder,
			}),
			w,
			level,
		)
		zapLogger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	}
	return &Logger{zapLogger, level}
}

func (b *Builder) Schema() string {
	return "zap"
}

type Logger struct {
	*zap.Logger
	level zap.AtomicLevel
}

func genFields(ctx context.Context, fields ...zap.Field) string {
	span := zipkin.SpanFromContext(ctx)
	if span == nil {
		return ""
	}
	return span.Context().TraceID.String() + " "
}

func (l *Logger) Error(ctx context.Context, msg string, fields ...zap.Field) {
	l.Logger.Error(genFields(ctx)+msg, fields...)
}

func (l *Logger) Info(ctx context.Context, msg string, fields ...zap.Field) {

	l.Logger.Info(genFields(ctx)+msg, fields...)
}

func (l *Logger) Debug(ctx context.Context, msg string, fields ...zap.Field) {
	l.Logger.Debug(genFields(ctx)+msg, fields...)
}

func (l *Logger) Warn(ctx context.Context, msg string, fields ...zap.Field) {
	l.Logger.Warn(genFields(ctx)+msg, fields...)
}

func (l *Logger) Fatal(ctx context.Context, msg string, fields ...zap.Field) {
	l.Logger.Fatal(genFields(ctx)+msg, fields...)
}

func (l *Logger) Errorf(ctx context.Context, msg string, args ...interface{}) {
	l.Logger.Sugar().Errorf(genFields(ctx)+msg, args...)
}

func (l *Logger) Infof(ctx context.Context, msg string, args ...interface{}) {
	l.Logger.Sugar().Infof(genFields(ctx)+msg, args...)
}

func (l *Logger) Debugf(ctx context.Context, msg string, args ...interface{}) {
	l.Logger.Sugar().Debugf(genFields(ctx)+msg, args...)
}

func (l *Logger) Warnf(ctx context.Context, msg string, args ...interface{}) {
	l.Logger.Sugar().Warnf(genFields(ctx)+msg, args...)
}

func (l *Logger) Fatalf(ctx context.Context, msg string, args ...interface{}) {
	l.Logger.Sugar().Fatalf(genFields(ctx)+msg, args...)
}

func (l *Logger) GetLevel(output ...string) zapcore.Level {
	return l.level.Level()
}

func (l *Logger) SetLevel(level zapcore.Level, output ...string) {
	l.level.SetLevel(level)
}

func (l *Logger) Sync() error {
	return l.Logger.Sync()
}

func (l *Logger) WithFields(fields ...zapcore.Field) logger.Logger {
	zapLogger := l.Logger.With(fields...)
	return &Logger{zapLogger, l.level}
}
