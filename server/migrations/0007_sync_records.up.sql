CREATE TABLE sync_records (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id     UUID NOT NULL UNIQUE REFERENCES tasks(id) ON DELETE CASCADE,
    status      TEXT NOT NULL DEFAULT 'pending'
                CHECK (status IN ('pending','synced','failed')),
    response    JSONB,
    attempts    INT  NOT NULL DEFAULT 0,
    synced_at   TIMESTAMPTZ,
    last_error  TEXT
);

CREATE INDEX idx_sync_pending ON sync_records(status)
    WHERE status IN ('pending','failed');
