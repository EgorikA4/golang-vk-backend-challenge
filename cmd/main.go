package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/golang-vk-backend-challenge/internal/app"
	"github.com/golang-vk-backend-challenge/internal/config"
	"github.com/golang-vk-backend-challenge/internal/services/subpub"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	cfg := config.MustLoad()
	log := setupLogger(cfg.Env)
	log.Info("starting application",
		slog.String("env", cfg.Env),
		slog.Int("port", cfg.GRPC.Port),
	)

	sp := subpub.NewSubPub()
	application := app.New(log, cfg.GRPC.Port, sp)
	go application.MustRun()

	exitCtx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer func() {
		if err := recover(); err != nil {
			log.Warn("recovered from panic", slog.Any("error", err))
		}

		cancel()
		log.Info("stopping application")
		application.Stop(exitCtx)
		log.Info("application stopped")
	}()

	<-exitCtx.Done()
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envDev, envProd:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}
	return log
}
