package logger

import (
	"fmt"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Config contains the options used to construct the Zap logger.
type Config struct {
	Development       bool
	DisableCaller     bool
	DisableStacktrace bool
	Encoding          string
	Level             string
}

// Logger is the logging surface used by application components.
type Logger interface {
	Debug(args ...any)
	Debugf(template string, args ...any)
	Info(args ...any)
	Infof(template string, args ...any)
	Warn(args ...any)
	Warnf(template string, args ...any)
	Error(args ...any)
	Errorf(template string, args ...any)
	Sync() error
}

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
		return nil, err
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
func (l *ZapLogger) Info(args ...any)                    { l.logger.Info(args...) }
func (l *ZapLogger) Infof(template string, args ...any)  { l.logger.Infof(template, args...) }
func (l *ZapLogger) Warn(args ...any)                    { l.logger.Warn(args...) }
func (l *ZapLogger) Warnf(template string, args ...any)  { l.logger.Warnf(template, args...) }
func (l *ZapLogger) Error(args ...any)                   { l.logger.Error(args...) }
func (l *ZapLogger) Errorf(template string, args ...any) { l.logger.Errorf(template, args...) }
func (l *ZapLogger) Sync() error                         { return l.logger.Sync() }
