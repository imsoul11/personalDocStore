package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/imsoul11/personalDocStore/internal/app/worker"
	"github.com/imsoul11/personalDocStore/internal/pkg/config"
	"github.com/imsoul11/personalDocStore/internal/pkg/db"
	pkglog "github.com/imsoul11/personalDocStore/internal/pkg/log"
	"github.com/imsoul11/personalDocStore/internal/pkg/persistence"
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

	dbInstance, err := db.New(cfg.Database)
	if err != nil {
		log.Fatalf("failed to connect db: %v", err)
	}
	store := persistence.New(dbInstance)

	broker := rabbitmq.New(ctx, cfg.RabbitMQ.URL)
	pkglog.Logger().Info().Msg("connected to rabbitmq")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		pkglog.Logger().Info().Msg("shutdown signal received, stopping worker")
		os.Exit(0)
	}()

	pkglog.Logger().Info().Msg("worker is running, press Ctrl+C to stop")

	w := worker.New(broker, store, worker.Config{
		WorkerName:      "docstore_worker",
		Concurrency:     5,
		ProcessedDir:    cfg.Storage.ProcessedPath,
		ProcessingDelay: 10 * time.Second,
	})

	err = w.Start(ctx)
	if err != nil {
		log.Fatalf("worker error: %v", err)
	}
}
