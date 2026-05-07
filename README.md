# Scrapper Google Maps (Multi-keyword)

Scraper Google Maps untuk mengumpulkan data **place** (cafe, barbershop, kuliner, restaurant, dll.) di Kota Bandung, scope per **kelurahan**. Output berupa **JSON SyncItem-ready** + tracking SQLite per keyword.

```bash
./scr scrape cafe --kelurahan "Cihapit" --limit 5     # local re-scrape
./scr scrape cafe                                      # full run, semua kelurahan
./scr --vps scrape cafe --auto-sync                   # production di VPS
```

> Dokumentasi helper: [`docs/scr.md`](docs/scr.md)

---

## Web UI (FastAPI + Svelte)

Single Docker image bundles **FastAPI** (control plane API) + **Svelte** (UI ringan, no SvelteKit). Setelah `docker compose up -d`, buka `http://localhost:8000`:

- **Dashboard** — progress per keyword (done/total), KPI active jobs, data dir size
- **Jobs** — start/stop scrape job dengan param keyword, shard, kelurahan filter, limit, resume
- **Files** — browse output JSON per keyword + preview + download
- **Logs** — live tail (SSE) per logfile / per job, dengan filter substring & auto-scroll
- **Health indicator** — dot pulse di header, polling `/api/health` setiap 10 detik

API docs auto-generated di `/docs` (Swagger UI). Bash wrapper `./scrape` lama tetap bisa dipakai parallel — UI dan CLI share SQLite progress.db lewat WAL mode.

> Frontend di-build di stage 1 (Node) lalu di-serve oleh FastAPI dari `web/static`. Tidak ada Node runtime di production image.

## Features

- **Multi-keyword** — `--keyword cafe`, `--keyword barbershop`, `--keyword resto`, dll. Storage isolated per keyword (`data/<keyword>/`).
- **Schema dispatcher** — output cafe/resto otomatis pakai schema extended (menu, payment, gallery, reviewsDistribution, dll). Barbershop tetap pakai schema dasar (backward-compat).
- **Resume tracking** — SQLite per keyword, skip kelurahan yang sudah `done`. Auto-disable saat `--kelurahan` filter dipakai (re-scrape mode).
- **Anti-ban safety** — random delay, CAPTCHA detection, exponential backoff, no Google account login.
- **Dual-mode helper** — `./scr` (bash) atau `./scr.ps1` (PowerShell) untuk LOCAL (docker/podman compose) atau VPS (SSH + remote docker).
- **Auto-detect compose tool** — prefer `docker compose`, fallback `podman-compose`.
- **CI/CD** — push ke `main` → build + push ke GHCR → SSH deploy ke VPS otomatis.

## Stack

| Komponen | Tools |
|---|---|
| Bahasa | Python 3.10+ (wajib karena pakai `int \| None` syntax) |
| Browser | Playwright + Chromium (pre-installed di base image) |
| Anti-detection | playwright-stealth + custom UA rotation |
| HTTP client | httpx (sync API POST) |
| Storage | SQLite (progress + sync tracking) + JSON (output) |
| Container | Docker Compose **atau** Podman Compose |
| Registry | GitHub Container Registry (GHCR) |
| Seed data | Static `data/kelurahan_bandung.json` (151 kelurahan Bandung) |

## Project Structure

```
scrapper/
├── README.md                    # ← Anda di sini
├── .env.example                 # Template credential — copy ke .env.local
├── requirements.txt             # Python deps (playwright pinned)
├── Dockerfile                   # Image generic, no secrets baked
├── docker-compose.dev.yml       # Local dev (build + bind mount source)
├── docker-compose.prod.yml      # VPS prod (pull image dari GHCR)
├── .gitattributes               # Enforce LF untuk shell scripts
├── scr                          # Helper bash (Linux/Mac/WSL/Git Bash)
├── scr.ps1                      # Helper PowerShell (Windows native)
├── config.py                    # Konfigurasi terpusat (env, paths, tuning)
├── src/
│   ├── browser.py               # Playwright stealth setup + UA rotation
│   ├── gmaps.py                 # Scraping logic (search, place, menu, reviews)
│   ├── storage.py               # SQLite + JSON writer
│   ├── transform.py             # Shape dispatcher (barbershop / cafe)
│   ├── sync_client.py           # HTTP client untuk POST ke API
│   ├── seed.py                  # Loader kelurahan
│   └── logger.py
├── scripts/
│   ├── scraper.py               # Entry point CLI scrape
│   ├── sync.py                  # Manual sync ke API
│   └── migrate.py               # One-time schema migration
├── docker/
│   └── entrypoint.sh            # Container entrypoint (LF line endings!)
├── docs/
│   └── scr.md                   # Helper script reference
├── data/
│   ├── kelurahan_bandung.json   # Seed data (tracked di Git)
│   ├── cafe/                    # ← per-keyword output (di-ignore Git)
│   │   ├── progress.db
│   │   └── Cihapit.json
│   └── barbershop/
│       └── ...
├── logs/                        # Log file Python (di-ignore Git)
└── .github/workflows/
    └── build-and-deploy.yml     # CI/CD: build → GHCR → SSH deploy
```

