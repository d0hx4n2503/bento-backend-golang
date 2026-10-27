package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/d0hx4n2503/bento-backend-golang/config"
	"github.com/d0hx4n2503/bento-backend-golang/internal/app"
	"github.com/d0hx4n2503/bento-backend-golang/internal/infrastructure/observability/logger"
)

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	bootstrapLogger, err := logger.NewZapLogger(logger.Config{
		DisableCaller:     true,
		DisableStacktrace: true,
		Encoding:          "console",
		Level:             "info",
	})
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "initialize bootstrap logger: %v\n", err)
		return err
	}
	defer func() { _ = bootstrapLogger.Sync() }()

	filename := os.Getenv("APP_CONFIG")
	if filename == "" {
		filename = "config-local.yml"
	}

	viperConfig, err := config.LoadConfig(filename)
	if err != nil {
		bootstrapLogger.Errorw("failed to load application configuration", "path", filename, "error", err)
		return fmt.Errorf("load application configuration: %w", err)
	}
	cfg, err := config.ParseConfig(viperConfig)
	if err != nil {
		bootstrapLogger.Errorw("failed to parse application configuration", "error", err)
		return fmt.Errorf("parse application configuration: %w", err)
	}
	applicationLogger, err := logger.NewZapLogger(logger.Config{
		Development:       cfg.Logger.Development,
		DisableCaller:     cfg.Logger.DisableCaller,
		DisableStacktrace: cfg.Logger.DisableStacktrace,
		Encoding:          cfg.Logger.Encoding,
		Level:             cfg.Logger.Level,
	})
	if err != nil {
		bootstrapLogger.Errorw("failed to initialize application logger", "error", err)
		return fmt.Errorf("initialize application logger: %w", err)
	}
	defer func() { _ = applicationLogger.Sync() }()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := app.Run(ctx, cfg, applicationLogger); err != nil {
		applicationLogger.Errorw("API process failed", "error", err)
		return fmt.Errorf("run API process: %w", err)
	}
	return nil
}
