# `scr` — Scrapper Helper Script

Wrapper CLI untuk operasi scraper di container (Docker/Podman). Satu interface untuk **local development** dan **VPS production**, dengan auto-detect compose tool dan auto-detect resume mode.

```bash
./scr scrape cafe --kelurahan "Cihapit" --limit 1   # local re-scrape
./scr --vps scrape cafe                             # VPS full run
```

Tersedia dua versi paralel:
- **`scr`** — bash, untuk Linux / Mac / WSL / Git Bash
- **`scr.ps1`** — PowerShell, untuk Windows native

Di PowerShell, `.\scr` auto-resolve ke `.\scr.ps1` lewat `PATHEXT`.

---

## Daftar Isi

- [Mode](#mode)
- [Ringkasan Action](#ringkasan-action)
- [Action Detail](#action-detail)
- [Pre-requisites](#pre-requisites)
- [Auto-detection](#auto-detection)
- [Override Manual](#override-manual)
- [Troubleshooting](#troubleshooting)
- [FAQ](#faq)

---

## Mode

`scr` punya dua mode:

| Mode | Trigger | Target | Use case |
|---|---|---|---|
| **LOCAL** (default) | tanpa flag | Container `scrapper-dev` di laptop, lewat docker/podman compose | Dev, test, local scrape |
| **VPS** | `--vps` (bash) / `-Vps` (PS) | Container `scrapper-prod` di VPS, lewat SSH | Production scrape long-running |

LOCAL mode auto-detect compose tool: prefer `docker compose`, fallback ke `podman-compose`.

VPS mode butuh `VPS_HOST` env var di-set:
```bash
export VPS_HOST=user@host                # bash
$env:VPS_HOST = "user@host"              # PowerShell
```

---

## Ringkasan Action

| Action | Butuh `<keyword>` | Keterangan |
|---|---|---|
| `scrape <keyword> [args]` | ✅ | Start scrape di **background (detached)** — close terminal aman |
| `sync <keyword> [args]` | ✅ | POST file JSON ke API backend (foreground) |
| `status <keyword>` | ✅ | Progress count + sync summary untuk keyword tertentu |
| `pull <keyword>` | ✅ | (VPS only) Download `data/<keyword>/` dari VPS ke laptop |
| `stop` | ❌ | Kill scrape process di container |
| `is-running` | ❌ | Cek apakah scrape sedang jalan |
| `logs` | ❌ | Tail log file Python real-time (Ctrl+C exit) |
| `shell` | ❌ | Bash interactive di container (debug) |
| `ps` | ❌ | Container status |
| `redeploy` | ❌ | LOCAL: rebuild image. VPS: pull image baru + restart |

Action yang **butuh keyword** akan exit dengan error kalau keyword tidak di-pass.

---

## Action Detail

### `scrape <keyword> [args]`

Start scrape detached di background. Kembali ke shell langsung — process tetap jalan walau terminal ditutup.

```bash
./scr scrape cafe                                    # full run, semua kelurahan
./scr scrape cafe --kelurahan "Cihapit"              # 1 kelurahan saja
./scr scrape cafe --kelurahan "Cihapit" --limit 5    # batasi 5 place per kelurahan
./scr scrape cafe --auto-sync                        # full run + auto-sync ke API
./scr scrape barbershop --resume                     # resume manual (default off untuk single keluharan)
./scr --vps scrape cafe --auto-sync                  # production VPS
```

**Auto-detect behavior:**

| Kondisi | `--resume` | Use case |
|---|---|---|
| Tanpa `--kelurahan` | ✅ on | Full run — skip kelurahan yang sudah `done` di SQLite |
| Dengan `--kelurahan X` | ❌ off | Re-scrape spesifik — bypass progress |

**Auto-sync** **TIDAK** auto-pass — opt-in via `--auto-sync` flag (butuh `GOOGLE_MAPS_SYNC_API_KEY` di `.env.local`).

Output file: `./data/<keyword>/<kelurahan>.json`

---

### `sync <keyword> [args]`

Manual sync file JSON yang sudah di-scrape ke API backend. Foreground (block sampai selesai).

```bash
./scr sync cafe --kelurahan "Cihapit"        # 1 file kelurahan
./scr sync cafe --all                         # semua file di data/cafe/, skip yang sudah ter-sync
./scr sync cafe --all --force                 # force re-sync semua (ignore SQLite tracking)
./scr sync cafe --all --dry-run               # preview, no POST
./scr --vps sync cafe --all                   # sync di VPS
```

`--all`, `--kelurahan X`, atau `--file path/x.json` — wajib salah satu (mutually exclusive).

---

### `status <keyword>`

Snapshot progress + sync untuk keyword tertentu.

```bash
./scr status cafe
# Output:
#   keyword: cafe
#   scrape: {'done': 12, 'failed': 1}
#   sync: {'done': {'count': 12, 'inserted': 134}}
```

---

### `pull <keyword>` (VPS only)

Download `data/<keyword>/` dari VPS ke `./data/<keyword>-from-vps/`.

```bash
./scr --vps pull cafe
```

LOCAL mode: no-op (data sudah volume-mounted di `./data/<keyword>/`).

---

### `stop` / `is-running` / `logs`

Tidak butuh keyword — beroperasi pada container yang aktif.

```bash
./scr stop          # kill scrape process
./scr is-running    # [RUNNING] / [STOPPED]
./scr logs          # tail -f log file Python (Ctrl+C exit)
./scr --vps logs    # tail log VPS via SSH
```

---

### `shell`

Bash interactive di dalam container — handy untuk debug.

```bash
./scr shell

# Inside container:
$ ls /app/data/cafe/
$ python -c "from src.storage import progress_summary; print(progress_summary())"
```

---

### `ps` / `redeploy`

```bash
./scr ps              # container status (LOCAL: dev compose, VPS: prod compose)
./scr redeploy        # LOCAL: rebuild image dari Dockerfile
                      # VPS: pull image baru dari GHCR + restart
```

---

## Pre-requisites

### LOCAL mode

```bash
# 1. Docker atau Podman terinstall + compose plugin/binary
docker compose version           # OR: podman-compose --version

# 2. Build dan start container dev (sekali)
docker compose -f docker-compose.dev.yml up -d --build
# OR:
podman-compose -f docker-compose.dev.yml up -d --build

# 3. (Opsional) .env.local untuk API sync
cat > .env.local <<EOF
APP_URL=https://api.example.com/api
GOOGLE_MAPS_SYNC_API_KEY=your_key_here
EOF
```

### VPS mode

```bash
# 1. SSH key sudah di-setup ke VPS
ssh user@host         # passwordless via key

# 2. Set VPS_HOST env var
export VPS_HOST=user@host                 # bash, taruh di ~/.bashrc supaya persistent
$env:VPS_HOST = "user@host"               # PowerShell, taruh di $PROFILE

# 3. (Opsional) custom path di VPS
export VPS_PROJ_DIR=/opt/scrapper-google-maps    # default ini

# 4. Container scrapper-prod di VPS sudah jalan (lewat docker-compose.prod.yml)
```

---

## Auto-detection

`scr` melakukan beberapa detection otomatis untuk minimize friction:

### Compose tool

```
prefer 'docker compose' → fallback 'podman-compose' → error
```

Detection pakai `docker compose version` (success?) lalu `command -v podman-compose`. Hanya jalan di LOCAL mode (VPS mode pakai docker remote).

### Resume mode

```
ada --kelurahan di args? → --resume off
                  else  → --resume on (full run)
```

### Container readiness

Sebelum `exec`, helper tidak cek explicit. Kalau container down, exec error → helper print error message dengan saran cek `./scr ps`.

---

## Override Manual

Auto-detect kadang salah pilih atau Anda butuh perilaku berbeda. Override via env var:

### `COMPOSE_TOOL` — paksa compose tool

```bash
COMPOSE_TOOL=docker  ./scr ps        # paksa docker walau podman juga ada
COMPOSE_TOOL=podman  ./scr ps        # paksa podman walau docker juga ada
```

PowerShell:
```powershell
$env:COMPOSE_TOOL = "podman"
.\scr ps
Remove-Item Env:COMPOSE_TOOL         # balik ke auto-detect
```

### `VPS_HOST` / `VPS_PROJ_DIR` — VPS connection

```bash
export VPS_HOST=dios@182.23.12.142
export VPS_PROJ_DIR=/opt/scrapper-google-maps    # default
```

### `--resume` manual

Auto-detect drop `--resume` saat ada `--kelurahan`. Kalau Anda mau **paksa resume** walau pakai filter:

```bash
./scr scrape cafe --kelurahan "Cihapit" --resume    # explicit, akan tetap auto-detect drop
                                                     # ⚠ behavior: helper masih akan strip --resume karena --kelurahan present
```

⚠ Limitation: helper strip `--resume` based on `--kelurahan` presence — kalau Anda butuh resume + filter sekaligus, panggil `python scripts/scraper.py` langsung di container:

```bash
./scr shell
# Inside:
python scripts/scraper.py --keyword cafe --kelurahan "Cihapit" --resume
```

---

## Troubleshooting

### `Error: can only create exec sessions on running containers: container state improper`

Container ada tapi tidak running. Cek state:

```bash
./scr ps                         # status: harus "Up X seconds"
podman logs scrapper-dev         # cek pesan exit
```

Common cause: `docker/entrypoint.sh` line ending CRLF (Windows). Fix: pastikan file LF, rebuild:

```bash
sed -i 's/\r$//' docker/entrypoint.sh    # convert ke LF
podman-compose -f docker-compose.dev.yml down
podman-compose -f docker-compose.dev.yml up -d --build --no-cache
```

### `Akan scrape 0 kelurahan` saat pakai `--kelurahan`

**Sudah di-fix** sejak v2 (auto-drop `--resume` saat `--kelurahan` present). Kalau masih kena, cek versi `scr` Anda — atau manual delete progress entry:

```bash
./scr shell
# Inside:
python -c "import sqlite3; c=sqlite3.connect('/app/data/cafe/progress.db'); c.execute(\"DELETE FROM kelurahan_progress WHERE kelurahan='Cihapit'\"); c.commit()"
```

### `Sync ... GAGAL: GOOGLE_MAPS_SYNC_API_KEY kosong`

API key belum di-set. Add ke `.env.local`:

```
GOOGLE_MAPS_SYNC_API_KEY=your_key_here
APP_URL=https://api.example.com/api
```

Lalu `./scr redeploy` (rebuild kalau dev compose passes via build args) atau restart container untuk pickup env baru.

Atau **skip auto-sync** — scrape only:
```bash
./scr scrape cafe                         # tanpa --auto-sync, scrape only
./scr sync cafe --all                     # sync manual nanti setelah API ready
```

### `'docker compose' atau 'podman-compose' tidak ditemukan di PATH`

Tidak ada compose tool. Install salah satu:
- **Docker Desktop** (compose plugin built-in) — [docs.docker.com](https://docs.docker.com)
- **Podman + podman-compose** — `pip install podman-compose`

Kalau tool ada tapi tidak ke-detect (misal binary di non-standard path), set `COMPOSE_TOOL` manual.

### `Set VPS_HOST=user@host dulu`

VPS mode butuh `VPS_HOST` env. Set dulu sebelum `--vps` action:

```bash
export VPS_HOST=user@host
./scr --vps status cafe
```

### Container response lama / lag

Pertama kali jalan setelah build: image pull + pip install bisa makan 5-15 menit (base Playwright image ~1.7GB). Setelahnya container exec <1 detik.

Kalau masih lambat: cek `podman info --format '{{.Store.GraphDriverName}}'` — kalau `vfs` (slow) bukan `overlay`, tweak `~/.config/containers/storage.conf`.

### Ganti versi Playwright

Pin lockstep di **dua tempat** (kalau drift, container exit dengan binary mismatch):

```dockerfile
# Dockerfile
FROM mcr.microsoft.com/playwright/python:v1.59.0-jammy
```

```
# requirements.txt
playwright==1.59.0
```

Lalu `./scr redeploy` (LOCAL) atau update di repo + `./scr --vps redeploy` (VPS).

---

## FAQ

### Q: Kenapa `--auto-sync` tidak default-on?

Auto-sync butuh API endpoint live + `GOOGLE_MAPS_SYNC_API_KEY` set. Kalau API belum ready (mis. lagi develop new keyword schema), default-on bikin tiap scrape kena error log noise. **Opt-in lebih aman** — Anda kontrol kapan POST ke API.

### Q: Kenapa `--resume` auto-drop saat ada `--kelurahan`?

Pattern dev paling sering: re-scrape kelurahan tertentu untuk testing schema baru. Kalau `--resume` on, scraper skip kelurahan yang sudah marked `done` → 0 kelurahan ke-scrape → frustration loop.

Untuk full-run production, kalau interrupted, tinggal `./scr scrape cafe` (tanpa `--kelurahan`) — `--resume` auto-on, lanjut dari progress terakhir.

### Q: LOCAL mode pakai `docker-compose.dev.yml`. VPS mode pakai compose mana?

VPS mode pakai SSH `docker exec scrapper-prod ...` langsung — **tidak** pakai compose file. Container `scrapper-prod` di VPS managed lewat `docker-compose.prod.yml` yang dijalankan **di VPS itu sendiri** saat deploy.

`scr --vps` cuma operate pada container yang sudah running.

### Q: Bagaimana cara stop scrape di tengah jalan tanpa loss progress?

`./scr stop` kill process gracefully. SQLite progress (`progress.db`) sudah di-update incrementally setiap kelurahan done — tidak ada loss. Resume dengan `./scr scrape <keyword>` (tanpa filter) → `--resume` auto-on, lanjut dari kelurahan terakhir.

### Q: Bisa scrape 2 keyword pararel di local?

**Tidak** — container `scrapper-dev` cuma 1, exec pararel akan saling rebut browser. Buat dua container terpisah kalau benar-benar butuh:

```yaml
# docker-compose.dev.yml
services:
  scraper-cafe:
    extends: scraper
    container_name: scrapper-cafe
  scraper-barber:
    extends: scraper
    container_name: scrapper-barber
```

Tapi simpler: jalan satu keyword sampai selesai, baru pindah. Per-keyword storage isolated (`data/cafe/`, `data/barbershop/`) jadi tidak konflik.

### Q: Helper script ini bisa dipakai dari directory selain project root?

Tidak. Path ke `docker-compose.dev.yml` dan `data/` relatif ke CWD. Selalu invoke dari project root:

```bash
cd /path/to/scrapper
./scr scrape cafe
```

### Q: Cara bikin `scr` callable dari mana saja (tidak perlu `./` prefix)?

Tambah project root ke PATH, atau bikin alias di `$PROFILE` (PowerShell) / `~/.bashrc` (bash):

```powershell
# PowerShell $PROFILE
function scr { & "C:\Users\Hype G12\Desktop\project\scrapper\scr.ps1" @args }
```

```bash
# ~/.bashrc
alias scr='/path/to/scrapper/scr'
```

Setelah `source $PROFILE` atau restart shell:
```
scr scrape cafe                       # tanpa ./ atau bin/
```

⚠ Tetap perlu `cd` ke project root karena CWD-relative paths inside script.

---

## Lihat juga

- [README.md](../README.md) — overview project & schema scraping
- [docker-compose.dev.yml](../docker-compose.dev.yml) — definisi container LOCAL
- [docker-compose.prod.yml](../docker-compose.prod.yml) — definisi container VPS
- [scripts/scraper.py](../scripts/scraper.py) — entry point Python yang di-wrap
- [scripts/sync.py](../scripts/sync.py) — sync CLI yang di-wrap
