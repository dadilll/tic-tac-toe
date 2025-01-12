package logger

import (
	"context"
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ctxKey string

const (
	LoggerKey   ctxKey = "logger"
	ServiceName        = "service"
)

type Logger interface {
	Info(ctx context.Context, msg string, fields ...zap.Field)
	Error(ctx context.Context, msg string, fields ...zap.Field)
	Warn(ctx context.Context, msg string, fields ...zap.Field)
	Sync() error
}

type logger struct {
	serviceName string
	logger      *zap.Logger
}

func (l *logger) Info(ctx context.Context, msg string, fields ...zap.Field) {
	l.log(ctx, msg, fields, zapcore.InfoLevel)
}

func (l *logger) Error(ctx context.Context, msg string, fields ...zap.Field) {
	l.log(ctx, msg, fields, zapcore.ErrorLevel)
}

func (l *logger) Warn(ctx context.Context, msg string, fields ...zap.Field) {
	l.log(ctx, msg, fields, zapcore.WarnLevel)
}

func (l *logger) log(ctx context.Context, msg string, fields []zap.Field, level zapcore.Level) {
	fields = append(fields, zap.String(ServiceName, l.serviceName))
	switch level {
	case zapcore.InfoLevel:
		l.logger.Info(msg, fields...)
	case zapcore.ErrorLevel:
		l.logger.Error(msg, fields...)
	case zapcore.WarnLevel:
		l.logger.Warn(msg, fields...)
	}
}

func (l *logger) Sync() error {
	return l.logger.Sync()
}

var (
	defaultLogger Logger
	once          sync.Once
)

func New(serviceName string) Logger {
	once.Do(func() {
		encoderConfig := zapcore.EncoderConfig{
			TimeKey:        "time",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.CapitalColorLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		}

		core := zapcore.NewCore(
			zapcore.NewConsoleEncoder(encoderConfig),
			zapcore.Lock(os.Stdout),
			zapcore.DebugLevel,
		)

		zapLogger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
		defaultLogger = &logger{
			serviceName: serviceName,
			logger:      zapLogger,
		}
	})
	return defaultLogger
}

func GetLoggerFromCtx(ctx context.Context) Logger {
	if logger, ok := ctx.Value(LoggerKey).(Logger); ok {
		return logger
	}
	return defaultLogger
}
