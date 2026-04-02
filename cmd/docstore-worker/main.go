package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/imsoul11/personalDocStore/internal/app/worker"
	"github.com/imsoul11/personalDocStore/internal/pkg/config"
	pkglog "github.com/imsoul11/personalDocStore/internal/pkg/log"
	"github.com/imsoul11/personalDocStore/internal/pkg/queue/rabbitmq"
)

func main() {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "configs/config.json"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	pkglog.New(cfg.Log.Level)
	pkglog.Logger().Info().Msg("docstore worker starting")

	ctx := context.Background()

	broker := rabbitmq.New(ctx, cfg.RabbitMQ.URL)
	pkglog.Logger().Info().Msg("connected to rabbitmq")

	w := worker.New(broker)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		pkglog.Logger().Info().Msg("shutdown signal received, stopping worker")
		os.Exit(0)
	}()

	pkglog.Logger().Info().Msg("worker is running, press Ctrl+C to stop")

	err = w.Start(ctx)
	if err != nil {
		log.Fatalf("worker error: %v", err)
	}
}