---

## Setup

Tiga mode setup. Pilih sesuai use case:

### Mode 1 — Container (Docker / Podman Compose) — **Recommended**

Setup paling konsisten — semua dependency terisolasi di container, hot-edit source code via volume mount.

```bash
# 1. Clone repo
git clone https://github.com/dnahilman/scrapper-google-maps.git
cd scrapper-google-maps

# 2. Buat .env.local (lihat .env.example)
cp .env.example .env.local
# Edit .env.local — isi APP_URL + GOOGLE_MAPS_SYNC_API_KEY

# 3. Build + start container (sekali — pull base image ~1.7 GB first time)
docker compose -f docker-compose.dev.yml up -d --build
# ATAU pakai podman-compose:
podman-compose -f docker-compose.dev.yml up -d --build

# 4. Verify
./scr ps                   # status container
./scr scrape cafe --kelurahan "Cihapit" --limit 1
```

Helper `./scr` auto-detect compose tool yang tersedia (`docker compose` atau `podman-compose`). Override paksa via `COMPOSE_TOOL=docker` atau `COMPOSE_TOOL=podman`.

### Mode 2 — Pure Python venv

Tanpa container. Cocok untuk debug cepat (browser visible, breakpoint, dll). Trade-off: dep di host Python, no isolation.

```bash
# 1. Python venv (3.10+)
python -m venv .venv
.\.venv\Scripts\Activate.ps1            # Windows PowerShell
# source .venv/bin/activate              # Linux/Mac

# 2. Install deps
pip install -r requirements.txt
playwright install chromium              # download Chromium binary (~280 MB)

# 3. .env.local — sama seperti Mode 1
cp .env.example .env.local

# 4. Run langsung (helper ./scr tidak applicable — itu wrapper container)
python scripts/scraper.py --keyword cafe --kelurahan "Cihapit" --limit 1
python scripts/sync.py --keyword cafe --all
```

Set `HEADLESS=false` di `.env.local` kalau mau lihat browser saat scraping (debugging).

### Mode 3 — VPS Production

Server-side deploy. Image di-pull dari GHCR, `.env.local` di-write CI/CD setiap deploy dari GitHub Secrets.

**Setup VPS sekali:**
```bash
# Di VPS (sebagai user yang punya akses Docker)
sudo mkdir -p /opt/scrapper-google-maps
sudo chown $USER:$USER /opt/scrapper-google-maps
cd /opt/scrapper-google-maps

# Letakkan compose file (dari repo)
curl -O https://raw.githubusercontent.com/dnahilman/scrapper-google-maps/main/docker-compose.prod.yml

# Login GHCR (sekali — kalau image private)
echo "$GHCR_TOKEN" | docker login ghcr.io -u <username> --password-stdin

# Pull + start (manual, atau biarkan CI/CD yang trigger)
docker compose -f docker-compose.prod.yml pull
docker compose -f docker-compose.prod.yml up -d
```

**Setup GitHub Secrets** (sekali, untuk CI/CD auto-deploy):
- `APP_URL` — backend API URL
- `GOOGLE_MAPS_SYNC_API_KEY` — sync API key
- `VPS_HOST_IP` — VPS IP/hostname
- `VPS_SSH_USER` — SSH user
- `VPS_SSH_KEY` — private key (full content)
- `VPS_SSH_PORT` — SSH port (default 22, optional)

**Trigger deploy** dari laptop (atau auto-trigger via push `main`):
```bash
./scr --vps scrape cafe --auto-sync          # start scrape detached
./scr --vps logs                              # tail log
./scr --vps status cafe                       # progress
./scr --vps stop                              # graceful kill
```

CI/CD flow: `push main` → build image → push GHCR → SSH ke VPS → tulis `.env.local` dari secrets → `docker compose pull && up -d`.

---

## Daily Usage

### Helper `./scr`

```bash
./scr scrape <keyword> [args]    # start scrape (background, detached)
./scr sync <keyword> [args]      # POST file JSON ke API
./scr status <keyword>           # progress + sync summary
./scr logs                       # tail log realtime
./scr stop                       # kill scrape process
./scr is-running                 # cek status scrape
./scr shell                      # bash di container
./scr ps                         # container status
./scr redeploy                   # rebuild image (LOCAL) atau pull baru (VPS)
./scr pull <keyword>             # download data dari VPS (VPS-only)
```

