package api

import (
	"context"
	"net/http"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"
	"github.com/imsoul11/personalDocStore/internal/pkg/persistence"
	"github.com/imsoul11/personalDocStore/restapi/operations"
)

type DocIMPL struct {
	store *persistence.PGStore
}

func NewDocuments(store *persistence.PGStore) *DocIMPL {
	return &DocIMPL{store: store}
}

// GetDocuments implements the document list handler for the API server.
func (d *DocIMPL) GetDocuments(ctx context.Context, params operations.GetDocumentsParams, principal interface{}) middleware.Responder {
	if d.store == nil {
		return middleware.ResponderFunc(func(rw http.ResponseWriter, _ runtime.Producer) {
			rw.WriteHeader(http.StatusInternalServerError)
		})
	}
	userID, ok := principal.(int64)
	if !ok || principal == nil {
		return operations.NewGetDocumentsUnauthorized()
	}
	docs, err := d.store.GetDocumentByUserID(ctx, userID)
	if err != nil {
		return middleware.ResponderFunc(func(rw http.ResponseWriter, _ runtime.Producer) {
			rw.WriteHeader(http.StatusInternalServerError)
		})
	}
	_ = docs // response body not in spec yet; return OK
	return operations.NewGetDocumentsOK()
}
