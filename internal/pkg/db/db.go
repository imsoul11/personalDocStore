package db

import (
	"context"

	"github.com/go-pg/pg/v10"
	"github.com/imsoul11/personalDocStore/internal/pkg/config"
)

func New(dbCfg config.DatabaseConfig) (*pg.DB, error){
	pgOptions,err := pg.ParseURL(dbCfg.URL)
	if err!=nil{
		return nil,err
	}
	db := pg.Connect(pgOptions)
	if err := db.Ping(context.Background()); err != nil {
    _ = db.Close()
    return nil, err
    }
    return db, nil
}