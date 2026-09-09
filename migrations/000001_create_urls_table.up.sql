CREATE TABLE IF NOT EXISTS urls (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NULL,
    short_code VARCHAR(64) NOT NULL,
    original_url TEXT NOT NULL,
    expires_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL,
    CONSTRAINT uq_urls_short_code UNIQUE (short_code)
);

CREATE INDEX IF NOT EXISTS idx_urls_expires_at ON urls (expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_urls_user_id ON urls (user_id) WHERE user_id IS NOT NULL;
