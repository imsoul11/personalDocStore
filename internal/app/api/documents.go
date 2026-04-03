package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"
	pkglog "github.com/imsoul11/personalDocStore/internal/pkg/log"
	"github.com/imsoul11/personalDocStore/internal/pkg/models"
	"github.com/imsoul11/personalDocStore/internal/pkg/persistence"
	"github.com/imsoul11/personalDocStore/internal/pkg/queue/rabbitmq"
	"github.com/imsoul11/personalDocStore/restapi/operations"
)

type DocIMPL struct {
	store  *persistence.PGStore
	broker rabbitmq.Broker
}

func NewDocuments(store *persistence.PGStore, broker rabbitmq.Broker) *DocIMPL {
	return &DocIMPL{
		store:  store,
		broker: broker,
	}
}

// GetDocuments implements the document list handler for the API server.
func (d *DocIMPL) GetDocuments(ctx context.Context, params operations.GetDocumentsParams, principal interface{}) middleware.Responder {
	log := pkglog.Logger()
	if d.store == nil {
		log.Error().Str("op", "get_documents").Msg("store not initialized")
		return middleware.ResponderFunc(func(rw http.ResponseWriter, _ runtime.Producer) {
			rw.WriteHeader(http.StatusInternalServerError)
		})
	}
	userID, ok := principal.(int64)
	if !ok || principal == nil {
		log.Warn().Str("op", "get_documents").Msg("unauthorized request")
		return operations.NewGetDocumentsUnauthorized()
	}
	log.Info().Str("op", "get_documents").Int64("user_id", userID).Msg("fetching user documents")
	docs, err := d.store.GetDocumentByUserID(ctx, userID)
	if err != nil {
		log.Error().Str("op", "get_documents").Int64("user_id", userID).Err(err).Msg("failed to fetch documents")
		return middleware.ResponderFunc(func(rw http.ResponseWriter, _ runtime.Producer) {
			rw.WriteHeader(http.StatusInternalServerError)
		})
	}
	log.Info().Str("op", "get_documents").Int64("user_id", userID).Int("documents_count", len(docs)).Msg("documents fetched")
	
	_ = docs // response body not in spec yet; return OK
	_ = params
	return operations.NewGetDocumentsOK()
}

func (d *DocIMPL) PostDocuments(ctx context.Context, params operations.PostDocumentsParams, principal interface{}) middleware.Responder {
	log := pkglog.Logger()
	
	userID, ok := principal.(int64)
	if !ok || principal == nil {
		log.Warn().Str("op", "post_documents").Msg("unauthorized request")
		return operations.NewPostDocumentsUnauthorized()
	}

	filename := ""
	if params.Filename != nil {
		filename = *params.Filename
	}
	if filename == "" {
		filename = fmt.Sprintf("doc_%d_%d", userID, time.Now().Unix())
	}

	log.Info().Str("op", "post_documents").Int64("user_id", userID).Str("filename", filename).Msg("document upload request")

	// Create upload directory if it doesn't exist
	uploadDir := "./storage/uploads"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		log.Error().Err(err).Msg("failed to create upload directory")
		return operations.NewPostDocumentsBadRequest()
	}

	// Save file to disk
	filePath := filepath.Join(uploadDir, filename)
	outFile, err := os.Create(filePath)
	if err != nil {
		log.Error().Err(err).Str("path", filePath).Msg("failed to create file")
		return operations.NewPostDocumentsBadRequest()
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, params.File)
	if err != nil {
		log.Error().Err(err).Msg("failed to save file")
		return operations.NewPostDocumentsBadRequest()
	}

	log.Info().Str("op", "post_documents").Str("path", filePath).Msg("file saved to disk")

	// Save document metadata to database
	doc := &models.Document{
		UserID:    userID,
		Filename:  filename,
		Status:    "uploaded",
		CreatedAt: time.Now(),
	}
	if err := d.store.CreateDocument(ctx, doc); err != nil {
		log.Error().Err(err).Msg("failed to save document metadata")
		return operations.NewPostDocumentsBadRequest()
	}

	log.Info().Str("op", "post_documents").Int64("document_id", doc.ID).Msg("document metadata saved")

	// Enqueue task to process the document
	err = d.broker.EnqueueTask("process_document", filePath)
	if err != nil {
		log.Error().Err(err).Msg("failed to enqueue task")
		return operations.NewPostDocumentsBadRequest()
	}

	log.Info().Str("op", "post_documents").Str("filename", filename).Msg("document processing task enqueued")
	
	return operations.NewPostDocumentsCreated()
}
