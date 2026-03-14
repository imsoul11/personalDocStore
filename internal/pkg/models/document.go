package models

import "time"

const (
	StatusUploaded   = "uploaded"
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
)

type Document struct {
	ID        int64     `pg:"id,pk"`
	UserID    int64     `pg:"user_id"`
	Filename  string    `pg:"filename"`
	Status    string    `pg:"status"`
	CreatedAt time.Time `pg:"created_at"`
}