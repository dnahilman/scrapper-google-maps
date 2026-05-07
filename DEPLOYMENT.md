# Multi-VPS Deployment Plan — Cafe + Barbershop

Dokumen ini cover continue scrape **cafe** dan barbershop di **2 VPS** dengan total **5 instance paralel**, setelah cleanup state lama.

## State setelah cleanup (sudah dijalankan di local)

Cleanup: hapus rows `in_progress` dan `failed` dari `kelurahan_progress` + `sync_progress` di kedua `progress.db` lokal. Done dipertahankan supaya `--resume` tetap bekerja.

| Keyword | Sebelum cleanup | Setelah cleanup | Remaining (perlu scrape) |
|---|---|---|---|
| **cafe** | 76 done + 11 failed + 2 in_progress | **76 done** | **75 kelurahan** |
| **barbershop** | 7 done + 1 in_progress | **7 done** | **144 kelurahan** |

Backup snapshot original (sebelum cleanup) ada di:
- `backups/snapshots/cafe-progress-original.db`
- `backups/snapshots/barbershop-progress-original.db`

JSON cafe lengkap (75 file) dari VPS A: `backups/cafe/`.

## Pembagian shard

**Algoritma:** `items[(K-1)::N]` dihitung dari **list lengkap 151 kelurahan** (posisi absolut), **lalu** `--resume` hapus item yang status `done`. Order ini penting:

1. Shard partition fix antar VPS — tidak peduli progress.db state.
2. Server A & B yang start di waktu berbeda akan compute partition yang sama.
3. Re-run setelah crash juga aman — partition tidak shift.

### Cafe — 2 shard (partition 76 + 75 dari 151, lalu resume hapus done)

| Shard | Partition | After resume | First → Last | Assigned to |
|---|---|---|---|---|
| `1/2` | 76 kel | **40 kel** | Samoja → Pasirwangi | **Server A** (continue) |
| `2/2` | 75 kel | **35 kel** | Babakan Asih → Pasirjati | **Server B** |

### Barbershop — 3 shard (partition 51 + 50 + 50 dari 151, lalu resume hapus done)

| Shard | Partition | After resume | First → Last | Assigned to |
|---|---|---|---|---|
| `1/3` | 51 kel | **48 kel** | Antapani Wetan → Pasirwangi | **Server A** |
| `2/3` | 50 kel | **48 kel** | Antapani Kulon → Pasir Endah | **Server B** instance #1 |
| `3/3` | 50 kel | **48 kel** | Antapani Tengah → Pasirjati | **Server B** instance #2 |

### Total instance per server

| Server | Instance | Workload |
|---|---|---|
| **Server A** | 2 | cafe shard `1/2` + barbershop shard `1/3` |
| **Server B** | 3 | cafe shard `2/2` + barbershop shard `2/3` + barbershop shard `3/3` |
| **Total** | **5** | 75 cafe + 144 barbershop = 219 kelurahan |

## Output folder structure (backward compatible)

Tetap pakai layout per-keyword yang sama dengan run sebelumnya — tidak ada breaking change:

```
/opt/scrapper/data/
├── kelurahan_bandung.json        ← seed (sama di semua VPS, tracked di repo)
├── cafe/
│   ├── progress.db               ← state scrape + sync (cleanup, 76 done baseline)
│   ├── Samoja.json               ← output baru (shard 1/2)
│   ├── Babakan_Asih.json         ← output baru (shard 2/2)
│   └── ... existing 75 file dari run sebelumnya (kalau di VPS A)
└── barbershop/
    ├── progress.db               ← state scrape + sync (cleanup, 7 done baseline)
    ├── Antapani_Kulon.json       ← output baru (shard 1/3)
    └── ... output baru
```

