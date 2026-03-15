package api

import (
	"context"

	"github.com/go-openapi/runtime/middleware"
	"github.com/imsoul11/personalDocStore/restapi/operations"
)

// DocumentsAPI is the interface for document operations.
type DocumentsAPI interface {
	GetDocuments(ctx context.Context, params operations.GetDocumentsParams, principal interface{}) middleware.Responder
}

// UsersAPI is the interface for user operations.
type UsersAPI interface {
	PostUsersRegister(ctx context.Context, params operations.PostUsersRegisterParams) middleware.Responder
	PostUsersLogin(ctx context.Context, params operations.PostUsersLoginParams) middleware.Responder
}

// Config holds the API implementers.
type Config struct {
	DocumentsAPI DocumentsAPI
	UsersAPI     UsersAPI
}

var Cfg *Config

