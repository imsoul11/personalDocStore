package persistence

import (
	"context"
	"github.com/go-pg/pg/v10"
	"github.com/imsoul11/personalDocStore/internal/pkg/models"
)

func (pgstr *PGStore) CreateDocument(ctx context.Context, document *models.Document) error {
     _, err := pgstr.db.ModelContext(ctx, document).Insert()
	 if err!=nil{
		return err
	}
	return nil
}

func (pgstr *PGStore) GetDocumentByID(ctx context.Context, id int64) (*models.Document, error) {
	doc := &models.Document{ID: id}
	err := pgstr.db.ModelContext(ctx, doc).WherePK().Select()
	if err != nil {
		if err == pg.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return doc, nil
}

func (pgstr *PGStore) GetDocumentByUserID(ctx context.Context, userID int64) ([]*models.Document, error) {
	var docs []*models.Document
	err := pgstr.db.ModelContext(ctx, &docs).Where("user_id = ?", userID).Order("created_at DESC").Select()
	if err != nil {
		return nil, err
	}
	return docs, nil
}

func (pgstr *PGStore) UpdateDocumentStatus(ctx context.Context, id int64, status string) error {
	_, err := pgstr.db.ModelContext(ctx, (*models.Document)(nil)).
		Set("status = ?", status).
		Where("id = ?", id).
		Update()
	if err != nil {
		return err
	}
	return nil
}

