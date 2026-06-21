package api

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

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

func TestGetDocumentsUnauthorizedIncludesErrorPayload(t *testing.T) {
	api := NewDocuments(&persistence.PGStore{}, nil, "")

	resp, ok := api.GetDocuments(context.Background(), operations.GetDocumentsParams{}, nil).(*operations.GetDocumentsUnauthorized)
	if !ok {
		t.Fatalf("expected GetDocumentsUnauthorized response")
	}
	if resp.Payload == nil || resp.Payload.Code == nil || *resp.Payload.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized error payload, got %#v", resp.Payload)
	}
}

func TestPostDocumentsUnauthorizedIncludesErrorPayload(t *testing.T) {
	api := NewDocuments(nil, nil, "")

	resp, ok := api.PostDocuments(context.Background(), operations.PostDocumentsParams{}, nil).(*operations.PostDocumentsUnauthorized)
	if !ok {
		t.Fatalf("expected PostDocumentsUnauthorized response")
	}
	if resp.Payload == nil || resp.Payload.Code == nil || *resp.Payload.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized error payload, got %#v", resp.Payload)
	}
}

func TestPostDocumentsMissingFileIncludesErrorPayload(t *testing.T) {
	api := NewDocuments(nil, nil, "")

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
