package api

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-openapi/runtime"
	intmodels "github.com/imsoul11/personalDocStore/internal/pkg/models"
	"github.com/imsoul11/personalDocStore/internal/pkg/persistence"
	"github.com/imsoul11/personalDocStore/restapi/operations"
)

func TestSwaggerDocumentFromModel(t *testing.T) {
	createdAt := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	doc := &intmodels.Document{
		ID:        17,
		UserID:    29,
		Filename:  "file.pdf",
		Status:    intmodels.StatusUploaded,
		CreatedAt: createdAt,
		UpdatedAt: createdAt.Add(2 * time.Hour),
	}

	payload := swaggerDocumentFromModel(doc)
	if payload == nil {
		t.Fatal("expected swagger document")
	}
	if payload.ID == nil || *payload.ID != doc.ID {
		t.Fatalf("expected id %d", doc.ID)
	}
	if payload.UserID == nil || *payload.UserID != doc.UserID {
		t.Fatalf("expected user id %d", doc.UserID)
	}
	if payload.Filename == nil || *payload.Filename != doc.Filename {
		t.Fatalf("expected filename %q", doc.Filename)
	}
	if payload.Status == nil || *payload.Status != doc.Status {
		t.Fatalf("expected status %q", doc.Status)
	}
	if payload.CreatedAt == nil || !time.Time(*payload.CreatedAt).Equal(createdAt) {
		t.Fatalf("expected created_at %s", createdAt.Format(time.RFC3339))
	}
	if payload.UpdatedAt == nil || !time.Time(*payload.UpdatedAt).Equal(doc.UpdatedAt) {
		t.Fatalf("expected updated_at %s", doc.UpdatedAt.Format(time.RFC3339))
	}
}

func TestSwaggerDocumentFromModelSupportsAllStatuses(t *testing.T) {
	createdAt := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)

	for _, status := range []string{
		intmodels.StatusUploaded,
		intmodels.StatusProcessing,
		intmodels.StatusCompleted,
		intmodels.StatusFailed,
	} {
		doc := &intmodels.Document{
			ID:        17,
			UserID:    29,
			Filename:  "file.pdf",
			Status:    status,
			CreatedAt: createdAt,
			UpdatedAt: createdAt.Add(2 * time.Hour),
		}

		payload := swaggerDocumentFromModel(doc)
		if payload == nil {
			t.Fatalf("expected swagger document for status %q", status)
		}
		if payload.Status == nil || *payload.Status != status {
			t.Fatalf("expected status %q, got %#v", status, payload.Status)
		}
	}
}

func TestDocumentResponsePayloadSupportsAllStatuses(t *testing.T) {
	createdAt := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)

	for _, status := range []string{
		intmodels.StatusUploaded,
		intmodels.StatusProcessing,
		intmodels.StatusCompleted,
		intmodels.StatusFailed,
	} {
		doc := &intmodels.Document{
			ID:        17,
			UserID:    29,
			Filename:  "file.pdf",
			Status:    status,
			CreatedAt: createdAt,
			UpdatedAt: createdAt.Add(2 * time.Hour),
		}

		payload := documentResponsePayload("Document fetched successfully", doc)
		if payload == nil || payload.Document == nil {
			t.Fatalf("expected document response payload for status %q", status)
		}
		if payload.Document.Status == nil || *payload.Document.Status != status {
			t.Fatalf("expected status %q, got %#v", status, payload.Document.Status)
		}
		if payload.Document.UpdatedAt == nil || !time.Time(*payload.Document.UpdatedAt).Equal(doc.UpdatedAt) {
			t.Fatalf("expected updated_at %s, got %#v", doc.UpdatedAt.Format(time.RFC3339), payload.Document.UpdatedAt)
		}
	}
}

func TestCleanupUploadedFileRemovesFile(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "upload.txt")
	if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	if err := cleanupUploadedFile(filePath); err != nil {
		t.Fatalf("cleanupUploadedFile returned error: %v", err)
	}

	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Fatalf("expected file to be removed, stat err = %v", err)
	}
}

func TestCleanupUploadedFileIgnoresMissingFile(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "missing.txt")
	if err := cleanupUploadedFile(filePath); err != nil {
		t.Fatalf("expected missing file cleanup to succeed, got %v", err)
	}
}

