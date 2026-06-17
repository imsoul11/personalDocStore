package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	pkglog "github.com/imsoul11/personalDocStore/internal/pkg/log"
	"github.com/imsoul11/personalDocStore/internal/pkg/models"
	"github.com/imsoul11/personalDocStore/internal/pkg/persistence"
	swgm "github.com/imsoul11/personalDocStore/models"
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
	log := pkglog.Logger()
	log.Info().Str("op", "post_users_register").Str("email", email).Msg("register request received")

	existing, err := u.store.GetUserByEmail(ctx, email)
	if err != nil {
		log.Error().Str("op", "post_users_register").Str("email", email).Err(err).Msg("failed to check existing user")
		return operations.NewPostUsersRegisterBadRequest().WithPayload(errorPayload(http.StatusBadRequest, "Unable to validate register request"))
	}
	if existing != nil {
		log.Warn().Str("op", "post_users_register").Str("email", email).Msg("register conflict: user already exists")
		return operations.NewPostUsersRegisterConflict().WithPayload(errorPayload(http.StatusConflict, "Email already exists"))
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Error().Str("op", "post_users_register").Str("email", email).Err(err).Msg("password hash generation failed")
		return operations.NewPostUsersRegisterBadRequest().WithPayload(errorPayload(http.StatusBadRequest, "Unable to process password"))
	}

	user := &models.User{
		Email:        email,
		PasswordHash: string(hash),
		CreatedAt:    time.Now(),
	}
	if err := u.store.CreateUser(ctx, user); err != nil {
		log.Error().Str("op", "post_users_register").Str("email", email).Err(err).Msg("failed to create user")
		return operations.NewPostUsersRegisterBadRequest().WithPayload(errorPayload(http.StatusBadRequest, "Unable to create user"))
	}
	log.Info().Str("op", "post_users_register").Str("email", email).Int64("user_id", user.ID).Msg("user registered")

	return operations.NewPostUsersRegisterCreated().WithPayload(ackPayload("User registered successfully"))
}

func (u *UsersIMPL) PostUsersLogin(ctx context.Context, params operations.PostUsersLoginParams) middleware.Responder {
	email := params.Body.Email.String()
	password := *params.Body.Password
	log := pkglog.Logger()
	log.Info().Str("op", "post_users_login").Str("email", email).Msg("login request received")

	user, err := u.store.GetUserByEmail(ctx, email)
	if err != nil || user == nil {
		log.Warn().Str("op", "post_users_login").Str("email", email).Err(err).Msg("login failed: user not found")
		return operations.NewPostUsersLoginUnauthorized().WithPayload(errorPayload(http.StatusUnauthorized, "Invalid credentials"))
	}

	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		log.Warn().Str("op", "post_users_login").Str("email", email).Int64("user_id", user.ID).Msg("login failed: invalid password")
		return operations.NewPostUsersLoginUnauthorized().WithPayload(errorPayload(http.StatusUnauthorized, "Invalid credentials"))
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})
	tokenStr, err := token.SignedString(u.jwtSecret)
	if err != nil {
		log.Error().Str("op", "post_users_login").Str("email", email).Int64("user_id", user.ID).Err(err).Msg("failed to sign token")
		return middleware.ResponderFunc(func(rw http.ResponseWriter, _ runtime.Producer) {
			rw.WriteHeader(http.StatusInternalServerError)
		})
	}
	log.Info().Str("op", "post_users_login").Str("email", email).Int64("user_id", user.ID).Msg("login success")
	success := true
	ack := "Login successful"
	return operations.NewPostUsersLoginOK().WithPayload(
		&swgm.LoginResponse{
			Success:         &success,
			Acknowledgement: &ack,
			Token:           &tokenStr,
		},
	)
}

func (u *UsersIMPL) GetUsersProfile(ctx context.Context, params operations.GetUsersProfileParams, principal interface{}) middleware.Responder {
	log := pkglog.Logger()
	userID, ok := principal.(int64)
	if !ok {
		log.Warn().Str("op", "get_users_profile").Msg("unauthorized profile fetch")
		return operations.NewGetUsersProfileUnauthorized().WithPayload(errorPayload(http.StatusUnauthorized, "Unauthorized"))
	}

	user, err := u.store.GetUserByID(ctx, userID)
	if err != nil || user == nil {
		log.Error().Str("op", "get_users_profile").Int64("user_id", userID).Err(err).Msg("failed to fetch user profile")
		return operations.NewGetUsersProfileUnauthorized().WithPayload(errorPayload(http.StatusUnauthorized, "Unauthorized"))
	}
	_ = params
	log.Info().Str("op", "get_users_profile").Int64("user_id", userID).Msg("user profile fetched")
	return operations.NewGetUsersProfileOK().WithPayload(userProfileResponsePayload("Profile fetched successfully", user))
}

