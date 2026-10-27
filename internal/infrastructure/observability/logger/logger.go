package logger

// Logger is the logging surface shared by application components.
type Logger interface {
	Debug(args ...any)
	Debugf(template string, args ...any)
	Debugw(message string, keysAndValues ...any)
	Info(args ...any)
	Infof(template string, args ...any)
	Infow(message string, keysAndValues ...any)
	Warn(args ...any)
	Warnf(template string, args ...any)
	Warnw(message string, keysAndValues ...any)
	Error(args ...any)
	Errorf(template string, args ...any)
	Errorw(message string, keysAndValues ...any)
	Fatal(args ...any)
	Fatalf(template string, args ...any)
	Sync() error
}
