// This file is safe to edit. Once it exists it will not be overwritten

package restapi

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"

	"github.com/imsoul11/personalDocStore/internal/app"
	appapi "github.com/imsoul11/personalDocStore/internal/app/api"
	pkglog "github.com/imsoul11/personalDocStore/internal/pkg/log"
	"github.com/imsoul11/personalDocStore/restapi/operations"
)

//go:generate swagger generate server --target ../../docStore --name Docstore --spec ../swagger.yaml --principal interface{}

func configureFlags(api *operations.DocstoreAPI) {
	// api.CommandLineOptionsGroups = []swag.CommandLineOptionsGroup{ ... }
}

func configureAPI(api *operations.DocstoreAPI) http.Handler {
	log := pkglog.Logger()
	log.Info().Str("op", "configure_api").Msg("starting API configuration")
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
	log.Info().Str("op", "configure_api").Msg("consumers and producers configured")

	// BearerAuth: validates JWT token and returns the user ID as principal.
	api.BearerAuth = func(tokenStr string) (interface{}, error) {
			log := pkglog.Logger()
			// Strip "Bearer " prefix if present
			tokenStr = strings.TrimPrefix(tokenStr, "Bearer ")

			jwtSecret := os.Getenv("JWT_SECRET")
			if jwtSecret == "" {
				jwtSecret = "changeme"
			}

			token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					log.Warn().Str("op", "bearer_auth").Interface("alg", t.Header["alg"]).Msg("unexpected signing method")
					return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
				}
				return []byte(jwtSecret), nil
			})
			if err != nil || !token.Valid {
				log.Warn().Str("op", "bearer_auth").Err(err).Msg("token validation failed")
				return nil, errors.New(401, "invalid token")
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				log.Warn().Str("op", "bearer_auth").Msg("invalid token claims type")
				return nil, errors.New(401, "invalid token claims")
			}

			userIDFloat, ok := claims["user_id"].(float64)
			if !ok {
				log.Warn().Str("op", "bearer_auth").Msg("user_id claim missing or invalid")
				return nil, errors.New(401, "invalid user_id in token")
			}
			userID := int64(userIDFloat)
			log.Debug().Str("op", "bearer_auth").Int64("user_id", userID).Msg("token validated")
			return userID, nil
	}

	// Set your custom authorizer if needed. Default one is security.Authorized()
	// Expected interface runtime.Authorizer
	//
	// Example:
	// api.APIAuthorizer = security.Authorized()
	// You may change here the memory limit for this multipart form parser. Below is the default (32 MB).
	// operations.PostDocumentsMaxParseMemory = 32 << 20

	api.GetDocumentsHandler = operations.GetDocumentsHandlerFunc(func(params operations.GetDocumentsParams, principal interface{}) middleware.Responder {
		log.Debug().Str("op", "route_get_documents").Msg("delegating to documents API")
		return appapi.Cfg.DocumentsAPI.GetDocuments(params.HTTPRequest.Context(), params, principal)
	})

	api.GetDocumentsIDHandler = operations.GetDocumentsIDHandlerFunc(func(params operations.GetDocumentsIDParams, principal interface{}) middleware.Responder {
		log.Debug().Str("op", "route_get_documents_id").Msg("handler hit (not implemented)")
		return middleware.NotImplemented("operation operations.GetDocumentsID has not yet been implemented")
	})

	api.GetUsersProfileHandler = operations.GetUsersProfileHandlerFunc(func(params operations.GetUsersProfileParams, principal interface{}) middleware.Responder {
		log.Debug().Str("op", "route_get_users_profile").Msg("delegating to users API")
		return appapi.Cfg.UsersAPI.GetUsersProfile(params.HTTPRequest.Context(), params, principal)
	})

	api.PostDocumentsHandler = operations.PostDocumentsHandlerFunc(func(params operations.PostDocumentsParams, principal interface{}) middleware.Responder {
		log.Debug().Str("op", "route_post_documents").Msg("delegating to documents API")
		return appapi.Cfg.DocumentsAPI.PostDocuments(params.HTTPRequest.Context(), params, principal)
	})

	api.PostUsersLoginHandler = operations.PostUsersLoginHandlerFunc(func(params operations.PostUsersLoginParams) middleware.Responder {
		log.Debug().Str("op", "route_post_users_login").Msg("delegating to users API")
		return appapi.Cfg.UsersAPI.PostUsersLogin(params.HTTPRequest.Context(), params)
	})

	api.PostUsersProfileHandler = operations.PostUsersProfileHandlerFunc(func(params operations.PostUsersProfileParams, principal interface{}) middleware.Responder {
		log.Debug().Str("op", "route_post_users_profile").Msg("delegating to users API")
		return appapi.Cfg.UsersAPI.PostUsersProfile(params.HTTPRequest.Context(), params, principal)
	})

	api.PostUsersRegisterHandler = operations.PostUsersRegisterHandlerFunc(func(params operations.PostUsersRegisterParams) middleware.Responder {
		log.Debug().Str("op", "route_post_users_register").Msg("delegating to users API")
		return appapi.Cfg.UsersAPI.PostUsersRegister(params.HTTPRequest.Context(), params)
	})

	api.PreServerShutdown = func() {
		log.Info().Str("op", "server_shutdown").Msg("pre server shutdown hook called")
	}

	api.ServerShutdown = func() {
		log.Info().Str("op", "server_shutdown").Msg("server shutdown completed")
	}
	log.Info().Str("op", "configure_api").Msg("API configuration complete")

	return setupGlobalMiddleware(api.Serve(setupMiddlewares))
}

// The TLS configuration before HTTPS server starts.
func configureTLS(tlsConfig *tls.Config) {
	// Make all necessary changes to the TLS configuration here.
	_ = tlsConfig
	pkglog.Logger().Info().Str("op", "configure_tls").Msg("configureTLS invoked")
}

// As soon as server is initialized but not run yet, this function will be called.
// If you need to modify a config, store server instance to stop it individually later, this is the place.
// This function can be called multiple times, depending on the number of serving schemes.
// scheme value will be set accordingly: "http", "https" or "unix".
func configureServer(s *http.Server, scheme, addr string) {
	pkglog.Logger().Info().Str("op", "configure_server").Str("scheme", scheme).Str("addr", addr).Msg("configureServer invoked")
	_ = s
}

// The middleware configuration is for the handler executors. These do not apply to the swagger.json document.
// The middleware executes after routing but before authentication, binding and validation.
func setupMiddlewares(handler http.Handler) http.Handler {
	log := pkglog.Logger()
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		log.Debug().Str("op", "middleware_stage").Str("stage", "executor").Str("method", r.Method).Str("path", r.URL.Path).Msg("request entering executor middleware")
		handler.ServeHTTP(rw, r)
		log.Debug().Str("op", "middleware_stage").Str("stage", "executor").Str("method", r.Method).Str("path", r.URL.Path).Msg("request exiting executor middleware")
	})
}

// The middleware configuration happens before anything, this middleware also applies to serving the swagger.json document.
// So this is a good place to plug in a panic handling middleware, logging and metrics.
func setupGlobalMiddleware(handler http.Handler) http.Handler {
	log := pkglog.Logger()
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		start := time.Now()
		handler.ServeHTTP(rw, r)
		log.Info().
			Str("op", "http_request").
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Dur("latency", time.Since(start)).
			Msg("request handled")
	})
}
