package api

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	intmodels "github.com/imsoul11/personalDocStore/internal/pkg/models"
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
