CREATE TABLE workers (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT NOT NULL,
    hostname        TEXT,
    ip_addr         INET,
    max_concurrency INT NOT NULL DEFAULT 2 CHECK (max_concurrency > 0 AND max_concurrency <= 16),
    capabilities    JSONB NOT NULL DEFAULT '{}'::jsonb,
    status          TEXT NOT NULL DEFAULT 'offline'
                    CHECK (status IN ('online','offline','draining')),
    last_heartbeat  TIMESTAMPTZ,
    registered_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    metadata        JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX idx_workers_status_heartbeat ON workers(status, last_heartbeat);
CREATE UNIQUE INDEX idx_workers_name ON workers(name);
