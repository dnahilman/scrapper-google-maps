# Server Setup Guide — Multi-VPS Parallel Scraping

Dokumen ini step-by-step setup Server A & B untuk scrape parallel:
- **Server A** (`diosone@100.79.26.53`, Ubuntu + Docker): 2 instance — cafe `1/2` + barbershop `1/3`
- **Server B** (`smoly@100.115.134.33` aka MiniServer, Bazzite + Podman): 3 instance — cafe `2/2` + barbershop `2/3` + barbershop `3/3`

Asumsi: image `0.1.4` sudah di GHCR, baseline `progress.db` (cleaned, 76 cafe + 7 barbershop done) sudah ada di local.

## Prerequisite (sudah ada di repo / GHCR — tidak perlu di-setup ulang)

- Image `ghcr.io/dnahilman/scrapper-google-maps:0.1.4` (public, tidak butuh auth pull)
- File `docker-compose.yml` dan `scrape` di branch `dev` GitHub
- Local clean baseline:
  - `data/cafe/progress.db` — 76 done
  - `data/barbershop/progress.db` — 7 done

## Server A — setup (sudah selesai dilakukan, tinggal continue/restart)

### Status saat ini
Server A (`diosone@100.79.26.53`) sudah jalan dengan:
- Container `scrapper` (image v0.1.4) running
- 2 shard aktif: cafe `1/2` + barbershop `1/3`
- Folder `/opt/scrapper/data/{cafe,barbershop}/progress.db` baseline 76 + 7 done

### Re-deploy (kalau perlu mulai ulang dari nol)

```bash
ssh diosone@100.79.26.53

# 1. Stop scraper + remove container lama
docker exec scrapper pkill -f "server/scripts/scraper.py" 2>/dev/null || true
docker stop scrapper 2>/dev/null
docker rm scrapper 2>/dev/null

# 2. Setup folder + permission
cd /opt/scrapper
sudo chown -R diosone:diosone /opt/scrapper/data
mkdir -p data/cafe data/barbershop logs

# 3. Fetch latest docker-compose.yml + scrape + seed dari GitHub (branch dev)
curl -fsSL -o docker-compose.yml https://raw.githubusercontent.com/dnahilman/scrapper-google-maps/refs/heads/dev/docker-compose.yml
curl -fsSL -o scrape https://raw.githubusercontent.com/dnahilman/scrapper-google-maps/refs/heads/dev/scrape
curl -fsSL -o data/kelurahan_bandung.json https://raw.githubusercontent.com/dnahilman/scrapper-google-maps/refs/heads/dev/data/kelurahan_bandung.json
chmod +x scrape

# 4. Pull image + start container (.env.local opsional, gak butuh untuk scrape-only)
docker compose pull
docker compose up -d
docker compose ps
```

### Sync `progress.db` dari local (kalau belum ada / corrupt)

Dari **PowerShell di local**:
```powershell
$VPS_A = "100.79.26.53"

# Verify ownership di VPS A dulu (run di SSH):
# sudo chown -R diosone:diosone /opt/scrapper/data

scp "C:\Users\Hype G12\Desktop\project\scrapper\data\cafe\progress.db" "diosone@${VPS_A}:/opt/scrapper/data/cafe/progress.db"
scp "C:\Users\Hype G12\Desktop\project\scrapper\data\barbershop\progress.db" "diosone@${VPS_A}:/opt/scrapper/data/barbershop/progress.db"
```

### Verify state

Di Server A:
```bash
./scrape progress cafe        # expect: ('done', 76)
./scrape progress barbershop  # expect: ('done', 7)

# Verify env defaults dari compose
docker exec scrapper python -c "
import os
print('MIN_DELAY:', os.getenv('MIN_DELAY_SEC'))   # 15
print('MAX_DELAY:', os.getenv('MAX_DELAY_SEC'))   # 30
print('API_KEY :', 'SET' if os.getenv('GOOGLE_MAPS_SYNC_API_KEY') else '(empty - scrape OK)')
"
```

### Spawn 2 shard

