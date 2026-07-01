package models

import (
	"path/filepath"
	"strings"
	"time"
)

const (
	StatusUploaded   = "uploaded"
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
)

var supportedDocumentExtensions = map[string]struct{}{
	".pdf":  {},
	".txt":  {},
	".doc":  {},
	".docx": {},
	".jpg":  {},
	".jpeg": {},
	".png":  {},
}

var supportedDocumentStatuses = map[string]struct{}{
	StatusUploaded:   {},
	StatusProcessing: {},
	StatusCompleted:  {},
	StatusFailed:     {},
}

type Document struct {
	ID        int64     `pg:"id,pk"`
	UserID    int64     `pg:"user_id"`
	Filename  string    `pg:"filename"`
	Status    string    `pg:"status"`
	CreatedAt time.Time `pg:"created_at"`
}

func IsSupportedDocument(filename string) bool {
	ext := strings.ToLower(strings.TrimSpace(filepath.Ext(filename)))
	_, ok := supportedDocumentExtensions[ext]
	return ok
}

func IsSupportedDocumentStatus(status string) bool {
	status = strings.TrimSpace(status)
	_, ok := supportedDocumentStatuses[status]
	return ok
}
