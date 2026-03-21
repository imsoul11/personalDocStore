-- +goose Up
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    dob VARCHAR(255),
    name VARCHAR(255),
    address VARCHAR(255)
);

-- +goose Down
DROP TABLE IF EXISTS users;
