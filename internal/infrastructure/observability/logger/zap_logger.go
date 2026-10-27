package logger

import (
	"fmt"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ZapLogger adapts Zap's sugared logger to the application logger interface.
type ZapLogger struct {
	logger *zap.SugaredLogger
}

// NewZapLogger creates a configured logger. The caller should call Sync during
// process shutdown to flush buffered output.
func NewZapLogger(cfg Config) (*ZapLogger, error) {
	levelName := strings.ToLower(strings.TrimSpace(cfg.Level))
	if levelName == "" {
		levelName = "info"
	}
	level, err := zapcore.ParseLevel(levelName)
	if err != nil {
		return nil, fmt.Errorf("parse log level %q: %w", levelName, err)
	}

	base := zap.NewProductionConfig()
	if cfg.Development {
		base = zap.NewDevelopmentConfig()
	}

	encoding := strings.ToLower(strings.TrimSpace(cfg.Encoding))
	if encoding == "" {
		encoding = "json"
	}
	if encoding != "json" && encoding != "console" {
		return nil, fmt.Errorf("unsupported log encoding %q: expected json or console", cfg.Encoding)
	}
	base.Encoding = encoding
	base.Level = zap.NewAtomicLevelAt(level)
	base.DisableCaller = cfg.DisableCaller
	base.DisableStacktrace = cfg.DisableStacktrace
	base.EncoderConfig.LevelKey = "level"
	base.EncoderConfig.CallerKey = "caller"
	base.EncoderConfig.TimeKey = "time"
	base.EncoderConfig.NameKey = "logger"
	base.EncoderConfig.MessageKey = "message"
	base.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	base.OutputPaths = []string{"stderr"}
	base.ErrorOutputPaths = []string{"stderr"}

	baseLogger, err := base.Build(zap.AddCallerSkip(1))
	if err != nil {
		return nil, fmt.Errorf("build zap logger: %w", err)
	}
	return &ZapLogger{logger: baseLogger.Sugar()}, nil
}

func (l *ZapLogger) Debug(args ...any)                   { l.logger.Debug(args...) }
func (l *ZapLogger) Debugf(template string, args ...any) { l.logger.Debugf(template, args...) }
func (l *ZapLogger) Debugw(message string, keysAndValues ...any) {
	l.logger.Debugw(message, keysAndValues...)
}
func (l *ZapLogger) Info(args ...any)                   { l.logger.Info(args...) }
func (l *ZapLogger) Infof(template string, args ...any) { l.logger.Infof(template, args...) }
func (l *ZapLogger) Infow(message string, keysAndValues ...any) {
	l.logger.Infow(message, keysAndValues...)
}
func (l *ZapLogger) Warn(args ...any)                   { l.logger.Warn(args...) }
func (l *ZapLogger) Warnf(template string, args ...any) { l.logger.Warnf(template, args...) }
func (l *ZapLogger) Warnw(message string, keysAndValues ...any) {
	l.logger.Warnw(message, keysAndValues...)
}
func (l *ZapLogger) Error(args ...any)                   { l.logger.Error(args...) }
func (l *ZapLogger) Errorf(template string, args ...any) { l.logger.Errorf(template, args...) }
func (l *ZapLogger) Errorw(message string, keysAndValues ...any) {
	l.logger.Errorw(message, keysAndValues...)
}
func (l *ZapLogger) Fatal(args ...any)                   { l.logger.Fatal(args...) }
func (l *ZapLogger) Fatalf(template string, args ...any) { l.logger.Fatalf(template, args...) }
func (l *ZapLogger) Sync() error                         { return l.logger.Sync() }

var _ Logger = (*ZapLogger)(nil)
