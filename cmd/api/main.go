package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/d0hx4n2503/bento-backend-golang/config"
	"github.com/d0hx4n2503/bento-backend-golang/internal/app"
)

func main() {
	filename := os.Getenv("APP_CONFIG")
	if filename == "" {
		filename = "config-local.yml"
	}

	viperConfig, err := config.LoadConfig(filename)
	if err != nil {
		log.Fatalf("load application configuration: %v", err)
	}
	cfg, err := config.ParseConfig(viperConfig)
	if err != nil {
		log.Fatalf("parse application configuration: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := app.Run(ctx, cfg); err != nil {
		log.Fatalf("run API process: %v", err)
	}
}
