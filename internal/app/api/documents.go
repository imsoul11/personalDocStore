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

const defaultMaxDocumentUploadBytes int64 = 10 << 20
const defaultDocumentListLimit int64 = 50

type DocIMPL struct {
	store          *persistence.PGStore
	broker         rabbitmq.Broker
	uploadDir      string
	maxUploadBytes int64
}

func NewDocuments(store *persistence.PGStore, broker rabbitmq.Broker, uploadDir string, maxUploadBytes int64) *DocIMPL {
	return &DocIMPL{
		store:          store,
		broker:         broker,
		uploadDir:      resolveUploadDir(uploadDir),
		maxUploadBytes: resolveMaxUploadBytes(maxUploadBytes),
	}
}

// GetDocuments implements the document list handler for the API server.
func (d *DocIMPL) GetDocuments(ctx context.Context, params operations.GetDocumentsParams, principal interface{}) middleware.Responder {
	log := pkglog.Logger()
	if d.store == nil {
		log.Error().Str("op", "get_documents").Msg("store not initialized")
		return operations.NewGetDocumentsInternalServerError().WithPayload(errorPayload(http.StatusInternalServerError, "Document storage is not configured"))
	}
	userID, ok := principal.(int64)
	if !ok || principal == nil {
		log.Warn().Str("op", "get_documents").Msg("unauthorized request")
		return operations.NewGetDocumentsUnauthorized().WithPayload(errorPayload(http.StatusUnauthorized, "Unauthorized"))
	}

	status, limit, offset, ok := resolveDocumentListQuery(params)
	if !ok {
		log.Warn().Str("op", "get_documents").Int64("user_id", userID).Msg("invalid pagination parameters")
		return operations.NewGetDocumentsBadRequest().WithPayload(errorPayload(http.StatusBadRequest, "Invalid document list parameters"))
	}

	log.Info().Str("op", "get_documents").Int64("user_id", userID).Str("status", status).Int64("limit", limit).Int64("offset", offset).Msg("fetching user documents")
	docs, err := d.store.GetDocumentByUserID(ctx, userID, status, limit, offset)
	if err != nil {
		log.Error().Str("op", "get_documents").Int64("user_id", userID).Err(err).Msg("failed to fetch documents")
		return operations.NewGetDocumentsInternalServerError().WithPayload(errorPayload(http.StatusInternalServerError, "Unable to fetch documents"))
	}
	totalCount, err := d.store.CountDocumentsByUserID(ctx, userID, status)
	if err != nil {
		log.Error().Str("op", "get_documents").Int64("user_id", userID).Err(err).Msg("failed to count documents")
		return operations.NewGetDocumentsInternalServerError().WithPayload(errorPayload(http.StatusInternalServerError, "Unable to count documents"))
	}
	log.Info().Str("op", "get_documents").Int64("user_id", userID).Int("documents_count", len(docs)).Int64("total_count", totalCount).Msg("documents fetched")

	_ = params

	success := true
	ack := "Documents fetched successfully"
	totalCountValue := totalCount
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
		TotalCount:      &totalCountValue,
	})
}

