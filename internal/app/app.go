package app

import (
	"log"
	"os"

	appapi "github.com/imsoul11/personalDocStore/internal/app/api"
	"github.com/imsoul11/personalDocStore/internal/pkg/config"
	"github.com/imsoul11/personalDocStore/internal/pkg/db"
	"github.com/imsoul11/personalDocStore/internal/pkg/persistence"
)

// Init initialises the application: loads config, creates DB, creates the
// persistence store, and wires the API implementers.
// It is idempotent – calling it more than once is safe.
// Must be called before the swagger server starts handling requests
// (i.e. from configureAPI in restapi/configure_docstore.go).
// Config path is read from CONFIG_PATH env var; defaults to "configs/config.json".
// JWT secret is read from JWT_SECRET env var; defaults to "changeme".
func Init() {
	if appapi.Cfg != nil {
		return
	}

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "configs/config.json"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "changeme"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("app.Init: config load failed: %v", err)
	}

	dbInstance, err := db.New(cfg.Database)
	if err != nil {
		log.Fatalf("app.Init: db connect failed: %v", err)
	}

	store := persistence.New(dbInstance)

	appapi.Cfg = &appapi.Config{
		DocumentsAPI: appapi.NewDocuments(store),
		UsersAPI:     appapi.NewUsers(store, jwtSecret),
	}
}