func TestIsSupportedDocument(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{name: "pdf", filename: "report.pdf", want: true},
		{name: "uppercase extension", filename: "scan.JPG", want: true},
		{name: "docx", filename: "notes.docx", want: true},
		{name: "missing extension", filename: "upload", want: false},
		{name: "unsupported extension", filename: "archive.zip", want: false},
	}

	for _, tt := range tests {
		if got := intmodels.IsSupportedDocument(tt.filename); got != tt.want {
			t.Fatalf("%s: expected %v, got %v", tt.name, tt.want, got)
		}
	}
}

func TestResolveUploadDir(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "configured path", in: "/tmp/uploads", want: "/tmp/uploads"},
		{name: "trim whitespace", in: "  ./custom/uploads  ", want: "./custom/uploads"},
		{name: "default path", in: "", want: "./storage/uploads"},
	}

	for _, tt := range tests {
		if got := resolveUploadDir(tt.in); got != tt.want {
			t.Fatalf("%s: expected %q, got %q", tt.name, tt.want, got)
		}
	}
}

func TestResolveMaxUploadBytes(t *testing.T) {
	tests := []struct {
		name string
		in   int64
		want int64
	}{
		{name: "configured bytes", in: 5 << 20, want: 5 << 20},
		{name: "default bytes", in: 0, want: defaultMaxDocumentUploadBytes},
		{name: "negative bytes use default", in: -1, want: defaultMaxDocumentUploadBytes},
	}

	for _, tt := range tests {
		if got := resolveMaxUploadBytes(tt.in); got != tt.want {
			t.Fatalf("%s: expected %d, got %d", tt.name, tt.want, got)
		}
	}
}

func TestResolveDocumentListQuery(t *testing.T) {
	status, limit, offset, ok := resolveDocumentListQuery(operations.GetDocumentsParams{})
	if !ok {
		t.Fatal("expected default document list query to be valid")
	}
	if status != "" || limit != defaultDocumentListLimit || offset != 0 {
		t.Fatalf("expected default status='' limit=%d offset=0, got status=%q limit=%d offset=%d", defaultDocumentListLimit, status, limit, offset)
	}

	customStatus := intmodels.StatusCompleted
	customLimit := int64(10)
	customOffset := int64(5)
	status, limit, offset, ok = resolveDocumentListQuery(operations.GetDocumentsParams{
		Status: &customStatus,
		Limit:  &customLimit,
		Offset: &customOffset,
	})
	if !ok {
		t.Fatal("expected custom document list query to be valid")
	}
	if status != customStatus || limit != customLimit || offset != customOffset {
		t.Fatalf("expected status=%q limit=%d offset=%d, got status=%q limit=%d offset=%d", customStatus, customLimit, customOffset, status, limit, offset)
	}
}

func TestResolveDocumentListQueryRejectsInvalidValues(t *testing.T) {
	zeroLimit := int64(0)
	if _, _, _, ok := resolveDocumentListQuery(operations.GetDocumentsParams{Limit: &zeroLimit}); ok {
		t.Fatal("expected zero limit to be rejected")
	}

	negativeOffset := int64(-1)
	if _, _, _, ok := resolveDocumentListQuery(operations.GetDocumentsParams{Offset: &negativeOffset}); ok {
		t.Fatal("expected negative offset to be rejected")
	}

	invalidStatus := "processed"
	if _, _, _, ok := resolveDocumentListQuery(operations.GetDocumentsParams{Status: &invalidStatus}); ok {
		t.Fatal("expected unsupported status to be rejected")
	}
}

func TestResolveUploadFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		file     io.ReadCloser
		want     string
	}{
		{name: "explicit filename wins", filename: "manual.pdf", file: uploadedFile("header.docx"), want: "manual.pdf"},
		{name: "header filename fallback", filename: "", file: uploadedFile("scan.jpg"), want: "scan.jpg"},
		{name: "trim explicit filename", filename: "  notes.txt  ", file: uploadedFile("scan.jpg"), want: "notes.txt"},
		{name: "default upload filename", filename: "", file: nil, want: "upload"},
	}

	for _, tt := range tests {
		if got := resolveUploadFilename(tt.filename, tt.file); got != tt.want {
			t.Fatalf("%s: expected %q, got %q", tt.name, tt.want, got)
		}
	}
}

