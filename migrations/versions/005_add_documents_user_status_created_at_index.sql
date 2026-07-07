-- +goose Up
CREATE INDEX idx_documents_user_id_status_created_at_desc
    ON documents (user_id, status, created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_documents_user_id_status_created_at_desc;