```bash
docker exec -d scrapper python server/scripts/scraper.py --keyword cafe --resume --shard 1/2
docker exec -d scrapper python server/scripts/scraper.py --keyword barbershop --resume --shard 1/3

# Verify
./scrape ps
# Harus tampil 2 baris scraper.py: cafe shard 1/2 + barbershop shard 1/3
```

---

## Server B — setup (MiniServer / Bazzite + Podman)

Server B (`smoly@100.115.134.33` di tailnet shared dari `frmn.play@`) pakai **Bazzite** (Fedora Atomic, gaming OS) + **podman-compose** (bukan docker). Quirk yang perlu diingat:

- **Tailscale SSH meng-hijack port 22 via tailnet IP** → password auth ditolak. Akses harus via:
  - Cockpit web terminal: https://100.115.134.33:9090 (login `smoly` + password)
  - LAN SSH ke `192.168.1.99` (kalau lo di subnet sama)
  - Setelah ACL Tailscale di firumanusia di-update, baru bisa `tailscale ssh smoly@100.115.134.33`
- **SELinux Enforcing** → bind mount Podman butuh suffix `:z` di compose, plus `chcon -Rt container_file_t` ke host dirs. Tanpa ini, SQLite WAL error `attempt to write a readonly database`.

### 1. Pre-flight di local — pastikan baseline `progress.db` belum ke-overwrite

```powershell
# Cek ukuran file (cleaned baseline ~28KB cafe + ~20KB barbershop)
ls "C:\Users\Hype G12\Desktop\project\scrapper\data\cafe\progress.db"
ls "C:\Users\Hype G12\Desktop\project\scrapper\data\barbershop\progress.db"

# Cek isi pakai Python:
python -c "
import sqlite3
for kw in ['cafe', 'barbershop']:
    conn = sqlite3.connect(f'data/{kw}/progress.db')
    print(kw, dict(conn.execute('SELECT status, COUNT(*) FROM kelurahan_progress GROUP BY status').fetchall()))
    conn.close()
"
# Expected: cafe {'done': 76}, barbershop {'done': 7}
```

Kalau hasil **bukan** 76 + 7 → restore dari snapshot:
```powershell
copy "backups\snapshots\cafe-progress-original.db" "data\cafe\progress.db"
copy "backups\snapshots\barbershop-progress-original.db" "data\barbershop\progress.db"
# Lalu re-run cleanup script (lihat history percakapan)
```

### 2. Akses Server B + setup folder

Buka Cockpit di https://100.115.134.33:9090 → login `smoly` → Terminal. Lalu:

```bash
# Setup direktori (smoly perlu password sudo)
sudo mkdir -p /opt/scrapper
sudo chown -R smoly:smoly /opt/scrapper
cd /opt/scrapper
mkdir -p data/cafe data/barbershop logs
```

### 3. Fetch `docker-compose.yml` + `scrape` + seed kelurahan dari GitHub (branch dev)

```bash
cd /opt/scrapper

# docker-compose.yml (image v0.1.4, container_name=scrapper, env defaults built-in)
curl -fsSL -o docker-compose.yml https://raw.githubusercontent.com/dnahilman/scrapper-google-maps/refs/heads/dev/docker-compose.yml

# scrape helper CLI
curl -fsSL -o scrape https://raw.githubusercontent.com/dnahilman/scrapper-google-maps/refs/heads/dev/scrape
chmod +x scrape

# Seed kelurahan (151 kelurahan Bandung) — WAJIB ada di host, image gak bawa
curl -fsSL -o data/kelurahan_bandung.json https://raw.githubusercontent.com/dnahilman/scrapper-google-maps/refs/heads/dev/data/kelurahan_bandung.json

# Verify
head -20 docker-compose.yml                                          # cek image tag = 0.1.4
./scrape help                                                        # cek scrape helper jalan
python3 -c "import json; print(len(json.load(open('data/kelurahan_bandung.json'))))"  # expect: 151
```

### 4. Copy `progress.db` baseline dari local

Karena Tailscale SSH block password (`tailnet policy does not permit you to SSH to this node`), `scp` standar tidak jalan. 2 opsi:

**A. Via Cockpit Files** (paling simple): buka https://100.115.134.33:9090 → File → navigasi ke `/opt/scrapper/data/cafe/` → Upload `progress.db` dari local. Ulangi untuk barbershop.