func TestNewDocumentsUsesResolvedUploadConfig(t *testing.T) {
	api := NewDocuments(nil, nil, " ./uploads ", 5<<20)

	if api.uploadDir != "./uploads" {
		t.Fatalf("expected upload dir to be trimmed, got %q", api.uploadDir)
	}
	if api.maxUploadBytes != 5<<20 {
		t.Fatalf("expected max upload bytes to be set, got %d", api.maxUploadBytes)
	}
}

func TestIsUploadTooLarge(t *testing.T) {
	tests := []struct {
		name string
		file io.ReadCloser
		want bool
	}{
		{name: "below limit", file: uploadedFileWithSize("small.pdf", defaultMaxDocumentUploadBytes), want: false},
		{name: "above limit", file: uploadedFileWithSize("big.pdf", defaultMaxDocumentUploadBytes+1), want: true},
		{name: "missing header", file: nil, want: false},
	}

	for _, tt := range tests {
		if got := isUploadTooLarge(tt.file, defaultMaxDocumentUploadBytes); got != tt.want {
			t.Fatalf("%s: expected %v, got %v", tt.name, tt.want, got)
		}
	}
}

func TestIsUploadEmpty(t *testing.T) {
	tests := []struct {
		name string
		file io.ReadCloser
		want bool
	}{
		{name: "empty file", file: uploadedFileWithSize("empty.pdf", 0), want: true},
		{name: "non-empty file", file: uploadedFileWithSize("full.pdf", 1), want: false},
		{name: "missing header", file: nil, want: false},
	}

	for _, tt := range tests {
		if got := isUploadEmpty(tt.file); got != tt.want {
			t.Fatalf("%s: expected %v, got %v", tt.name, tt.want, got)
		}
	}
}

func TestGetDocumentsUnauthorizedIncludesErrorPayload(t *testing.T) {
	api := NewDocuments(&persistence.PGStore{}, nil, "", 0)

	resp, ok := api.GetDocuments(context.Background(), operations.GetDocumentsParams{}, nil).(*operations.GetDocumentsUnauthorized)
	if !ok {
		t.Fatalf("expected GetDocumentsUnauthorized response")
	}
	if resp.Payload == nil || resp.Payload.Code == nil || *resp.Payload.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized error payload, got %#v", resp.Payload)
	}
}

func TestGetDocumentsInvalidPaginationIncludesErrorPayload(t *testing.T) {
	api := NewDocuments(&persistence.PGStore{}, nil, "", 0)
	zeroLimit := int64(0)

	resp, ok := api.GetDocuments(context.Background(), operations.GetDocumentsParams{Limit: &zeroLimit}, int64(7)).(*operations.GetDocumentsBadRequest)
	if !ok {
		t.Fatalf("expected GetDocumentsBadRequest response")
	}
	if resp.Payload == nil || resp.Payload.Code == nil || *resp.Payload.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request payload, got %#v", resp.Payload)
	}
}

func TestGetDocumentsInvalidStatusIncludesErrorPayload(t *testing.T) {
	api := NewDocuments(&persistence.PGStore{}, nil, "", 0)
	invalidStatus := "processed"

	resp, ok := api.GetDocuments(context.Background(), operations.GetDocumentsParams{Status: &invalidStatus}, int64(7)).(*operations.GetDocumentsBadRequest)
	if !ok {
		t.Fatalf("expected GetDocumentsBadRequest response")
	}
	if resp.Payload == nil || resp.Payload.Code == nil || *resp.Payload.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request payload, got %#v", resp.Payload)
	}
}

func TestGetDocumentsStoreNotConfiguredIncludesErrorPayload(t *testing.T) {
	api := NewDocuments(nil, nil, "", 0)

	resp, ok := api.GetDocuments(context.Background(), operations.GetDocumentsParams{}, int64(7)).(*operations.GetDocumentsInternalServerError)
	if !ok {
		t.Fatalf("expected GetDocumentsInternalServerError response")
	}
	if resp.Payload == nil || resp.Payload.Code == nil || *resp.Payload.Code != http.StatusInternalServerError {
		t.Fatalf("expected internal server error payload, got %#v", resp.Payload)
	}
}

