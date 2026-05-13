CREATE TABLE jobs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    city_id       UUID NOT NULL REFERENCES cities(id),
    keyword       TEXT NOT NULL,
    status        TEXT NOT NULL DEFAULT 'pending'
                  CHECK (status IN ('pending','running','completed','failed','cancelled')),
    options       JSONB NOT NULL DEFAULT '{}'::jsonb,
    total_tasks   INT  NOT NULL DEFAULT 0,
    done_count    INT  NOT NULL DEFAULT 0,
    failed_count  INT  NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at    TIMESTAMPTZ,
    completed_at  TIMESTAMPTZ,
    created_by    TEXT
);

CREATE INDEX idx_jobs_status ON jobs(status);
CREATE INDEX idx_jobs_city_keyword ON jobs(city_id, keyword);

CREATE TABLE tasks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id          UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    kelurahan_id    UUID NOT NULL REFERENCES kelurahan(id),
    priority        INT  NOT NULL DEFAULT 0,
    status          TEXT NOT NULL DEFAULT 'queued'
                    CHECK (status IN ('queued','in_progress','done','failed','cancelled')),
    worker_id       UUID REFERENCES workers(id),
    attempt         INT  NOT NULL DEFAULT 0,
    max_attempts    INT  NOT NULL DEFAULT 3,
    visible_after   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_heartbeat  TIMESTAMPTZ,
    last_error      TEXT,
    result_path     TEXT,
    places_count    INT,
    enqueued_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    UNIQUE (job_id, kelurahan_id)
);

-- Critical: queue dequeue index (partial)
CREATE INDEX idx_tasks_queue ON tasks(priority DESC, visible_after, enqueued_at)
    WHERE status = 'queued';
CREATE INDEX idx_tasks_running ON tasks(worker_id, last_heartbeat)
    WHERE status = 'in_progress';
CREATE INDEX idx_tasks_job ON tasks(job_id, status);
