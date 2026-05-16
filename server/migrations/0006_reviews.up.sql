CREATE TABLE reviews (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    place_id        TEXT NOT NULL REFERENCES places(place_id) ON DELETE CASCADE,
    review_id       TEXT,
    name            TEXT,
    profile_picture TEXT,
    rating          INT CHECK (rating BETWEEN 1 AND 5),
    description     TEXT,
    images          TEXT[] NOT NULL DEFAULT '{}',
    when_text       TEXT,
    age_days        INT,
    owner_response  JSONB,
    extended        BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (place_id, review_id)
);

CREATE INDEX idx_reviews_place_rating ON reviews(place_id, rating);
CREATE INDEX idx_reviews_age ON reviews(age_days);