Semua command bisa di-prefix `--vps` (bash) atau `-Vps` (PS) untuk VPS mode.

**Detail lengkap, troubleshooting, dan FAQ → [`docs/scr.md`](docs/scr.md)**

### CLI langsung (tanpa helper)

Kalau Anda di pure Python venv atau butuh kontrol penuh:

```bash
python scripts/scraper.py --keyword cafe --kelurahan "Cihapit" --limit 5
python scripts/scraper.py --keyword cafe --resume --auto-sync
python scripts/sync.py --keyword cafe --all --force
python scripts/migrate.py --keyword cafe                   # migrate old schema files
```

Flag scraper:

| Flag | Default | Keterangan |
|---|---|---|
| `--keyword <k>` | `cafe` | Target keyword (cafe/barbershop/resto/dll) — controls storage path & schema dispatcher |
| `--kelurahan <name>` | semua | Filter substring nama kelurahan |
| `--limit N` | unlimited | Maks N place per kelurahan (test mode) |
| `--resume` | off | Skip kelurahan status `done` |
| `--auto-sync` | off | Auto-POST ke API tiap kelurahan selesai |
| `--dry-run` | off | List kelurahan saja, tidak scrape |

---

## Output Schema

Schema otomatis dipilih oleh `transform.get_transformer(keyword)`. Default fallback ke barbershop (16 field). Cafe/resto pakai extended (26 field).

### Schema dasar — barbershop (default)

```jsonc
{
  "name": "Barber X",
  "address": "...",
  "phone": "+62...",
  "location": [107.62, -6.91],         // [lng, lat]
  "openingHours": { "Senin": {"open": "09.00", "close": "21.00"}, ... },
  "description": null,
  "features": ["Haircut", "Shaving", ...],
  "coverImage": "https://...",
  "rating": "4.7",                      // string
  "reviewCount": 234,
  "website": "...",
  "urlGoogleMaps": "...",
  "googlePlaceId": "...",
  "status": "active",
  "claimed": false,
  "reviews": [{"rating": 5, "comment": "...", "photoUrl": null, "guest": {"name": "...", "image": null}}, ...]
}
```

### Schema extended — cafe / resto / kuliner / restaurant / kafe

Trigger: `--keyword cafe` atau `kafe`/`resto`/`restaurant`/`kuliner`.

```jsonc
{
  "name": "Bermula Coffee",
  "address": "...",
  "phone": "...",
  "category": "Coffee shop",            // ← BARU
  "location": [107.62, -6.91],
  "openingHours": {...},
  "description": null,
  "features": [...],                    // semua items dari "Tentang" tab Google Maps
  "coverImage": "...",
  "rating": 4.1,                        // ← float (bukan string)
  "totalReviews": 190,                  // ← rename dari reviewCount
  "reviewsDistribution": {              // ← BARU dari aria-label histogram
    "oneStar": 32, "twoStar": 5, "threeStar": 5, "fourStar": 13, "fiveStar": 135
  },
  "website": "...", "urlGoogleMaps": "...", "googlePlaceId": "...",
  "status": "active", "claimed": false,
  "wifiAvailable": true,                // ← BARU scan about → "wi-fi"/"wifi"
  "hasParking": true,                   // ← BARU scan about → "parkir"/"parking"
  "payment": {                          // ← BARU dari section "Pembayaran"
    "cash": false, "debitCard": true, "creditCard": true,
    "qris": true, "nfc": false, "ewallet": true
  },
  "pricing": "Rp 25.000-50.000",       // ← BARU dari price_level
  "gallery": ["url1", "url2", ...],    // ← BARU full photos array
  "menu": {                             // ← BARU dari tab "Menu"
    "items": [{"name": "Espresso", "price": "Rp 30.000"}, ...],
    "photos": ["url", ...]
  },
  "city": "Bandung",                    // ← BARU hardcoded
  "district": "Bandung Wetan",          // ← BARU dari kecamatan seed
  "reviews": [...]                      // dipindah ke akhir (heavy array)
}
```

**Backward compat:** schema barbershop tetap 16 field — tidak ada field cafe-specific yang nyangkut. Frontend lama yang konsumsi schema barbershop tetap jalan tanpa perubahan.

**Migration old data:** kalau ada file JSON lama dengan format `{kelurahan, barbershops}` (pre-refactor), pakai `python scripts/migrate.py --keyword cafe` untuk transform ke SyncItem array in-place.

---

## Configuration

### `.env.local` (runtime injection)

Dibuat di root project + di VPS `/opt/scrapper-google-maps/`. Compose `env_file:` directive injectkan ke container env saat startup.

