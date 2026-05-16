-- gosom-style place schema (mirrors gmaps.Entry)
CREATE TABLE places (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id             UUID REFERENCES tasks(id) ON DELETE SET NULL,
    kelurahan_id        UUID REFERENCES kelurahan(id),
    keyword             TEXT NOT NULL,

    place_id            TEXT UNIQUE NOT NULL,
    data_id             TEXT,
    cid                 TEXT,

    title               TEXT NOT NULL,
    categories          TEXT[] NOT NULL DEFAULT '{}',
    category            TEXT,
    address             TEXT,
    complete_address    JSONB,

    open_hours          JSONB,
    popular_times       JSONB,

    website             TEXT,
    phone               TEXT,
    plus_code           TEXT,

    review_count        INT NOT NULL DEFAULT 0,
    review_rating       DOUBLE PRECISION,
    reviews_per_rating  JSONB,

    latitude            DOUBLE PRECISION,
    longitude           DOUBLE PRECISION,

    status              TEXT NOT NULL DEFAULT 'active',
    description         TEXT,
    reviews_link        TEXT,
    thumbnail           TEXT,
    timezone            TEXT,
    price_range         TEXT,

    images              JSONB NOT NULL DEFAULT '[]'::jsonb,
    reservations        JSONB NOT NULL DEFAULT '[]'::jsonb,
    order_online        JSONB NOT NULL DEFAULT '[]'::jsonb,
    menu                JSONB,
    owner               JSONB,
    about               JSONB NOT NULL DEFAULT '[]'::jsonb,
    emails              TEXT[] NOT NULL DEFAULT '{}',

    scraped_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_places_kelurahan_keyword ON places(kelurahan_id, keyword);
CREATE INDEX idx_places_keyword ON places(keyword);
CREATE INDEX idx_places_categories ON places USING GIN (categories);