func (d *DocIMPL) GetDocumentsID(ctx context.Context, params operations.GetDocumentsIDParams, principal interface{}) middleware.Responder {
	log := pkglog.Logger()
	if d.store == nil {
		log.Error().Str("op", "get_documents_id").Msg("store not initialized")
		return operations.NewGetDocumentsIDInternalServerError().WithPayload(errorPayload(http.StatusInternalServerError, "Document storage is not configured"))
	}

	userID, ok := principal.(int64)
	if !ok || principal == nil {
		log.Warn().Str("op", "get_documents_id").Msg("unauthorized request")
		return operations.NewGetDocumentsIDUnauthorized().WithPayload(errorPayload(http.StatusUnauthorized, "Unauthorized"))
	}

	doc, err := d.store.GetDocumentByID(ctx, params.ID)
	if err != nil {
		log.Error().Str("op", "get_documents_id").Int64("user_id", userID).Int64("document_id", params.ID).Err(err).Msg("failed to fetch document")
		return operations.NewGetDocumentsIDInternalServerError().WithPayload(errorPayload(http.StatusInternalServerError, "Unable to fetch document"))
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
		return operations.NewPostDocumentsUnauthorized().WithPayload(errorPayload(http.StatusUnauthorized, "Unauthorized"))
	}

	filename := ""
	if params.Filename != nil {
		filename = *params.Filename
	}

	if params.File == nil {
		log.Warn().Str("op", "post_documents").Msg("document upload missing file")
		return operations.NewPostDocumentsBadRequest().WithPayload(errorPayload(http.StatusBadRequest, "File is required"))
	}

	// Close uploaded stream when we're done
	defer func() {
		_ = params.File.Close()
	}()

	// Sanitize filename to avoid path traversal / separators
	filename = resolveUploadFilename(filename, params.File)
	filename = filepath.Base(filename)
	filename = strings.ReplaceAll(filename, string(os.PathSeparator), "_")

	if !intmodels.IsSupportedDocument(filename) {
		log.Warn().Str("op", "post_documents").Str("filename", filename).Msg("unsupported upload extension")
		return operations.NewPostDocumentsBadRequest().WithPayload(errorPayload(http.StatusBadRequest, "Unsupported file type"))
	}
	if isUploadEmpty(params.File) {
		log.Warn().Str("op", "post_documents").Str("filename", filename).Msg("document upload is empty")
		return operations.NewPostDocumentsBadRequest().WithPayload(errorPayload(http.StatusBadRequest, "Uploaded file is empty"))
	}
	if isUploadTooLarge(params.File, d.maxUploadBytes) {
		log.Warn().Str("op", "post_documents").Str("filename", filename).Int64("max_upload_bytes", d.maxUploadBytes).Msg("document upload exceeds size limit")
		return operations.NewPostDocumentsBadRequest().WithPayload(errorPayload(http.StatusBadRequest, "Uploaded file exceeds size limit"))
	}

	// Prefix with user+timestamp to avoid collisions/overwrites
	storedName := fmt.Sprintf("%d_%d_%s", userID, time.Now().UnixNano(), filename)

	log.Info().Str("op", "post_documents").Int64("user_id", userID).Str("filename", storedName).Msg("document upload request")

	// Create upload directory if it doesn't exist
	uploadDir := d.uploadDir
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		log.Error().Err(err).Msg("failed to create upload directory")
		return operations.NewPostDocumentsBadRequest().WithPayload(errorPayload(http.StatusBadRequest, "Unable to prepare upload storage"))
	}

	// Save file to disk
	filePath := filepath.Join(uploadDir, storedName)
	outFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		log.Error().Err(err).Str("path", filePath).Msg("failed to create file")
		return operations.NewPostDocumentsBadRequest().WithPayload(errorPayload(http.StatusBadRequest, "Unable to create upload file"))
	}
	defer outFile.Close()

	written, err := io.Copy(outFile, io.LimitReader(params.File, d.maxUploadBytes+1))
	if err != nil {
		log.Error().Err(err).Msg("failed to save file")
		return operations.NewPostDocumentsBadRequest().WithPayload(errorPayload(http.StatusBadRequest, "Unable to save uploaded file"))
	}
	if written > d.maxUploadBytes {
		log.Warn().Str("op", "post_documents").Str("filename", storedName).Int64("max_upload_bytes", d.maxUploadBytes).Msg("document upload exceeded size limit during write")
		if cleanupErr := cleanupUploadedFile(filePath); cleanupErr != nil {
			log.Error().Err(cleanupErr).Str("path", filePath).Msg("failed to clean up oversized uploaded file")
		}
		return operations.NewPostDocumentsBadRequest().WithPayload(errorPayload(http.StatusBadRequest, "Uploaded file exceeds size limit"))
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
		return operations.NewPostDocumentsBadRequest().WithPayload(errorPayload(http.StatusBadRequest, "Unable to create document metadata"))
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
		return operations.NewPostDocumentsBadRequest().WithPayload(errorPayload(http.StatusBadRequest, "Unable to queue document processing"))
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

func resolveDocumentListQuery(params operations.GetDocumentsParams) (string, int64, int64, bool) {
	status := ""
	limit := defaultDocumentListLimit
	offset := int64(0)

	if params.Status != nil {
		status = strings.TrimSpace(*params.Status)
		if status != "" && !intmodels.IsSupportedDocumentStatus(status) {
			return "", 0, 0, false
		}
	}
	if params.Limit != nil {
		if *params.Limit <= 0 {
			return "", 0, 0, false
		}
		limit = *params.Limit
	}
	if params.Offset != nil {
		if *params.Offset < 0 {
			return "", 0, 0, false
		}
		offset = *params.Offset
	}

	return status, limit, offset, true
}

func resolveMaxUploadBytes(maxUploadBytes int64) int64 {
	if maxUploadBytes <= 0 {
		return defaultMaxDocumentUploadBytes
	}
	return maxUploadBytes
}

func resolveUploadFilename(filename string, file io.ReadCloser) string {
	filename = strings.TrimSpace(filename)
	if filename != "" {
		return filename
	}
	if uploadedFile, ok := file.(*runtime.File); ok && uploadedFile.Header != nil {
		headerFilename := strings.TrimSpace(uploadedFile.Header.Filename)
		if headerFilename != "" {
			return headerFilename
		}
	}
	return "upload"
}

func isUploadTooLarge(file io.ReadCloser, maxBytes int64) bool {
	if uploadedFile, ok := file.(*runtime.File); ok && uploadedFile.Header != nil {
		return uploadedFile.Header.Size > maxBytes
	}
	return false
}

func isUploadEmpty(file io.ReadCloser) bool {
	if uploadedFile, ok := file.(*runtime.File); ok && uploadedFile.Header != nil {
		return uploadedFile.Header.Size == 0
	}
	return false
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
