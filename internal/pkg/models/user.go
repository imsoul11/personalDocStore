package models

import "time"

type User struct{
    ID               int64        `pg:"id,pk"`
    Email            string       `pg:"email"`
    PasswordHash    string       `pg:"password_hash"`
    CreatedAt       time.Time    `pg:"created_at"`
    Dob             string        `pg:"dob"`
    Name            string        `pg:"name"`
    Address         string        `pg:"address"`     
}
 