```bash
APP_URL=https://api.example.com/api
GOOGLE_MAPS_SYNC_API_KEY=your_key_here

# Optional override (default sudah baik)
HEADLESS=true
MIN_DELAY_SEC=10
MAX_DELAY_SEC=25
MAX_REVIEWS_PER_SHOP=200
MAX_REVIEW_AGE_DAYS=730
LOG_LEVEL=INFO
```

### Tuning parameter di [config.py](config.py)

| Parameter | Default | Note |
|---|---|---|
| `MIN_DELAY_SEC` / `MAX_DELAY_SEC` | 10–25 | Random delay antar request (anti-ban) |
| `MAX_REVIEWS_PER_SHOP` | 200 | Batas review per place |
| `MAX_REVIEW_AGE_DAYS` | 730 | Skip review > N hari (≈2 tahun) |
| `MAX_CAPTCHA_RETRY` | 2 | Stop kalau CAPTCHA berturut-turut |
| `SKIP_EMPTY_REVIEWS` | true | Skip review kosong (hanya rating) |
| `SORT_REVIEWS_BY_NEWEST` | true | Klik sort newest sebelum scrape reviews |
| `HEADLESS` | true | `false` saat debug supaya browser visible |

Override via env var (di `.env.local` atau saat invoke).

### Tuning di Docker (compose dev)

Compose dev override delay lebih agresif (5–12 detik vs prod 10–25) untuk dev cepat:

```yaml
build.args:
  MIN_DELAY_SEC: 5
  MAX_DELAY_SEC: 12
```

---

## Anti-Ban Safety

Project ini dirancang **TIDAK PERNAH** menyentuh akun Google:

| Aturan | Implementasi |
|---|---|
| ❌ JANGAN login ke Google saat scraping | Browser context fresh tiap session, no persistent profile |
| ❌ JANGAN pakai profile Chrome utama | Playwright isolated context |
| ❌ JANGAN simpan cookie session | `storage_state` tidak di-save |
| ✅ Random delay agresif | 5–25 detik random antar request |
| ✅ Stealth mode | playwright-stealth + UA rotation |
| ✅ Detect CAPTCHA | Auto-pause + log alert kalau Google curiga |
| ✅ Fingerprint randomization | Viewport + locale random per session |
| ✅ Auto-stop pada anomali | CAPTCHA streak ≥2 atau network error ≥5 → stop |

**Risiko ban akun = NOL** (tidak ada akun dipakai). Yang berisiko cuma **IP block sementara** (recover sendiri 6–12 jam, ganti IP/restart router).

---

## CI/CD

**Trigger:** push ke `main` atau tag `v*`, atau manual `workflow_dispatch`.

**Pipeline:**
1. **build-and-push** — Buildx build → push ke GHCR dengan tag `latest`, `sha-<short>`, `v<tag>`. Image **tidak** mengandung secret (env_file injection di runtime).
2. **deploy-vps** (only main/tags) — SSH ke VPS, tulis `.env.local` fresh dari GitHub Secrets, `docker compose pull && up -d`, prune image >7 hari.

**GitHub Secrets yang harus di-set:**
- `APP_URL`, `GOOGLE_MAPS_SYNC_API_KEY` — credential backend
- `VPS_HOST_IP`, `VPS_SSH_USER`, `VPS_SSH_KEY`, `VPS_SSH_PORT` (opt)

Workflow file: [.github/workflows/build-and-deploy.yml](.github/workflows/build-and-deploy.yml)

---

## Estimasi Runtime

- **151 kelurahan** × rata-rata **8–15 place** × **20–30 detik** per place (termasuk reviews + menu)
- **Total: ±6–12 jam** per keyword (jalankan overnight di VPS)

Aktifkan `--auto-sync` supaya hasil langsung POST ke API tanpa langkah manual.

---

## Troubleshooting

Lihat [docs/scr.md#troubleshooting](docs/scr.md#troubleshooting) untuk diagnosis 7 error paling umum (container state improper, version drift Playwright, API key kosong, dll).

Quick reference:

| Masalah | Solusi |
|---|---|
| Container exit langsung | Cek `docker/entrypoint.sh` line ending — wajib LF, bukan CRLF |
| `Akan scrape 0 kelurahan` | `--resume` skip kelurahan done — pakai `--kelurahan X` (auto-drop resume) |
| `Sync GAGAL: API_KEY kosong` | Set `.env.local` + `docker compose up -d` (no rebuild) |
| Playwright version mismatch | Pin `Dockerfile FROM v1.X.0-jammy` ↔ `requirements.txt playwright==1.X.0` lockstep |
| First build/pull lama | Base image ~1.7 GB — first time 5–15 menit, after that cached |

---

## Catatan Legal

Project untuk **riset / penggunaan personal**. Scraping Google Maps melanggar Google ToS. Untuk komersial, pakai [Google Places API](https://developers.google.com/maps/documentation/places/web-service) yang resmi.

## Lisensi

MIT
