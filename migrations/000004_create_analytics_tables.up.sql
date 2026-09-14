CREATE TABLE IF NOT EXISTS outbox_events (
    id VARCHAR(36) PRIMARY KEY,
    event_type VARCHAR(255) NOT NULL,
    aggregate_type VARCHAR(255) NOT NULL,
    aggregate_id VARCHAR(36) NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ NULL,
    attempts INT NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_outbox_events_published_at ON outbox_events(published_at) WHERE published_at IS NULL;

CREATE TABLE IF NOT EXISTS click_events (
    id VARCHAR(36) PRIMARY KEY,
    event_id VARCHAR(36) NOT NULL,
    url_id VARCHAR(36) NOT NULL,
    country VARCHAR(255) NULL,
    device VARCHAR(255) NULL,
    browser VARCHAR(255) NULL,
    operating_system VARCHAR(255) NULL,
    referrer TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_click_events_event_id UNIQUE (event_id),
    CONSTRAINT fk_click_events_url_id FOREIGN KEY (url_id) REFERENCES urls (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_click_events_url_id ON click_events(url_id);