**B. Via gist (kalau perlu otomatis dari local)**: base64-encode → upload ke secret gist via `gh gist create` → di MiniServer: `curl -fsSL <raw> | base64 -d > data/<keyword>/progress.db`. Hapus gist setelah selesai (`gh gist delete <id> --yes`).

### 5. Patch compose untuk SELinux + pull image + start container di Server B

```bash
cd /opt/scrapper

# WAJIB di Bazzite: tambah ':z' ke bind mount + relabel host dirs
sed -i -E 's|(- \./data:/app/data)$|\1:z|' docker-compose.yml
sed -i -E 's|(- \./logs:/app/logs)$|\1:z|' docker-compose.yml
sudo chcon -Rt container_file_t data logs

docker compose pull          # podman-compose pull v0.1.4 dari GHCR
docker compose up -d         # start container "scrapper"
docker compose ps            # verify running
```

### 6. Verify state baseline + env

```bash
./scrape progress cafe        # expect: ('done', 76)
./scrape progress barbershop  # expect: ('done', 7)

docker exec scrapper python -c "
import os
print('MIN_DELAY:', os.getenv('MIN_DELAY_SEC'))   # 15
print('MAX_DELAY:', os.getenv('MAX_DELAY_SEC'))   # 30
print('API_KEY :', 'SET' if os.getenv('GOOGLE_MAPS_SYNC_API_KEY') else '(empty - scrape OK)')
"
```

Kalau output `(done, 76)` & `(done, 7)` → baseline benar, lanjut spawn shard.
Kalau output kosong / 0 → progress.db belum ke-copy / file kosong, ulangi step 4.

### 7. Spawn 3 shard

```bash
docker exec -d scrapper python server/scripts/scraper.py --keyword cafe --resume --shard 2/2
docker exec -d scrapper python server/scripts/scraper.py --keyword barbershop --resume --shard 2/3
docker exec -d scrapper python server/scripts/scraper.py --keyword barbershop --resume --shard 3/3

# Verify
./scrape ps
# Harus tampil 3 baris scraper.py
```

### 8. Monitor progress

```bash
# One-shot status check
./scrape ps
./scrape progress cafe
./scrape progress barbershop

# Auto-refresh tiap 10 menit (Ctrl+C untuk stop)
watch -n 600 "./scrape ps; echo; ./scrape progress cafe; echo; ./scrape progress barbershop"
```

---

## Troubleshooting

### Server B / Bazzite — `sqlite3.OperationalError: attempt to write a readonly database`

SELinux + podman bind mount tanpa `:z` label. Fix:
```bash
cd /opt/scrapper
sed -i -E 's|(- \./data:/app/data)$|\1:z|' docker-compose.yml
sed -i -E 's|(- \./logs:/app/logs)$|\1:z|' docker-compose.yml
sudo chcon -Rt container_file_t data logs
docker compose down && docker compose up -d
```
Verify: `docker exec scrapper bash -c "echo t > /app/data/_w.txt && rm /app/data/_w.txt && echo OK"`

### Server B — `tailscale: tailnet policy does not permit you to SSH to this node`

Tailscale SSH override port 22 dan ACL belum izinkan user lo. Workaround:
- Cockpit web terminal di https://<tailscale-ip>:9090 (selama Cockpit reachable)
- Atau minta admin tailnet (firumanusia) update ACL untuk allow `dnahilman@github` SSH ke node sebagai `smoly`

### `./scrape progress` output kosong

**Kemungkinan penyebab:**
1. `progress.db` belum ke-copy / kosong → cek `ls -la /opt/scrapper/data/cafe/progress.db`. Size harus ~28KB (cafe) atau ~20KB (barbershop). Kalau 0 / 4KB → re-scp.
2. `progress.db` di-mount sebagai owned `root` (dari container init) → `sudo chown -R dios:dios /opt/scrapper/data`, lalu re-scp.
3. Path mismatch di container → cek `docker exec scrapper python -c "import config; config.set_keyword('cafe'); print(config.progress_db())"`. Harus print `/app/data/cafe/progress.db`.

### Permission denied saat scp

```bash
# Di VPS:
sudo chown -R <USER>:<USER> /opt/scrapper/data
sudo chmod -R u+rw /opt/scrapper/data
```

