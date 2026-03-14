package persistence

import "github.com/go-pg/pg/v10"

type PGStore struct {
    db *pg.DB
}

func New(db *pg.DB) *PGStore{
    return &PGStore{db: db}
}

