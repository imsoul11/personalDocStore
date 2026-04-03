package worker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/imsoul11/personalDocStore/internal/pkg/log"
	"github.com/imsoul11/personalDocStore/internal/pkg/queue/rabbitmq"
)

type Worker struct {
	broker rabbitmq.Broker
}

func New(broker rabbitmq.Broker) *Worker {
	return &Worker{
		broker: broker,
	}
}

func (w *Worker) Start(ctx context.Context) error {
	err := w.broker.RegisterTask("process_document", ProcessDocument)
	if err != nil {
		return fmt.Errorf("failed to register process_document: %w", err)
	}

	log.Logger().Info().Msg("tasks registered, starting worker")

	worker := w.broker.GetServer().NewWorker("docstore_worker", 5)
	return worker.Launch()
}

func ProcessDocument(filePath string) error {
	startTime := time.Now()
	log.Logger().Info().Str("file_path", filePath).Msg("starting document processing")
	
	// 1. Check if file exists
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		log.Logger().Error().Err(err).Str("file_path", filePath).Msg("file not found")
		return fmt.Errorf("file not found: %w", err)
	}

	fileSize := fileInfo.Size()
	log.Logger().Info().
		Str("file_path", filePath).
		Int64("size_bytes", fileSize).
		Msg("file found")

	// 2. Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		log.Logger().Error().Err(err).Str("file_path", filePath).Msg("failed to read file")
		return fmt.Errorf("failed to read file: %w", err)
	}

	// 3. Get file extension and validate
	ext := strings.ToLower(filepath.Ext(filePath))
	log.Logger().Info().
		Str("file_path", filePath).
		Str("extension", ext).
		Int("content_length", len(content)).
		Msg("file read successfully")

	// 4. Basic file type validation
	supportedTypes := []string{".pdf", ".txt", ".doc", ".docx", ".jpg", ".jpeg", ".png"}
	isSupported := false
	for _, supportedExt := range supportedTypes {
		if ext == supportedExt {
			isSupported = true
			break
		}
	}

	if !isSupported {
		log.Logger().Warn().Str("extension", ext).Msg("unsupported file type")
	}

	// 5. Simulate processing time (e.g., OCR, text extraction)
	log.Logger().Info().Msg("simulating document processing (5 seconds)...")
	time.Sleep(5 * time.Second)

	// 6. Move to processed directory
	processedDir := "./storage/processed"
	if err := os.MkdirAll(processedDir, 0755); err != nil {
		log.Logger().Error().Err(err).Msg("failed to create processed directory")
		return fmt.Errorf("failed to create processed directory: %w", err)
	}

	filename := filepath.Base(filePath)
	processedPath := filepath.Join(processedDir, filename)
	
	if err := os.Rename(filePath, processedPath); err != nil {
		log.Logger().Error().Err(err).Msg("failed to move file to processed")
		return fmt.Errorf("failed to move file: %w", err)
	}

	duration := time.Since(startTime)
	log.Logger().Info().
		Str("file_path", processedPath).
		Str("original_path", filePath).
		Dur("duration", duration).
		Int64("size_bytes", fileSize).
		Msg("document processing completed successfully")

	// TODO: Add more processing logic here:
	// - OCR for images/PDFs
	// - Text extraction
	// - Metadata extraction
	// - Update database status to "processed"
	// - Generate thumbnails
	// - Send notification to user
	
	return nil
}
