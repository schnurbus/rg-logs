-- +migrate Up
-- Wipe legacy anonymous uploads (no ownership / storage paths).
TRUNCATE TABLE uploads CASCADE;

ALTER TABLE uploads
    ADD COLUMN IF NOT EXISTS user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    ADD COLUMN IF NOT EXISTS content_hash TEXT NOT NULL,
    ADD COLUMN IF NOT EXISTS is_private BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS storage_path TEXT NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uploads_user_hash_uidx
    ON uploads (user_id, content_hash);

CREATE INDEX IF NOT EXISTS idx_uploads_user_id ON uploads (user_id);
CREATE INDEX IF NOT EXISTS idx_uploads_public ON uploads (created_at DESC)
    WHERE is_private = FALSE;
