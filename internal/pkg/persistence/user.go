package persistence

import (
	"context"
	"github.com/go-pg/pg/v10"
	"github.com/imsoul11/personalDocStore/internal/pkg/models"
)

func (pgstr *PGStore) CreateUser(ctx context.Context, user *models.User) error {
    _, err := pgstr.db.ModelContext(ctx, user).Insert()
	if err!=nil{
		return err
	}
	return nil
}

func (pgstr *PGStore) GetUserByID(ctx context.Context, id int64) (*models.User, error) {
	user := &models.User{ID: id}
	err := pgstr.db.ModelContext(ctx, user).WherePK().Select()
	if err != nil {
		if err == pg.ErrNoRows {
			return nil, nil 
		}
		return nil, err
	}
	return user, nil
}

func (pgstr *PGStore) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	user := new(models.User)
	err := pgstr.db.ModelContext(ctx, user).Where("email = ?", email).Select()
	if err != nil {
		if err == pg.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return user, nil
}

