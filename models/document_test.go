package models

import (
	"testing"
	"time"

	"github.com/go-openapi/strfmt"
)

func TestDocumentValidateStatusEnum(t *testing.T) {
	createdAt := strfmt.DateTime(time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC))
	filename := "file.pdf"
	id := int64(1)
	userID := int64(2)

	for _, status := range []string{"uploaded", "processing", "completed", "failed"} {
		doc := &Document{
			CreatedAt: &createdAt,
			Filename:  &filename,
			ID:        &id,
			Status:    &status,
			UserID:    &userID,
		}
		if err := doc.Validate(strfmt.Default); err != nil {
			t.Fatalf("expected status %q to validate, got %v", status, err)
		}
	}

	invalidStatus := "processed"
	doc := &Document{
		CreatedAt: &createdAt,
		Filename:  &filename,
		ID:        &id,
		Status:    &invalidStatus,
		UserID:    &userID,
	}
	if err := doc.Validate(strfmt.Default); err == nil {
		t.Fatal("expected invalid status to fail validation")
	}
}
