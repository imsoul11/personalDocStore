package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/imsoul11/personalDocStore/internal/pkg/log"
	"github.com/imsoul11/personalDocStore/internal/pkg/persistence"
	"github.com/imsoul11/personalDocStore/internal/pkg/queue/rabbitmq"
)

type Config struct {
	WorkerName      string
	Concurrency     int
	ProcessedDir    string
	ProcessingDelay time.Duration
}

type Worker struct {
	broker    rabbitmq.Broker
	store     *persistence.PGStore
	cfg       Config
	documents *DocumentsConsumer
}

func ResolveProcessingDelay(seconds int) time.Duration {
	if seconds <= 0 {
		seconds = 10
	}
	return time.Duration(seconds) * time.Second
}

func New(broker rabbitmq.Broker, store *persistence.PGStore, cfg Config) *Worker {
	return &Worker{
		broker: broker,
		store:  store,
		cfg:    cfg,
		// documents consumer uses cfg values
		documents: NewDocumentsConsumer(store, cfg.ProcessedDir, cfg.ProcessingDelay),
	}
}

func (w *Worker) RegisterTasks() error {
	if err := w.broker.RegisterTask("process_document", w.documents.ProcessDocument); err != nil {
		return fmt.Errorf("failed to register process_document: %w", err)
	}
	return nil
}

func (w *Worker) Start(ctx context.Context) error {
	if err := w.RegisterTasks(); err != nil {
		return err
	}

	workerName := w.cfg.WorkerName
	if workerName == "" {
		workerName = "docstore_worker"
	}
	concurrency := w.cfg.Concurrency
	if concurrency <= 0 {
		concurrency = 5
	}

	log.Logger().Info().
		Str("worker_name", workerName).
		Int("concurrency", concurrency).
		Msg("tasks registered, starting worker")

	mw := w.broker.GetServer().NewWorker(workerName, concurrency)
	return mw.Launch()
}