func (u *UsersIMPL) PostUsersProfile(ctx context.Context, params operations.PostUsersProfileParams, principal interface{}) middleware.Responder {
	log := pkglog.Logger()
	userID, ok := principal.(int64)
	if !ok {
		log.Warn().Str("op", "post_users_profile").Msg("unauthorized profile update")
		return operations.NewPostUsersProfileUnauthorized().WithPayload(errorPayload(http.StatusUnauthorized, "Unauthorized"))
	}

	oldUser, err := u.store.GetUserByID(ctx, userID)
	if err != nil {
		log.Error().Str("op", "post_users_profile").Int64("user_id", userID).Err(err).Msg("failed to get user")
		return middleware.ResponderFunc(func(rw http.ResponseWriter, _ runtime.Producer) {
			rw.WriteHeader(http.StatusInternalServerError)
		})
	}
	if oldUser == nil {
		log.Warn().Str("op", "post_users_profile").Int64("user_id", userID).Msg("user not found for profile update")
		return operations.NewPostUsersProfileUnauthorized().WithPayload(errorPayload(http.StatusUnauthorized, "Unauthorized"))
	}

	updatedUser, changed := applyUserProfileUpdate(oldUser, params.Body)
	if !changed {
		log.Warn().Str("op", "post_users_profile").Int64("user_id", userID).Msg("profile update request contained no fields")
		return operations.NewPostUsersProfileBadRequest().WithPayload(errorPayload(http.StatusBadRequest, "At least one profile field must be provided"))
	}

	if err := u.store.UpdateUser(ctx, updatedUser); err != nil {
		log.Error().Str("op", "post_users_profile").Int64("user_id", userID).Err(err).Msg("failed to update user")
		return middleware.ResponderFunc(func(rw http.ResponseWriter, _ runtime.Producer) {
			rw.WriteHeader(http.StatusInternalServerError)
		})
	}

	log.Info().Str("op", "post_users_profile").Int64("user_id", userID).Msg("profile updated")
	return operations.NewPostUsersProfileOK().WithPayload(ackPayload("Profile updated successfully"))
}

func ackPayload(message string) *swgm.AckResponse {
	success := true
	ack := message
	return &swgm.AckResponse{
		Success:         &success,
		Acknowledgement: &ack,
	}
}

func errorPayload(code int, message string) *swgm.ErrorResponse {
	success := false
	ack := "Request failed"
	errCode := int32(code)
	msg := message
	return &swgm.ErrorResponse{
		Success:         &success,
		Acknowledgement: &ack,
		Code:            &errCode,
		Message:         &msg,
	}
}

// Treat empty strings as omitted so partial updates do not wipe existing fields.
func applyUserProfileUpdate(user *models.User, body operations.PostUsersProfileBody) (*models.User, bool) {
	updated := *user
	changed := false

	if name := strings.TrimSpace(body.Name); name != "" && name != user.Name {
		updated.Name = name
		changed = true
	}
	if address := strings.TrimSpace(body.Address); address != "" && address != user.Address {
		updated.Address = address
		changed = true
	}
	if !swag.IsZero(body.Dob) {
		dob := strings.TrimSpace(body.Dob.String())
		if dob != "" && dob != user.Dob {
			updated.Dob = dob
			changed = true
		}
	}

	return &updated, changed
}

func userProfileResponsePayload(message string, user *models.User) *swgm.UserProfileResponse {
	success := true
	ack := message
	id := user.ID
	email := strfmt.Email(user.Email)
	createdAt := strfmt.DateTime(user.CreatedAt)

	return &swgm.UserProfileResponse{
		Success:         &success,
		Acknowledgement: &ack,
		User: &swgm.UserProfile{
			ID:        &id,
			Email:     &email,
			Name:      user.Name,
			Dob:       user.Dob,
			Address:   user.Address,
			CreatedAt: &createdAt,
		},
	}
}
