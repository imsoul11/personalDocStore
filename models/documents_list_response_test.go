package models

import (
	"testing"
	"time"

	"github.com/go-openapi/strfmt"
)

func TestDocumentsListResponseRequiresTotalCount(t *testing.T) {
	success := true
	ack := "Documents fetched successfully"
	id := int64(1)
	userID := int64(2)
	filename := "file.pdf"
	status := "uploaded"
	createdAt := strfmt.DateTime(time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC))

	doc := &Document{
		ID:        &id,
		UserID:    &userID,
		Filename:  &filename,
		Status:    &status,
		CreatedAt: &createdAt,
	}

	resp := &DocumentsListResponse{
		Success:         &success,
		Acknowledgement: &ack,
		Documents:       []*Document{doc},
	}
	if err := resp.Validate(strfmt.Default); err == nil {
		t.Fatal("expected missing total_count to fail validation")
	}

	totalCount := int64(1)
	resp.TotalCount = &totalCount
	if err := resp.Validate(strfmt.Default); err != nil {
		t.Fatalf("expected total_count to validate, got %v", err)
	}
}
