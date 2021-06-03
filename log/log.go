package log

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/natefinch/lumberjack"
	"github.com/openzipkin/zipkin-go"
	"github.com/tencentyun/tsf-go/pkg/meta"
	"github.com/tencentyun/tsf-go/pkg/sys/env"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// DefaultLog is default tsf logger
	DefaultLog *log.Helper = log.NewHelper(NewLogger())
)

func newZap() *zap.Logger {
	var zapLogger *zap.Logger
	level := zap.NewAtomicLevelAt(zapcore.Level(env.LogLevel()))
	if env.LogPath() == "stdout" || env.LogPath() == "stderr" || env.LogPath() == "std" {
		var err error
		config := &zap.Config{
			Level:       level,
			Development: false,
			Encoding:    "console",
			EncoderConfig: zapcore.EncoderConfig{
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
			},
			OutputPaths:      []string{"stderr"},
			ErrorOutputPaths: []string{"stderr"},
		}
		zapLogger, err = config.Build(zap.AddCallerSkip(3))
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
		zapLogger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(3))
	}
	return zapLogger
}

type tsfLogger struct {
	logger *zap.Logger
	pool   *sync.Pool
}

// newTSFLogger new a logger with writer.
func newTSFLogger() log.Logger {
	return &tsfLogger{
		logger: newZap(),
		pool: &sync.Pool{
			New: func() interface{} {
				return new(bytes.Buffer)
			},
		},
	}
}

// Log print the kv pairs log.
func (l *tsfLogger) Log(level log.Level, keyvals ...interface{}) error {
	if len(keyvals) == 0 {
		return nil
	}
	if len(keyvals)%2 != 0 {
		keyvals = append(keyvals, "")
	}
	var trace string
	var msg string
	var newKvs []interface{}
	for i := 0; i < len(keyvals); i += 2 {
		if k, ok := keyvals[i].(string); ok {
			if k == "trace" {
				trace, _ = keyvals[i+1].(string)
			} else if k == "msg" {
				msg, _ = keyvals[i+1].(string)
			} else {
				newKvs = append(newKvs, keyvals[i])
				newKvs = append(newKvs, keyvals[i+1])
			}
		}
	}
	buf := l.pool.Get().(*bytes.Buffer)

	fmt.Fprintf(buf, "[%s] %s", trace, msg)
	for i := 0; i < len(newKvs); i += 2 {
		fmt.Fprintf(buf, " %s=%v", keyvals[i], keyvals[i+1])
	}
	if level == log.LevelDebug {
		l.logger.Debug(buf.String())
	} else if level == log.LevelInfo {
		l.logger.Info(buf.String())
	} else if level == log.LevelWarn {
		l.logger.Warn(buf.String())
	} else if level == log.LevelError {
		l.logger.Error(buf.String())
	}

	buf.Reset()
	l.pool.Put(buf)
	return nil
}

// NewLogger return tsf new logger
func NewLogger() log.Logger {
	logger := newTSFLogger()
	return log.With(logger, "trace", Trace())
}

// Trace returns a traceid valuer.
func Trace() log.Valuer {
	return func(ctx context.Context) interface{} {
		if ctx == nil {
			return ""
		}
		var serverName string
		if res := meta.Sys(ctx, meta.ServiceName); res != nil {
			serverName = res.(string)
		}

		span := zipkin.SpanFromContext(ctx)
		if span == nil {
			return fmt.Sprintf("%s,,,true", serverName)
		}
		traceID := span.Context().TraceID
		spanID := span.Context().ID

		return fmt.Sprintf("%s,%s,%s,true", serverName, traceID, spanID)
	}
}
