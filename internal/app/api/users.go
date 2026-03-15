package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"
	"golang.org/x/crypto/bcrypt"

	"github.com/imsoul11/personalDocStore/internal/pkg/models"
	"github.com/imsoul11/personalDocStore/internal/pkg/persistence"
	"github.com/imsoul11/personalDocStore/restapi/operations"
)

type UsersIMPL struct {
	store     *persistence.PGStore
	jwtSecret []byte
}

func NewUsers(store *persistence.PGStore, jwtSecret string) *UsersIMPL {
	return &UsersIMPL{store: store, jwtSecret: []byte(jwtSecret)}
}

func (u *UsersIMPL) PostUsersRegister(ctx context.Context, params operations.PostUsersRegisterParams) middleware.Responder {
	email := params.Body.Email.String()
	password := *params.Body.Password

	existing, err := u.store.GetUserByEmail(ctx, email)
	if err != nil {
		return operations.NewPostUsersRegisterBadRequest()
	}
	if existing != nil {
		return operations.NewPostUsersRegisterConflict()
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return operations.NewPostUsersRegisterBadRequest()
	}

	user := &models.User{
		Email:        email,
		PasswordHash: string(hash),
		CreatedAt:    time.Now(),
	}
	if err := u.store.CreateUser(ctx, user); err != nil {
		return operations.NewPostUsersRegisterBadRequest()
	}

	return operations.NewPostUsersRegisterCreated()
}

func (u *UsersIMPL) PostUsersLogin(ctx context.Context, params operations.PostUsersLoginParams) middleware.Responder {
	email := params.Body.Email.String()
	password := *params.Body.Password

	user, err := u.store.GetUserByEmail(ctx, email)
	if err != nil || user == nil {
		return operations.NewPostUsersLoginUnauthorized()
	}

	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return operations.NewPostUsersLoginUnauthorized()
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})
	tokenStr, err := token.SignedString(u.jwtSecret)
	if err != nil {
		return middleware.ResponderFunc(func(rw http.ResponseWriter, _ runtime.Producer) {
			rw.WriteHeader(http.StatusInternalServerError)
		})
	}

	return middleware.ResponderFunc(func(rw http.ResponseWriter, _ runtime.Producer) {
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(map[string]string{"token": tokenStr})
	})
}
