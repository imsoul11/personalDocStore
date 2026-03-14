package models

import "time"

type User struct{
    ID               int64        `pg:"id,pk"`
    Email            string       `pg:"email"`
    PasswordHash    string       `pg:"password_hash"`
    CreatedAt       time.Time    `pg:"created_at"`
}

