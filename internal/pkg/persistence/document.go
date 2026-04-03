package persistence

import (
	"context"
	"github.com/go-pg/pg/v10"
	pkglog "github.com/imsoul11/personalDocStore/internal/pkg/log"
	"github.com/imsoul11/personalDocStore/internal/pkg/models"
)

func (pgstr *PGStore) CreateDocument(ctx context.Context, document *models.Document) error {
	log := pkglog.Logger()
	log.Debug().Str("op", "store_create_document").Int64("user_id", document.UserID).Str("filename", document.Filename).Msg("creating document")
	_, err := pgstr.db.ModelContext(ctx, document).Insert()
	if err != nil {
		log.Error().Str("op", "store_create_document").Int64("user_id", document.UserID).Err(err).Msg("create document failed")
		return err
	}
	log.Debug().Str("op", "store_create_document").Int64("user_id", document.UserID).Int64("document_id", document.ID).Msg("document created")
	return nil
}

func (pgstr *PGStore) GetDocumentByID(ctx context.Context, id int64) (*models.Document, error) {
	log := pkglog.Logger()
	log.Debug().Str("op", "store_get_document_by_id").Int64("document_id", id).Msg("fetching document by id")
	doc := &models.Document{ID: id}
	err := pgstr.db.ModelContext(ctx, doc).WherePK().Select()
	if err != nil {
		if err == pg.ErrNoRows {
			log.Debug().Str("op", "store_get_document_by_id").Int64("document_id", id).Msg("document not found")
			return nil, nil
		}
		log.Error().Str("op", "store_get_document_by_id").Int64("document_id", id).Err(err).Msg("get document by id failed")
		return nil, err
	}
	log.Debug().Str("op", "store_get_document_by_id").Int64("document_id", id).Int64("user_id", doc.UserID).Msg("document fetched")
	return doc, nil
}

func (pgstr *PGStore) GetDocumentByUserID(ctx context.Context, userID int64) ([]*models.Document, error) {
	log := pkglog.Logger()
	log.Debug().Str("op", "store_get_documents_by_user_id").Int64("user_id", userID).Msg("fetching documents by user id")
	var docs []*models.Document
	err := pgstr.db.ModelContext(ctx, &docs).Where("user_id = ?", userID).Order("created_at DESC").Select()
	if err != nil {
		log.Error().Str("op", "store_get_documents_by_user_id").Int64("user_id", userID).Err(err).Msg("fetch documents by user id failed")
		return nil, err
	}
	log.Debug().Str("op", "store_get_documents_by_user_id").Int64("user_id", userID).Int("documents_count", len(docs)).Msg("documents fetched")
	return docs, nil
}

func (pgstr *PGStore) UpdateDocumentStatus(ctx context.Context, id int64, status string) error {
	log := pkglog.Logger()
	log.Debug().Str("op", "store_update_document_status").Int64("document_id", id).Str("status", status).Msg("updating document status")
	_, err := pgstr.db.ModelContext(ctx, (*models.Document)(nil)).
		Set("status = ?", status).
		Where("id = ?", id).
		Update()
	if err != nil {
		log.Error().Str("op", "store_update_document_status").Int64("document_id", id).Str("status", status).Err(err).Msg("update document status failed")
		return err
	}
	log.Debug().Str("op", "store_update_document_status").Int64("document_id", id).Str("status", status).Msg("document status updated")
	return nil
}

