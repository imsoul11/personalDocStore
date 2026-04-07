package app

import (
	"context"
	"log"
	"os"

	appapi "github.com/imsoul11/personalDocStore/internal/app/api"
	"github.com/imsoul11/personalDocStore/internal/pkg/config"
	"github.com/imsoul11/personalDocStore/internal/pkg/db"
	pkglog "github.com/imsoul11/personalDocStore/internal/pkg/log"
	"github.com/imsoul11/personalDocStore/internal/pkg/persistence"
	"github.com/imsoul11/personalDocStore/internal/pkg/queue/rabbitmq"
)

var jwtSecretValue string

func JWTSecret() string {
	return jwtSecretValue
}

// Init initialises the application: loads config, creates DB, creates the
// persistence store, and wires the API implementers.
// It is idempotent – calling it more than once is safe.
// Must be called before the swagger server starts handling requests
// (i.e. from configureAPI in restapi/configure_docstore.go).
// Config path is read from CONFIG_PATH env var; defaults to "configs/config.json".
// JWT secret is read from JWT_SECRET env var; defaults to "changeme".
func Init() {
	if appapi.Cfg != nil {
		pkglog.Logger().Debug().Str("op", "app_init").Msg("app config already initialized")
		return
	}

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "configs/config.json"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "caskjdbaiudhhiadiasahiassdiuashdisaundasjn"
	}
	jwtSecretValue = jwtSecret

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("app.Init: config load failed: %v", err)
	}
	pkglog.New(cfg.Log.Level)
	pkglog.Logger().Info().Str("op", "app_init").Str("config_path", configPath).Msg("config loaded")

	dbInstance, err := db.New(cfg.Database)
	if err != nil {
		log.Fatalf("app.Init: db connect failed: %v", err)
	}
	pkglog.Logger().Info().Str("op", "app_init").Msg("database connection established")

	ctx := context.Background()
	broker := rabbitmq.New(ctx, cfg.RabbitMQ.URL)
	pkglog.Logger().Info().Str("op", "app_init").Msg("rabbitmq connection established")

	store := persistence.New(dbInstance)

	appapi.Cfg = &appapi.Config{
		DocumentsAPI: appapi.NewDocuments(store, broker),
		UsersAPI:     appapi.NewUsers(store, jwtSecret),
	}
	pkglog.Logger().Info().Str("op", "app_init").Msg("application APIs wired")
}
