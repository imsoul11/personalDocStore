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
	updatedAt := strfmt.DateTime(time.Date(2026, time.January, 2, 4, 4, 5, 0, time.UTC))

	for _, status := range []string{"uploaded", "processing", "completed", "failed"} {
		doc := &Document{
			CreatedAt: &createdAt,
			Filename:  &filename,
			ID:        &id,
			Status:    &status,
			UpdatedAt: &updatedAt,
			UserID:    &userID,
		}
		if err := doc.Validate(strfmt.Default); err != nil {
			t.Fatalf("expected status %q to validate, got %v", status, err)
		}
	}

	invalidStatus := "processed"
	validStatus := "uploaded"
	doc := &Document{
		CreatedAt: &createdAt,
		Filename:  &filename,
		ID:        &id,
		Status:    &invalidStatus,
		UpdatedAt: &updatedAt,
		UserID:    &userID,
	}
	if err := doc.Validate(strfmt.Default); err == nil {
		t.Fatal("expected invalid status to fail validation")
	}

	doc = &Document{
		CreatedAt: &createdAt,
		Filename:  &filename,
		ID:        &id,
		Status:    &validStatus,
		UserID:    &userID,
	}
	if err := doc.Validate(strfmt.Default); err == nil {
		t.Fatal("expected missing updated_at to fail validation")
	}
}
