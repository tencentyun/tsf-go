package log

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/natefinch/lumberjack"
	"github.com/tencentyun/tsf-go/pkg/meta"
	"github.com/tencentyun/tsf-go/pkg/sys/env"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Level int8

const (
	LevelDebug Level = iota - 1
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

type Option func(t *options)

func WithLevel(l Level) Option {
	return func(t *options) {
		t.level = l
	}
}

func WithZap(logger *zap.Logger) Option {
	return func(t *options) {
		t.logger = logger
	}
}

func WithPath(path string) Option {
	return func(t *options) {
		t.path = path
	}
}

func WithTrace(enable bool) Option {
	return func(t *options) {
		t.traceEnable = enable
	}
}

var (
	// DefaultLog is default tsf logger
	DefaultLogger log.Logger  = NewLogger(WithTrace(true), WithPath(env.LogPath()), WithLevel(Level(env.LogLevel())))
	DefaultLog    *log.Helper = log.NewHelper(DefaultLogger)
)

func newZap(path string) *zap.Logger {
	var zapLogger *zap.Logger
	level := zap.NewAtomicLevelAt(zapcore.DebugLevel)
	if path == "stdout" || path == "stderr" || path == "std" {
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
				StacktraceKey:  "",
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
				StacktraceKey:  "",
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

type options struct {
	level       Level
	logger      *zap.Logger
	path        string
	traceEnable bool
}

type tsfLogger struct {
	level  Level
	logger *zap.Logger
	pool   *sync.Pool
}

// Log print the kv pairs log.
func (l *tsfLogger) Log(level log.Level, keyvals ...interface{}) error {
	if len(keyvals) == 0 {
		return nil
	}
	if int8(l.level) > int8(level) {
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
		fmt.Fprintf(buf, " %s=%v", newKvs[i], newKvs[i+1])
	}
	if level == log.LevelDebug {
		l.logger.Debug(buf.String())
	} else if level == log.LevelInfo {
		l.logger.Info(buf.String())
	} else if level == log.LevelWarn {
		l.logger.Warn(buf.String())
	} else if level == log.LevelError {
		l.logger.Error(buf.String())
	} else if level == log.LevelFatal {
		l.logger.Fatal(buf.String())
	}

	buf.Reset()
	l.pool.Put(buf)
	return nil
}

// NewLogger return tsf new logger
func NewLogger(opts ...Option) log.Logger {
	o := options{
		level:       Level(env.LogLevel()),
		path:        env.LogPath(),
		traceEnable: true,
	}
	for _, opt := range opts {
		opt(&o)
	}
	if o.logger == nil {
		o.logger = newZap(o.path)
	}
	logger := &tsfLogger{
		logger: o.logger,
		pool: &sync.Pool{
			New: func() interface{} {
				return new(bytes.Buffer)
			},
		},
		level: o.level,
	}
	if o.traceEnable {
		return log.With(logger, "trace", Trace())
	}
	return logger
}

// NewHelper return tsf new logger helper
func NewHelper(l log.Logger) *log.Helper {
	return log.NewHelper(l)
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
		var traceID string
		var spanID string
		if span := trace.SpanContextFromContext(ctx); span.HasTraceID() {
			traceID = span.TraceID().String()
			spanID = span.SpanID().String()
		}

		return fmt.Sprintf("%s,%s,%s,true", serverName, traceID, spanID)
	}
}