func TestPostDocumentsUnauthorizedIncludesErrorPayload(t *testing.T) {
	api := NewDocuments(nil, nil, "", 0)

	resp, ok := api.PostDocuments(context.Background(), operations.PostDocumentsParams{}, nil).(*operations.PostDocumentsUnauthorized)
	if !ok {
		t.Fatalf("expected PostDocumentsUnauthorized response")
	}
	if resp.Payload == nil || resp.Payload.Code == nil || *resp.Payload.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized error payload, got %#v", resp.Payload)
	}
}

func TestPostDocumentsMissingFileIncludesErrorPayload(t *testing.T) {
	api := NewDocuments(nil, nil, "", 0)

	resp, ok := api.PostDocuments(context.Background(), operations.PostDocumentsParams{}, int64(7)).(*operations.PostDocumentsBadRequest)
	if !ok {
		t.Fatalf("expected PostDocumentsBadRequest response")
	}
	if resp.Payload == nil || resp.Payload.Code == nil || *resp.Payload.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request payload, got %#v", resp.Payload)
	}
	if resp.Payload.Message == nil || *resp.Payload.Message != "File is required" {
		t.Fatalf("expected missing file message, got %#v", resp.Payload.Message)
	}
}

func TestPostDocumentsOversizedFileIncludesErrorPayload(t *testing.T) {
	api := NewDocuments(nil, nil, "", 0)
	params := operations.PostDocumentsParams{
		File: uploadedFileWithSize("large.pdf", defaultMaxDocumentUploadBytes+1),
	}

	resp, ok := api.PostDocuments(context.Background(), params, int64(7)).(*operations.PostDocumentsBadRequest)
	if !ok {
		t.Fatalf("expected PostDocumentsBadRequest response")
	}
	if resp.Payload == nil || resp.Payload.Code == nil || *resp.Payload.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request payload, got %#v", resp.Payload)
	}
	if resp.Payload.Message == nil || *resp.Payload.Message != "Uploaded file exceeds size limit" {
		t.Fatalf("expected size limit message, got %#v", resp.Payload.Message)
	}
}

func TestPostDocumentsEmptyFileIncludesErrorPayload(t *testing.T) {
	api := NewDocuments(nil, nil, "", 0)
	params := operations.PostDocumentsParams{
		File: uploadedFileWithSize("empty.pdf", 0),
	}

	resp, ok := api.PostDocuments(context.Background(), params, int64(7)).(*operations.PostDocumentsBadRequest)
	if !ok {
		t.Fatalf("expected PostDocumentsBadRequest response")
	}
	if resp.Payload == nil || resp.Payload.Code == nil || *resp.Payload.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request payload, got %#v", resp.Payload)
	}
	if resp.Payload.Message == nil || *resp.Payload.Message != "Uploaded file is empty" {
		t.Fatalf("expected empty file message, got %#v", resp.Payload.Message)
	}
}

func TestGetDocumentsIDStoreNotConfiguredIncludesErrorPayload(t *testing.T) {
	api := NewDocuments(nil, nil, "", 0)

	resp, ok := api.GetDocumentsID(context.Background(), operations.GetDocumentsIDParams{}, int64(7)).(*operations.GetDocumentsIDInternalServerError)
	if !ok {
		t.Fatalf("expected GetDocumentsIDInternalServerError response")
	}
	if resp.Payload == nil || resp.Payload.Code == nil || *resp.Payload.Code != http.StatusInternalServerError {
		t.Fatalf("expected internal server error payload, got %#v", resp.Payload)
	}
}

func uploadedFile(filename string) io.ReadCloser {
	return &runtime.File{
		Data:   testMultipartFile{Reader: bytes.NewReader(nil)},
		Header: &multipart.FileHeader{Filename: filename},
	}
}

func uploadedFileWithSize(filename string, size int64) io.ReadCloser {
	return &runtime.File{
		Data:   testMultipartFile{Reader: bytes.NewReader(nil)},
		Header: &multipart.FileHeader{Filename: filename, Size: size},
	}
}

type testMultipartFile struct {
	*bytes.Reader
}

func (f testMultipartFile) Close() error {
	return nil
}
