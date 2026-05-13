# Scrapper-Go Rewrite (Phase 1)

This branch (`feat/go-rewrite`) replaces the Python+FastAPI scraper with a
**Go + Gin + GORM + Playwright-Go** stack and introduces a worker-based
architecture so we no longer have to compute shards by hand.

> **Status:** Phase 1 foundation only — the scraping engine itself is a
> `NoopExecutor` stub that simulates work. Phase 3 swaps in real
> Playwright-Go scraping. See [plan file](C:\Users\Hype G12\.claude\plans\buat-plannya-dulu-go-iridescent-lemur.md)
> for the full roadmap.

## Quick Start (dev)

```bash
# 1. Copy env template and set a strong MASTER_TOKEN.
cp .env.example .env
$EDITOR .env     # set POSTGRES_PASSWORD and MASTER_TOKEN

# 2. Bring up the stack (PostgreSQL + master + 1 local worker × 2 slots).
docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.dev.yml up -d --build

# 3. Seed Bandung (calls emsifa via the master API).
./scripts/seed-bandung.sh

# 4. Hit the API.
curl http://localhost:8080/api/v1/health
curl http://localhost:8080/api/v1/workers
curl http://localhost:8080/api/v1/cities/bandung
```

## What's wired up

- **PostgreSQL 16** with 7 migrations (cities, kelurahan, workers, jobs, tasks,
  places, reviews, sync_records).
- **Gin master** at `:8080` exposing `/api/v1/*` for cities/jobs/tasks/workers
  and `/api/v1/internal/*` for worker-only endpoints (Bearer `MASTER_TOKEN`).
- **GORM repositories** (`internal/storage/*_repo.go`) using `pgx` driver.
- **Postgres-backed queue** using `FOR UPDATE SKIP LOCKED` so any number of
  workers can dequeue concurrently with zero coordination.
- **Reaper** goroutine that re-queues tasks whose worker stopped heart-beating
  and marks stale workers offline.
- **Emsifa client + seeder** to populate cities/kelurahan from
  `emsifa.github.io/api-wilayah-indonesia`.
- **Worker binary** that registers, heart-beats, claims tasks, and (currently)
  runs the `NoopExecutor` — enough to verify the queue end-to-end.

## Folder map

```
cmd/master/      # master entry: serve | migrate | healthcheck
cmd/worker/      # worker entry
internal/
  api/           # Gin routes + handlers (cities, jobs, tasks, workers, places, sync, internal)
  config/        # viper-backed config for master & worker
  domain/        # Pure types + GORM models + gosom-style PlacePayload
  emsifa/        # emsifa.github.io client + city/kelurahan seeder
  logger/        # zerolog setup
  queue/         # PostgresQueue (SKIP LOCKED) + Reaper
  storage/       # GORM repositories + golang-migrate runner
  version/       # build-time version
  workeragent/   # worker-side: client + agent loop + executors
migrations/      # 0001..0007 SQL up/down files
deploy/          # Dockerfiles + 3 compose files (full, dev override, worker-only)
scripts/         # seed-bandung.sh, migrate.sh
web/             # existing Svelte UI (now includes lib/ws.ts stub)
```

## Phase 1 acceptance test

The single most important thing: **dead-worker recovery**.

1. Create a job (currently you need to seed cities + kelurahan first):
   ```bash
   ./scripts/seed-bandung.sh
   CITY_ID=$(curl -s http://localhost:8080/api/v1/cities/bandung | jq -r '.city.id')
   curl -X POST http://localhost:8080/api/v1/jobs \
     -H 'content-type: application/json' \
     -d "{\"city_id\":\"$CITY_ID\",\"keyword\":\"cafe\",\"kelurahan_names\":[\"Cihapit\",\"Tamansari\",\"Antapani Tengah\"]}"
   ```
2. Watch `/api/v1/tasks` — they should flip queued → in_progress → done.
3. While a task is in_progress, `docker kill scrapper-worker-1`.
4. After ~2 minutes (`DEAD_AFTER_MIN`) the reaper re-queues it; another worker
   picks it up automatically.

## What is NOT done yet (later phases)

- **Phase 2** — minor: extra emsifa caching, more UI components.
- **Phase 3** — the actual scraping (Playwright-Go port of Python `gmaps.py`).
- **Phase 4** — WebSocket real-time UI (the `ws.ts` store is a stub).
- **Phase 5** — sync to external API, Prometheus metrics, full deploy CI.
