package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
	pkglog "github.com/imsoul11/personalDocStore/internal/pkg/log"
	intmodels "github.com/imsoul11/personalDocStore/internal/pkg/models"
	"github.com/imsoul11/personalDocStore/internal/pkg/persistence"
	"github.com/imsoul11/personalDocStore/internal/pkg/queue/rabbitmq"
	swgmodels "github.com/imsoul11/personalDocStore/models"
	"github.com/imsoul11/personalDocStore/restapi/operations"
)

type DocIMPL struct {
	store     *persistence.PGStore
	broker    rabbitmq.Broker
	uploadDir string
}

func NewDocuments(store *persistence.PGStore, broker rabbitmq.Broker, uploadDir string) *DocIMPL {
	return &DocIMPL{
		store:     store,
		broker:    broker,
		uploadDir: resolveUploadDir(uploadDir),
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

	_ = params

	success := true
	ack := "Documents fetched successfully"
	respDocs := make([]*swgmodels.Document, 0, len(docs))
	for _, doc := range docs {
		if swaggerDoc := swaggerDocumentFromModel(doc); swaggerDoc != nil {
			respDocs = append(respDocs, swaggerDoc)
		}
	}

	return operations.NewGetDocumentsOK().WithPayload(&swgmodels.DocumentsListResponse{
		Success:         &success,
		Acknowledgement: &ack,
		Documents:       respDocs,
	})
}

func (d *DocIMPL) GetDocumentsID(ctx context.Context, params operations.GetDocumentsIDParams, principal interface{}) middleware.Responder {
	log := pkglog.Logger()
	if d.store == nil {
		log.Error().Str("op", "get_documents_id").Msg("store not initialized")
		return middleware.ResponderFunc(func(rw http.ResponseWriter, _ runtime.Producer) {
			rw.WriteHeader(http.StatusInternalServerError)
		})
	}

	userID, ok := principal.(int64)
	if !ok || principal == nil {
		log.Warn().Str("op", "get_documents_id").Msg("unauthorized request")
		return operations.NewGetDocumentsIDUnauthorized().WithPayload(errorPayload(http.StatusUnauthorized, "Unauthorized"))
	}

	doc, err := d.store.GetDocumentByID(ctx, params.ID)
	if err != nil {
		log.Error().Str("op", "get_documents_id").Int64("user_id", userID).Int64("document_id", params.ID).Err(err).Msg("failed to fetch document")
		return middleware.ResponderFunc(func(rw http.ResponseWriter, _ runtime.Producer) {
			rw.WriteHeader(http.StatusInternalServerError)
		})
	}

	if doc == nil || doc.UserID != userID {
		log.Warn().Str("op", "get_documents_id").Int64("user_id", userID).Int64("document_id", params.ID).Msg("document not found for user")
		return operations.NewGetDocumentsIDNotFound().WithPayload(errorPayload(http.StatusNotFound, "Document not found"))
	}

	log.Info().Str("op", "get_documents_id").Int64("user_id", userID).Int64("document_id", params.ID).Msg("document fetched")
	return operations.NewGetDocumentsIDOK().WithPayload(documentResponsePayload("Document fetched successfully", doc))
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

	// Close uploaded stream when we're done
	if params.File != nil {
		defer func() {
			_ = params.File.Close()
		}()
	}

	// Sanitize filename to avoid path traversal / separators
	filename = strings.TrimSpace(filename)
	if filename == "" {
		filename = "upload"
	}
	filename = filepath.Base(filename)
	filename = strings.ReplaceAll(filename, string(os.PathSeparator), "_")

	if !intmodels.IsSupportedDocument(filename) {
		log.Warn().Str("op", "post_documents").Str("filename", filename).Msg("unsupported upload extension")
		return operations.NewPostDocumentsBadRequest().WithPayload(errorPayload(http.StatusBadRequest, "Unsupported file type"))
	}

	// Prefix with user+timestamp to avoid collisions/overwrites
	storedName := fmt.Sprintf("%d_%d_%s", userID, time.Now().UnixNano(), filename)

	log.Info().Str("op", "post_documents").Int64("user_id", userID).Str("filename", storedName).Msg("document upload request")

	// Create upload directory if it doesn't exist
	uploadDir := d.uploadDir
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		log.Error().Err(err).Msg("failed to create upload directory")
		return operations.NewPostDocumentsBadRequest()
	}

	// Save file to disk
	filePath := filepath.Join(uploadDir, storedName)
	outFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
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

	cleanupFile := func(reason string) {
		if err := cleanupUploadedFile(filePath); err != nil {
			log.Error().Err(err).Str("path", filePath).Msg("failed to clean up uploaded file after " + reason)
			return
		}
		log.Warn().Str("op", "post_documents").Str("path", filePath).Msg("cleaned up uploaded file after " + reason)
	}

	// Save document metadata to database
	doc := &intmodels.Document{
		UserID:    userID,
		Filename:  storedName,
		Status:    intmodels.StatusUploaded,
		CreatedAt: time.Now(),
	}
	if err := d.store.CreateDocument(ctx, doc); err != nil {
		log.Error().Err(err).Msg("failed to save document metadata")
		cleanupFile("database failure")
		return operations.NewPostDocumentsBadRequest()
	}

	log.Info().Str("op", "post_documents").Int64("document_id", doc.ID).Msg("document metadata saved")

	// Enqueue task to process the document
	err = d.broker.EnqueueTask("process_document", fmt.Sprint(doc.ID), filePath)
	if err != nil {
		log.Error().Err(err).Msg("failed to enqueue task")
		if deleteErr := d.store.DeleteDocumentByID(ctx, doc.ID); deleteErr != nil {
			log.Error().Err(deleteErr).Int64("document_id", doc.ID).Msg("failed to roll back document metadata after queue failure")
		}
		cleanupFile("queue failure")
		return operations.NewPostDocumentsBadRequest()
	}

	log.Info().Str("op", "post_documents").Int64("document_id", doc.ID).Str("filename", storedName).Msg("document processing task enqueued")

	success := true
	ack := "Document uploaded and queued for processing"

	return operations.NewPostDocumentsCreated().WithPayload(&swgmodels.DocumentCreatedResponse{
		Success:         &success,
		Acknowledgement: &ack,
		Document:        swaggerDocumentFromModel(doc),
	})
}

func cleanupUploadedFile(filePath string) error {
	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return nil
}

func resolveUploadDir(uploadDir string) string {
	uploadDir = strings.TrimSpace(uploadDir)
	if uploadDir == "" {
		return "./storage/uploads"
	}
	return uploadDir
}

func swaggerDocumentFromModel(doc *intmodels.Document) *swgmodels.Document {
	if doc == nil {
		return nil
	}

	id := doc.ID
	uid := doc.UserID
	filename := doc.Filename
	status := doc.Status
	created := strfmt.DateTime(doc.CreatedAt)

	return &swgmodels.Document{
		ID:        &id,
		UserID:    &uid,
		Filename:  &filename,
		Status:    &status,
		CreatedAt: &created,
	}
}

func documentResponsePayload(message string, doc *intmodels.Document) *swgmodels.DocumentResponse {
	success := true
	ack := message
	return &swgmodels.DocumentResponse{
		Success:         &success,
		Acknowledgement: &ack,
		Document:        swaggerDocumentFromModel(doc),
	}
}
