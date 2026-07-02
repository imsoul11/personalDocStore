-- +goose Up
ALTER TABLE documents
    ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

UPDATE documents
SET updated_at = created_at
WHERE updated_at IS NULL;

-- +goose Down
ALTER TABLE documents
    DROP COLUMN IF EXISTS updated_at;
