package persistence

import (
	"context"
	"github.com/go-pg/pg/v10"
	pkglog "github.com/imsoul11/personalDocStore/internal/pkg/log"
	"github.com/imsoul11/personalDocStore/internal/pkg/models"
)

func (pgstr *PGStore) CreateUser(ctx context.Context, user *models.User) error {
	log := pkglog.Logger()
	log.Debug().Str("op", "store_create_user").Str("email", user.Email).Msg("creating user")
	_, err := pgstr.db.ModelContext(ctx, user).Insert()
	if err != nil {
		log.Error().Str("op", "store_create_user").Str("email", user.Email).Err(err).Msg("create user failed")
		return err
	}
	log.Debug().Str("op", "store_create_user").Str("email", user.Email).Int64("user_id", user.ID).Msg("user created")
	return nil
}

func (pgstr *PGStore) GetUserByID(ctx context.Context, id int64) (*models.User, error) {
	log := pkglog.Logger()
	log.Debug().Str("op", "store_get_user_by_id").Int64("user_id", id).Msg("fetching user by id")
	user := &models.User{ID: id}
	err := pgstr.db.ModelContext(ctx, user).WherePK().Select()
	if err != nil {
		if err == pg.ErrNoRows {
			log.Debug().Str("op", "store_get_user_by_id").Int64("user_id", id).Msg("user not found")
			return nil, nil
		}
		log.Error().Str("op", "store_get_user_by_id").Int64("user_id", id).Err(err).Msg("get user by id failed")
		return nil, err
	}
	log.Debug().Str("op", "store_get_user_by_id").Int64("user_id", id).Msg("user fetched")
	return user, nil
}

func (pgstr *PGStore) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	log := pkglog.Logger()
	log.Debug().Str("op", "store_get_user_by_email").Str("email", email).Msg("fetching user by email")
	user := new(models.User)
	err := pgstr.db.ModelContext(ctx, user).Where("email = ?", email).Select()
	if err != nil {
		if err == pg.ErrNoRows {
			log.Debug().Str("op", "store_get_user_by_email").Str("email", email).Msg("user not found")
			return nil, nil
		}
		log.Error().Str("op", "store_get_user_by_email").Str("email", email).Err(err).Msg("get user by email failed")
		return nil, err
	}
	log.Debug().Str("op", "store_get_user_by_email").Str("email", email).Int64("user_id", user.ID).Msg("user fetched")
	return user, nil
}

func (pgstr *PGStore) UpdateUser(ctx context.Context, user *models.User) error {
	log := pkglog.Logger()
	log.Debug().Str("op", "store_update_user").Str("email", user.Dob).Str("name",user.Name).Msg("creating user")
	_, err := pgstr.db.ModelContext(ctx, user).WherePK().Update()
	if err != nil {
		log.Error().Str("op", "store_update_user").Str("email", user.Email).Err(err).Msg("update user failed")
		return err
	}
	log.Debug().Str("op", "store_update_user").Str("email", user.Email).Int64("user_id", user.ID).Msg("user updated")
	return nil
}



