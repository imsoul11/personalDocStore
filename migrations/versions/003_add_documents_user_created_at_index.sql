-- +goose Up
CREATE INDEX idx_documents_user_id_created_at_desc
    ON documents (user_id, created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_documents_user_id_created_at_desc;