`save_raw_json()` di [src/storage.py:97](src/storage.py#L97) tetap nulis ke `data/<keyword>/<kelurahan>.json` — backward compatible 100%.

## Prerequisite per VPS

### 1. Image version

Tag minimum **`0.1.4`** (include `--shard` + WAL + **shard-first order** + delay 15/30 default):

```yaml
image: ghcr.io/dnahilman/scrapper-google-maps:0.1.4
```

> ⚠️ **Image `0.1.3` punya bug**: order resume-then-shard menyebabkan partition shift kalau scraper restart mid-run. **Jangan deploy `0.1.3` untuk multi-VPS.** Tunggu `v0.1.4` di-build (CI ~2 menit setelah tag push).

### 2. `.env.local` di VPS — **OPSIONAL**

Compose v0.1.4+ sudah bake semua default di `environment:` section:
- Scrape behavior: `HEADLESS=true`, `MIN_DELAY_SEC=15`, `MAX_DELAY_SEC=30`, dll
- Sync URLs: `CAFE_SYNC_URL`, `BARBERSHOP_SYNC_URL` (URL production langsung di compose)
- API key: kosong by default

**Untuk scrape-only mode** (saat ini): **tidak butuh `.env.local`** sama sekali. Container jalan langsung dengan defaults dari compose.

**Untuk sync mode** (nanti): set API key — pilih salah satu cara:

```bash
# Cara 1: .env.local (paling simple)
cat > /opt/scrapper/.env.local <<EOF
GOOGLE_MAPS_SYNC_API_KEY=6e9aef67f3229841509d7a888924eccf05bfe0bc43ff099eb2238526cdb21c82
EOF

# Cara 2: shell env var (sekali pakai)
GOOGLE_MAPS_SYNC_API_KEY=xxx docker compose up -d

# Cara 3: docker run secret / external secret manager (advanced)
```

Override delay per-VPS opsional:
```env
MIN_DELAY_SEC=10   # override default 15
MAX_DELAY_SEC=20   # override default 30
```

### 3. Sync `progress.db` baseline ke kedua VPS

Setelah cleanup local, copy `progress.db` cleaned dari local ke kedua VPS supaya state baseline sama:

```powershell
# Local → Server A
scp "C:\Users\Hype G12\Desktop\project\scrapper\data\cafe\progress.db" diosone@<SERVER_A_IP>:/opt/scrapper/data/cafe/progress.db
scp "C:\Users\Hype G12\Desktop\project\scrapper\data\barbershop\progress.db" diosone@<SERVER_A_IP>:/opt/scrapper/data/barbershop/progress.db

# Local → Server B
scp "C:\Users\Hype G12\Desktop\project\scrapper\data\cafe\progress.db" dios@<SERVER_B_IP>:/opt/scrapper/data/cafe/progress.db
scp "C:\Users\Hype G12\Desktop\project\scrapper\data\barbershop\progress.db" dios@<SERVER_B_IP>:/opt/scrapper/data/barbershop/progress.db
```

> Pastikan folder `data/cafe/` dan `data/barbershop/` sudah ada di VPS sebelum copy. Buat dengan `mkdir -p /opt/scrapper/data/{cafe,barbershop}`.

### 4. (Optional) Copy 75 file JSON cafe ke Server B

Server B tidak punya 75 file JSON cafe yang sudah ke-scrape sebelumnya. Untuk konsistensi & nanti gampang sync semua dari Server B juga, copy:

```powershell
rsync -avz "C:\Users\Hype G12\Desktop\project\scrapper\backups\cafe\" "dios@<SERVER_B_IP>:/opt/scrapper/data/cafe/"
```

Atau biarkan tidak di-copy — sync ke API tetap bisa dilakukan dari **central machine** setelah scrape selesai (lihat section akhir).

### 5. Resource

| Setting | Server A (2 instance) | Server B (3 instance) |
|---|---|---|
| RAM minimum | 2 GB | 4 GB |
| CPU | 2 vCPU | 2-3 vCPU |
| `mem_limit` di compose | 2g (default OK) | bump ke `4g` |

Kalau Server B pakai `docker-compose.prod.yml`, edit `mem_limit: 4g` (default 2g → tidak cukup untuk 3 Chromium).

## Deployment steps

### Server A (2 instance — continue cafe + barbershop)

```bash
cd /opt/scrapper

# 1. Pull image baru
sed -i 's|scrapper-google-maps:0\.1\.[123]|scrapper-google-maps:0.1.4|g' docker-compose.yml
docker compose pull
docker compose up -d
docker compose ps

# 2. Update helper scrape (kalau belum ada / belum versi terbaru)
curl -O https://raw.githubusercontent.com/dnahilman/scrapper-google-maps/dev/scrape
chmod +x scrape

# 3. Verify state baseline
./scrape progress cafe          # harus tampil: done = 76
./scrape progress barbershop    # harus tampil: done = 7

# 4. Spawn 2 shard (cafe 1/2 + barbershop 1/3)
docker exec -d scrapper-cafe python scripts/scraper.py --keyword cafe --resume --shard 1/2
docker exec -d scrapper-cafe python scripts/scraper.py --keyword barbershop --resume --shard 1/3

# 5. Verify proses jalan
./scrape ps
# Harus tampil 2 baris scraper.py: 1 cafe shard 1/2, 1 barbershop shard 1/3

# 6. Pantau
./scrape progress cafe          # done count nambah
./scrape progress barbershop
docker stats scrapper-cafe --no-stream
```

> **Catatan:** container name di Server A adalah `scrapper-cafe` (sesuai compose existing). Kalau scrape helper-nya hardcoded `scrapper-cafe`, OK. Kalau ganti ke `scrapper`, edit `CONTAINER` variable di awal file `scrape`.

### Server B (3 instance — 1 cafe + 2 barbershop)

```bash
cd /opt/scrapper

# 1. Setup compose (kalau belum ada). Pakai compose dari repo:
mkdir -p data/cafe data/barbershop logs
curl -O https://raw.githubusercontent.com/dnahilman/scrapper-google-maps/dev/docker-compose.yml
# Edit container_name kalau perlu, dan mem_limit jadi 4g
sed -i 's|scrapper-cafe|scrapper-bandung|g' docker-compose.yml

# 2. Setup .env.local (lihat section Prerequisite #2)
nano .env.local

# 3. Pull image
docker compose pull
docker compose up -d
docker compose ps

# 4. Setup helper
curl -O https://raw.githubusercontent.com/dnahilman/scrapper-google-maps/dev/scrape
chmod +x scrape
sed -i 's|CONTAINER="scrapper-cafe"|CONTAINER="scrapper-bandung"|' scrape

# 5. Verify baseline (harus tampil 76 cafe done, 7 barbershop done — dari progress.db yang di-scp tadi)
./scrape progress cafe
./scrape progress barbershop

# 6. Spawn 3 shard
docker exec -d scrapper-bandung python scripts/scraper.py --keyword cafe --resume --shard 2/2
docker exec -d scrapper-bandung python scripts/scraper.py --keyword barbershop --resume --shard 2/3
docker exec -d scrapper-bandung python scripts/scraper.py --keyword barbershop --resume --shard 3/3

# 7. Verify
./scrape ps
# Harus tampil 3 baris: cafe shard 2/2, barbershop shard 2/3, barbershop shard 3/3

# 8. Pantau
./scrape progress cafe
./scrape progress barbershop
docker stats scrapper-bandung --no-stream
```

## Skenario: Server B start belakangan (mis. malam hari)

**Aman 100%.** Karena partition pakai posisi absolut di 151 list, Server B yang start kapan pun akan compute partition yang sama untuk shard 2/2 (cafe) dan 2/3 + 3/3 (barbershop).

Yang harus dipastikan saat Server B mulai start:
1. Image `0.1.4` (yang sudah include fix shard-first), atau image apapun yang sudah punya order shard-then-resume.
2. `progress.db` baseline yang **sama** dengan saat Server A start (cleaned: 76 cafe done + 7 barbershop done). Bukan progress.db Server A yang sudah berkembang — **gunakan baseline cleaned dari local**.

**Step Server B malam hari:**
```bash
# Local — pastikan progress.db cleaned masih ada (jangan ditimpa)
ls "data/cafe/progress.db" "data/barbershop/progress.db"

# Local → Server B (scp progress.db cleaned, BUKAN dari Server A)
scp "data/cafe/progress.db" dios@<SERVER_B_IP>:/opt/scrapper/data/cafe/progress.db
scp "data/barbershop/progress.db" dios@<SERVER_B_IP>:/opt/scrapper/data/barbershop/progress.db

# SSH Server B → setup container + mulai 3 shard
# (sama dengan langkah utama Server B di section sebelumnya)
```

**Kenapa harus cleaned baseline, bukan Server A's progress.db?**
- Cleaned baseline punya 76 cafe done — semua di shard 1/2 atau 2/2 yang DISJOINT.
- Saat Server B `--resume`, dia hanya skip kelurahan baseline yang kebetulan jatuh di shard-nya (40 dari 76 cafe done jatuh di shard 2/2 → skip 40 → tinggal 35 untuk discrape).
- Server A's progress.db (yang sudah evolved) bisa berisi item baru di shard 1/2. Kalau di-copy ke Server B, dia akan skip item itu — tapi item itu memang gak ada di shard 2/2-nya Server B. Jadi tidak ada efek bahaya, **cuma redundant**. Pakai cleaned baseline lebih bersih.

**Yang TIDAK perlu disinkronkan antar VPS:**
- File JSON output (per-shard disjoint, beda kelurahan)
- progress.db yang berkembang (state lokal per VPS)

## Cara cek progress gabungan (dari mana saja)

Tiap VPS punya `progress.db` independen. Untuk cek total progress:

```bash
# Server A
ssh diosone@<SERVER_A_IP> 'cd /opt/scrapper && ./scrape progress cafe; ./scrape progress barbershop'

# Server B
ssh dios@<SERVER_B_IP> 'cd /opt/scrapper && ./scrape progress cafe; ./scrape progress barbershop'
```

Penjumlahan manual `done` count dari kedua VPS = total kelurahan yang sudah selesai.

## CAPTCHA handling

Kalau salah satu instance kena CAPTCHA berturut-turut, instance itu akan auto-stop (`MAX_CAPTCHA_RETRY=2`). Sisa instance yang sehat tetap jalan.

Recovery:
1. Tunggu 6-12 jam
2. Restart container untuk fresh browser fingerprint: `docker compose restart`
3. Jalankan ulang shard yang berhenti (sama command, `--resume` otomatis skip yang sudah done):
   ```bash
   docker exec -d <container> python scripts/scraper.py --keyword <kw> --resume --shard <K>/<N>
   ```

> Pakai IP/proxy berbeda per VPS sangat membantu — Server A pakai IP-nya, Server B pakai IP-nya. Total: kelurahan didistribusi via 2 IP berbeda → CAPTCHA risk jauh lebih rendah daripada 5 instance dari 1 IP.

## Setelah scrape selesai — sync ke API

Strategi: sync **dari 1 lokasi central**, bukan dari masing-masing VPS, supaya `sync_progress` (anti-duplicate) konsisten.

### 1. Pull semua JSON ke central (local atau 1 VPS dedicated sync)

```powershell
# Local (PowerShell), atau dari VPS dedicated:
mkdir -p data\cafe data\barbershop

# Pull dari Server A (cafe shard 1/2 + barbershop shard 1/3)
rsync -avz "diosone@<SERVER_A_IP>:/opt/scrapper/data/cafe/"  "data/cafe/"
rsync -avz "diosone@<SERVER_A_IP>:/opt/scrapper/data/barbershop/" "data/barbershop/"

# Pull dari Server B (cafe shard 2/2 + barbershop shard 2/3 & 3/3)
# rsync skip file yang sudah ada (size match), JSON dari Server A tetap aman
rsync -avz "dios@<SERVER_B_IP>:/opt/scrapper/data/cafe/"  "data/cafe/"
rsync -avz "dios@<SERVER_B_IP>:/opt/scrapper/data/barbershop/" "data/barbershop/"
```

### 2. Verify count

```powershell
(Get-ChildItem "data\cafe\*.json").Count        # harus 151 (atau 150-152 toleransi)
(Get-ChildItem "data\barbershop\*.json").Count  # harus 151
```

### 3. Sync ke API

```powershell
# Cafe
python scripts/sync.py --keyword cafe --all

# Barbershop
python scripts/sync.py --keyword barbershop --all
```

`sync.py` skip file yang sudah `done` di `sync_progress` table — aman dijalankan ulang. POST pakai JSON dispatch by keyword (lihat [src/sync_client.py](src/sync_client.py)).

## Risiko & checklist

| Risk | Mitigasi |
|---|---|
| `progress.db` baseline beda di kedua VPS → shard overlap | **Pre-flight WAJIB:** scp `progress.db` dari local ke kedua VPS sebelum start scrape. |
| CAPTCHA tinggi (3 browser di Server B) | Bumped delay 15-30s. Pisah IP via 2 VPS. Restart kalau berturut-turut kena. |
| RAM exhaust di Server B | `mem_limit: 4g` di compose. Pantau `docker stats`. |
| File JSON sama di-overwrite saat rsync | Rsync default skip kalau size+mtime match. Karena shard disjoint, file dari A & B tidak overlap. |
| `kelurahan_bandung.json` beda antar VPS | Selalu pakai versi yang sama (tracked di repo). Image `0.1.3` punya seed yang sama. |

## Checklist pre-deploy

- [ ] Local `progress.db` cafe & barbershop sudah cleanup (verified: 76 + 7 done)
- [ ] Backup snapshot ada di `backups/snapshots/`
- [ ] Image `ghcr.io/dnahilman/scrapper-google-maps:0.1.4` tersedia di GHCR
- [ ] `.env.local` di kedua VPS punya 3 env: API key + CAFE_SYNC_URL + BARBERSHOP_SYNC_URL
- [ ] `progress.db` cafe + barbershop di-scp ke kedua VPS
- [ ] Folder `data/cafe/` dan `data/barbershop/` ada di kedua VPS
- [ ] `scrape` helper installed di kedua VPS
- [ ] Container running di kedua VPS dengan image 0.1.3
- [ ] Verify `./scrape progress cafe` → 76 done, `./scrape progress barbershop` → 7 done

## Estimasi waktu

| Keyword | Total kelurahan | Avg per kel | Sequential | Dengan paralel |
|---|---|---|---|---|
| Cafe (75 remaining) | 75 | ~50 min | ~63 jam | 2 shard → ~32 jam |
| Barbershop (144 remaining) | 144 | ~12 min | ~29 jam | 3 shard → ~10 jam |
| **Combined** | **219** | — | ~92 jam | **~32 jam** (cafe bottleneck) |

Realistis: **1.5 hari** untuk semua selesai, dengan asumsi tidak ada CAPTCHA major + delay 15-30s.