### `docker compose` versi lama (gak support `env_file: required: false`)

Cek versi:
```bash
docker compose version
# Butuh v2.24+
```

Update Docker / Docker Compose plugin:
```bash
sudo apt install --only-upgrade docker.io docker-compose-plugin
```

Atau workaround: bikin file `.env.local` kosong:
```bash
touch /opt/scrapper/.env.local
```
Compose lama gak akan fail kalau file ada walau kosong.

### Scraper di-`ps` ada tapi `progress` count gak nambah

Scrape per kelurahan rata-rata 30-90 menit (cafe heavy). Kalau dalam 2 jam masih sama:
- Cek log: `docker exec scrapper bash -c "tail -100 /app/logs/scraper-*.log"`
- Cek CAPTCHA: `docker exec scrapper bash -c "grep -i captcha /app/logs/scraper-*.log | tail -5"`
- Kalau kena CAPTCHA berturut-turut, scraper auto-stop. Restart container + re-spawn shard.

### CAPTCHA recovery

```bash
# 1. Kill scraper
./scrape stop

# 2. Tunggu 6-12 jam (Google reset rate limit)

# 3. Restart container untuk fresh fingerprint
cd /opt/scrapper
docker compose restart

# 4. Re-spawn shard yang stop (--resume otomatis skip done)
docker exec -d scrapper python server/scripts/scraper.py --keyword cafe --resume --shard 2/2
# atau shard berapa pun yang stop
```

---

## Setelah scrape selesai (Server A + Server B)

### 1. Pull semua JSON ke central machine (local)

Dari PowerShell:
```powershell
$VPS_A = "100.79.26.53"

# Pull dari Server A — file dari shard 1/2 cafe + 1/3 barbershop
rsync -avz "diosone@${VPS_A}:/opt/scrapper/data/cafe/"       "data/cafe/"
rsync -avz "diosone@${VPS_A}:/opt/scrapper/data/barbershop/" "data/barbershop/"

# Pull dari Server B (MiniServer) — Tailscale SSH block scp/rsync via tailnet IP.
# Pakai Cockpit Files (https://100.115.134.33:9090) untuk download per-folder ZIP,
# atau ssh ke LAN IP 192.168.1.99 dari subnet sama, atau update ACL Tailscale dulu.
```

### 2. Verify count

```powershell
(Get-ChildItem "data\cafe\*.json").Count        # expect: 151
(Get-ChildItem "data\barbershop\*.json").Count  # expect: 151
```

### 3. Sync ke API (saat siap)

Set API key + sync URLs di `.env.local` local, lalu:
```powershell
python server/scripts/sync.py --keyword cafe --all
python server/scripts/sync.py --keyword barbershop --all
```

`sync.py` skip file yang sudah `done` di `sync_progress` table — aman dijalankan ulang.

---

## Reference: image versions

| Tag | Description |
|---|---|
| `0.1.4` | Latest stable: shard-first order fix + env defaults |
| `0.1.3` | ⚠ Has shard order bug, jangan dipakai untuk multi-VPS |
| `0.1.2` | Cafe-aware sync dispatch + scrape helper |
| `0.1.1` | Initial JSON sync format |

URL repo image: https://github.com/dnahilman/scrapper-google-maps/pkgs/container/scrapper-google-maps

---

## Reference: shard mapping

| Server | Instance | Shard | Partition | After --resume |
|---|---|---|---|---|
| A (`diosone`)  | 1 | cafe `1/2`       | 76 kel | 40 kel |
| A (`diosone`)  | 2 | barbershop `1/3` | 51 kel | 48 kel |
| B (`miniserver`) | 1 | cafe `2/2`       | 75 kel | 35 kel |
| B (`miniserver`) | 2 | barbershop `2/3` | 50 kel | 48 kel |
| B (`miniserver`) | 3 | barbershop `3/3` | 50 kel | 48 kel |

**Total work:** 75 cafe + 144 barbershop = 219 kelurahan, terbagi disjoint antar 5 instance. Round-robin partition pakai posisi absolut di list 151, jadi gak peduli kapan tiap server start, partisi tetap fix.
