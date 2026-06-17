package worker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/imsoul11/personalDocStore/internal/pkg/log"
	"github.com/imsoul11/personalDocStore/internal/pkg/models"
	"github.com/imsoul11/personalDocStore/internal/pkg/persistence"
)

type DocumentsConsumer struct {
	store        *persistence.PGStore
	ProcessedDir string
	Delay        time.Duration
}

func NewDocumentsConsumer(store *persistence.PGStore, processedDir string, delay time.Duration) *DocumentsConsumer {
	return &DocumentsConsumer{
		store:        store,
		ProcessedDir: processedDir,
		Delay:        delay,
	}
}

// Machinery task handler: called when "process_document" runs
func (c *DocumentsConsumer) ProcessDocument(documentID string, filePath string) error {
	startTime := time.Now()
	log.Logger().Info().
		Str("document_id", documentID).
		Str("file_path", filePath).
		Msg("starting document processing")

	docID, err := strconv.ParseInt(documentID, 10, 64)
	if err != nil {
		log.Logger().Error().Err(err).Str("document_id", documentID).Msg("invalid document id")
		return fmt.Errorf("invalid document id: %w", err)
	}

	failDocument := func(cause error, message string) error {
		if c.store != nil {
			if statusErr := c.store.UpdateDocumentStatus(context.Background(), docID, models.StatusFailed); statusErr != nil {
				log.Logger().Error().Err(statusErr).Int64("document_id", docID).Msg("failed to set document status to failed")
			}
		}
		return fmt.Errorf("%s: %w", message, cause)
	}

	if c.store != nil {
		if err := c.store.UpdateDocumentStatus(context.Background(), docID, models.StatusProcessing); err != nil {
			log.Logger().Error().Err(err).Int64("document_id", docID).Msg("failed to set document status to processing")
		}
	}

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		log.Logger().Error().Err(err).Str("file_path", filePath).Msg("file not found")
		return failDocument(err, "file not found")
	}

	fileSize := fileInfo.Size()
	log.Logger().Info().
		Str("file_path", filePath).
		Int64("size_bytes", fileSize).
		Msg("file found")

	content, err := os.ReadFile(filePath)
	if err != nil {
		log.Logger().Error().Err(err).Str("file_path", filePath).Msg("failed to read file")
		return failDocument(err, "failed to read file")
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	log.Logger().Info().
		Str("file_path", filePath).
		Str("extension", ext).
		Int("content_length", len(content)).
		Msg("file read successfully")

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

	if c.Delay > 0 {
		log.Logger().Info().Dur("delay", c.Delay).Msg("simulating document processing delay")
		time.Sleep(c.Delay)
	}

	processedDir := c.ProcessedDir
	if processedDir == "" {
		processedDir = "./storage/processed"
	}
	if err := os.MkdirAll(processedDir, 0755); err != nil {
		log.Logger().Error().Err(err).Msg("failed to create processed directory")
		return failDocument(err, "failed to create processed directory")
	}

	filename := filepath.Base(filePath)
	processedPath := filepath.Join(processedDir, filename)

	if err := os.Rename(filePath, processedPath); err != nil {
		log.Logger().Error().Err(err).Msg("failed to move file to processed")
		return failDocument(err, "failed to move file")
	}

	if c.store != nil {
		if err := c.store.UpdateDocumentStatus(context.Background(), docID, models.StatusCompleted); err != nil {
			log.Logger().Error().Err(err).Int64("document_id", docID).Msg("failed to set document status to completed")
		}
	}

	log.Logger().Info().
		Str("file_path", processedPath).
		Str("original_path", filePath).
		Dur("duration", time.Since(startTime)).
		Int64("size_bytes", fileSize).
		Msg("document processing completed successfully")

	return nil
}
