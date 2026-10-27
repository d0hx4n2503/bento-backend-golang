package logger

import (
	"fmt"
	"hash/fnv"
	"path/filepath"
	"sort"
	"strings"
	"time"

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
	if encoding == "console" {
		base.EncoderConfig.EncodeLevel = colorLevelEncoder
		base.EncoderConfig.EncodeCaller = colorCallerEncoder
	}
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

	var baseLogger *zap.Logger
	if encoding == "console" {
		baseLogger, err = buildColorConsoleLogger(base)
	} else {
		baseLogger, err = base.Build(zap.AddCallerSkip(1))
	}
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

const ansiReset = "\x1b[0m"

func colorLevelEncoder(level zapcore.Level, encoder zapcore.PrimitiveArrayEncoder) {
	color := "\x1b[90m"
	switch level {
	case zapcore.InfoLevel:
		color = "\x1b[32m"
	case zapcore.WarnLevel:
		color = "\x1b[33m"
	case zapcore.ErrorLevel:
		color = "\x1b[31m"
	case zapcore.DPanicLevel:
		color = "\x1b[35m"
	case zapcore.PanicLevel, zapcore.FatalLevel:
		color = "\x1b[1;37;41m"
	}
	encoder.AppendString(color + level.CapitalString() + ansiReset)
}

func colorCallerEncoder(caller zapcore.EntryCaller, encoder zapcore.PrimitiveArrayEncoder) {
	callerPath := filepath.ToSlash(caller.TrimmedPath())
	if !caller.Defined {
		encoder.AppendString(callerPath)
		return
	}

	component := filepath.Base(filepath.Dir(callerPath))
	shortPath := filepath.Base(callerPath)
	if component != "." && component != "/" {
		shortPath = component + "/" + shortPath
	}

	color := callerColor(component)
	encoder.AppendString(color + fmt.Sprintf("%s:%d", shortPath, caller.Line) + ansiReset)
}

func callerColor(component string) string {
	switch component {
	case "app":
		return "\x1b[96m"
	case "auth":
		return "\x1b[92m"
	case "http":
		return "\x1b[93m"
	case "middleware":
		return "\x1b[95m"
	case "signin":
		return "\x1b[94m"
	}

	hash := fnv.New32a()
	_, _ = hash.Write([]byte(component))
	colors := [...]string{
		"\x1b[91m",
		"\x1b[92m",
		"\x1b[93m",
		"\x1b[94m",
		"\x1b[95m",
		"\x1b[96m",
	}
	return colors[hash.Sum32()%uint32(len(colors))]
}

func buildColorConsoleLogger(cfg zap.Config) (*zap.Logger, error) {
	output, _, err := zap.Open(cfg.OutputPaths...)
	if err != nil {
		return nil, fmt.Errorf("open logger output: %w", err)
	}
	errorOutput, _, err := zap.Open(cfg.ErrorOutputPaths...)
	if err != nil {
		return nil, fmt.Errorf("open logger error output: %w", err)
	}

	encoder := colorConsoleEncoder{Encoder: zapcore.NewConsoleEncoder(cfg.EncoderConfig)}
	core := zapcore.NewCore(encoder, output, cfg.Level)
	options := []zap.Option{zap.ErrorOutput(errorOutput), zap.AddCallerSkip(1)}
	if cfg.Development {
		options = append(options, zap.Development())
	}
	if !cfg.DisableCaller {
		options = append(options, zap.AddCaller())
	}
	stacktraceLevel := zapcore.ErrorLevel
	if cfg.Development {
		stacktraceLevel = zapcore.WarnLevel
	}
	if !cfg.DisableStacktrace {
		options = append(options, zap.AddStacktrace(stacktraceLevel))
	}
	if sampling := cfg.Sampling; sampling != nil {
		options = append(options, zap.WrapCore(func(core zapcore.Core) zapcore.Core {
			var samplingOptions []zapcore.SamplerOption
			if sampling.Hook != nil {
				samplingOptions = append(samplingOptions, zapcore.SamplerHook(sampling.Hook))
			}
			return zapcore.NewSamplerWithOptions(core, time.Second, sampling.Initial, sampling.Thereafter, samplingOptions...)
		}))
	}
	if len(cfg.InitialFields) > 0 {
		keys := make([]string, 0, len(cfg.InitialFields))
		for key := range cfg.InitialFields {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		fields := make([]zap.Field, 0, len(keys))
		for _, key := range keys {
			fields = append(fields, zap.Any(key, cfg.InitialFields[key]))
		}
		options = append(options, zap.Fields(fields...))
	}

	return zap.New(core, options...), nil
}
