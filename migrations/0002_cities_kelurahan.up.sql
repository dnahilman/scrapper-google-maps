CREATE TABLE cities (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    emsifa_regency_id    TEXT UNIQUE NOT NULL,
    emsifa_province_id   TEXT NOT NULL,
    name                 TEXT NOT NULL,
    slug                 CITEXT UNIQUE NOT NULL,
    province_name        TEXT NOT NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_cities_province ON cities(emsifa_province_id);

CREATE TRIGGER cities_updated_at
    BEFORE UPDATE ON cities
    FOR EACH ROW EXECUTE FUNCTION trg_set_updated_at();

CREATE TABLE kelurahan (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    city_id             UUID NOT NULL REFERENCES cities(id) ON DELETE CASCADE,
    emsifa_village_id   TEXT UNIQUE NOT NULL,
    emsifa_district_id  TEXT NOT NULL,
    name                TEXT NOT NULL,
    kecamatan_name      TEXT NOT NULL,
    code                TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_kelurahan_city ON kelurahan(city_id);
CREATE INDEX idx_kelurahan_city_name ON kelurahan(city_id, name);
CREATE INDEX idx_kelurahan_district ON kelurahan(emsifa_district_id);
