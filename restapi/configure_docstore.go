// This file is safe to edit. Once it exists it will not be overwritten

package restapi

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"

	"github.com/imsoul11/personalDocStore/internal/app"
	appapi "github.com/imsoul11/personalDocStore/internal/app/api"
	"github.com/imsoul11/personalDocStore/restapi/operations"
)

//go:generate swagger generate server --target ../../docStore --name Docstore --spec ../swagger.yaml --principal interface{}

func configureFlags(api *operations.DocstoreAPI) {
	// api.CommandLineOptionsGroups = []swag.CommandLineOptionsGroup{ ... }
}

func configureAPI(api *operations.DocstoreAPI) http.Handler {
	// Initialise the application (DB, store, API implementers).
	app.Init()

	// configure the api here
	api.ServeError = errors.ServeError

	// Set your custom logger if needed. Default one is log.Printf
	// Expected interface func(string, ...interface{})
	//
	// Example:
	// api.Logger = log.Printf

	api.UseSwaggerUI()
	// To continue using redoc as your UI, uncomment the following line
	// api.UseRedoc()

	api.JSONConsumer = runtime.JSONConsumer()
	api.MultipartformConsumer = runtime.DiscardConsumer

	api.JSONProducer = runtime.JSONProducer()

	// BearerAuth: validates JWT token and returns the user ID as principal.
	api.BearerAuth = func(tokenStr string) (interface{}, error) {
			// Strip "Bearer " prefix if present
			tokenStr = strings.TrimPrefix(tokenStr, "Bearer ")

			jwtSecret := os.Getenv("JWT_SECRET")
			if jwtSecret == "" {
				jwtSecret = "changeme"
			}

			token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
				}
				return []byte(jwtSecret), nil
			})
			if err != nil || !token.Valid {
				return nil, errors.New(401, "invalid token")
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				return nil, errors.New(401, "invalid token claims")
			}

			userIDFloat, ok := claims["user_id"].(float64)
			if !ok {
				return nil, errors.New(401, "invalid user_id in token")
			}

			return int64(userIDFloat), nil
	}

	// Set your custom authorizer if needed. Default one is security.Authorized()
	// Expected interface runtime.Authorizer
	//
	// Example:
	// api.APIAuthorizer = security.Authorized()
	// You may change here the memory limit for this multipart form parser. Below is the default (32 MB).
	// operations.PostDocumentsMaxParseMemory = 32 << 20

	api.GetDocumentsHandler = operations.GetDocumentsHandlerFunc(func(params operations.GetDocumentsParams, principal interface{}) middleware.Responder {
		return appapi.Cfg.DocumentsAPI.GetDocuments(params.HTTPRequest.Context(), params, principal)
	})

	api.GetDocumentsIDHandler = operations.GetDocumentsIDHandlerFunc(func(params operations.GetDocumentsIDParams, principal interface{}) middleware.Responder {
		return middleware.NotImplemented("operation operations.GetDocumentsID has not yet been implemented")
	})

	api.GetUsersProfileHandler = operations.GetUsersProfileHandlerFunc(func(params operations.GetUsersProfileParams, principal interface{}) middleware.Responder {
		return middleware.NotImplemented("operation operations.GetUsersProfile has not yet been implemented")
	})

	api.PostDocumentsHandler = operations.PostDocumentsHandlerFunc(func(params operations.PostDocumentsParams, principal interface{}) middleware.Responder {
		return middleware.NotImplemented("operation operations.PostDocuments has not yet been implemented")
	})

	api.PostUsersLoginHandler = operations.PostUsersLoginHandlerFunc(func(params operations.PostUsersLoginParams) middleware.Responder {
		return appapi.Cfg.UsersAPI.PostUsersLogin(params.HTTPRequest.Context(), params)
	})

	api.PostUsersProfileHandler = operations.PostUsersProfileHandlerFunc(func(params operations.PostUsersProfileParams, principal interface{}) middleware.Responder {
		return middleware.NotImplemented("operation operations.PostUsersProfile has not yet been implemented")
	})

	api.PostUsersRegisterHandler = operations.PostUsersRegisterHandlerFunc(func(params operations.PostUsersRegisterParams) middleware.Responder {
		return appapi.Cfg.UsersAPI.PostUsersRegister(params.HTTPRequest.Context(), params)
	})

	api.PreServerShutdown = func() {}

	api.ServerShutdown = func() {}

	return setupGlobalMiddleware(api.Serve(setupMiddlewares))
}

// The TLS configuration before HTTPS server starts.
func configureTLS(tlsConfig *tls.Config) {
	// Make all necessary changes to the TLS configuration here.
}

// As soon as server is initialized but not run yet, this function will be called.
// If you need to modify a config, store server instance to stop it individually later, this is the place.
// This function can be called multiple times, depending on the number of serving schemes.
// scheme value will be set accordingly: "http", "https" or "unix".
func configureServer(s *http.Server, scheme, addr string) {
}

// The middleware configuration is for the handler executors. These do not apply to the swagger.json document.
// The middleware executes after routing but before authentication, binding and validation.
func setupMiddlewares(handler http.Handler) http.Handler {
	return handler
}

// The middleware configuration happens before anything, this middleware also applies to serving the swagger.json document.
// So this is a good place to plug in a panic handling middleware, logging and metrics.
func setupGlobalMiddleware(handler http.Handler) http.Handler {
	return handler
}
