package logger

// Config contains the options used to construct the Zap logger.
type Config struct {
	Development       bool
	DisableCaller     bool
	DisableStacktrace bool
	Encoding          string
	Level             string
}
