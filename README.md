# Bandung Barbershop Scraper

Scraper Google Maps untuk mengumpulkan data **barbershop** di Kota Bandung, dengan scope per **kelurahan**. Output berupa **raw JSON** + database SQLite.

## Tujuan

Mengumpulkan dataset barbershop Bandung per kelurahan, mencakup:

- Nama, alamat, koordinat (lat/lng)
- Rating, jumlah review
- Jam buka, nomor telepon, website
- Daftar service / harga (jika tersedia)
- Reviews (teks, rating, waktu, owner response)
- Foto (URL)

## Stack Teknologi

| Komponen | Tools |
|---|---|
| Bahasa | Python 3.11+ (terdeteksi: 3.14.4 ✅) |
| Browser automation | Playwright (Chromium headless) |
| Anti-detection | playwright-stealth + custom user-agent rotation |
| Storage | SQLite (progress tracking) + JSON (raw output) |
| Logging | Python `logging` ke file + console |
| Seed data | wilayah.id API (daftar kelurahan Bandung) |

## Struktur Project

```
scrapper/
├── README.md
├── requirements.txt
├── .gitignore
├── .env.example
├── config.py                  # Konfigurasi (delay, retry, paths)
├── scraper.py                 # Entry point CLI
├── fetch_kelurahan.py         # Fetch seed kelurahan dari wilayah.id
├── src/
│   ├── __init__.py
│   ├── browser.py             # Playwright setup + stealth
│   ├── gmaps.py               # Scraping logic Google Maps
│   ├── seed.py                # Load kelurahan list
│   ├── storage.py             # SQLite + JSON writer
│   └── logger.py              # Logging setup
├── data/
│   ├── kelurahan_bandung.json # Seed data (151 kelurahan)
│   └── output/                # Raw JSON per kelurahan
├── logs/                      # Log file harian
└── progress.db                # SQLite progress tracker
```

## Setup (sekali saja)

```bash
# 1. Install dependencies
pip install -r requirements.txt

# 2. Install browser Chromium untuk Playwright
python -m playwright install chromium

# 3. Fetch daftar kelurahan Bandung (sekali saja)
python fetch_kelurahan.py
```

## Cara Pakai

```bash
# Jalankan scraping (semua kelurahan Bandung)
python scraper.py

# Jalankan untuk kelurahan tertentu saja
python scraper.py --kelurahan "Cihapit"

# Testing: scrape hanya N barbershop pertama per kelurahan
python scraper.py --kelurahan "Sukawarna" --limit 1

# Resume dari progress terakhir (otomatis skip yang sudah selesai)
python scraper.py --resume

# Dry-run (lihat berapa kelurahan akan di-scrape, tanpa eksekusi)
python scraper.py --dry-run
```

**Flag yang tersedia:**
| Flag | Default | Keterangan |
|---|---|---|
| `--kelurahan NAME` | semua | Substring match nama kelurahan |
| `--limit N` | unlimited | Maks N barbershop per kelurahan (untuk testing) |
| `--resume` | off | Skip kelurahan yang status `done` di SQLite |
| `--dry-run` | off | List kelurahan saja, tidak scrape |

## Jalan di Background

**Windows (Task Scheduler / hidden):**
```powershell
# Gunakan pythonw.exe (no console window)
Start-Process pythonw -ArgumentList "scraper.py --resume" -WindowStyle Hidden
```

**Monitor progress:**
```bash
# Tail log realtime
Get-Content logs/scraper.log -Wait -Tail 50

# Cek progress di SQLite
python -c "import sqlite3; c=sqlite3.connect('progress.db'); print(c.execute('SELECT status, COUNT(*) FROM kelurahan_progress GROUP BY status').fetchall())"
```

## Estimasi Runtime

- 151 kelurahan × rata-rata 8-15 barbershop × 20-30 detik per shop (termasuk reviews)
- **Total: ±6-12 jam** (jalankan overnight)

## ⚠️ Keamanan Akun Google (ANTI-BAN)

Project ini dirancang **TIDAK PERNAH** menyentuh akun Google Anda:

| Aturan | Implementasi |
|---|---|
| ❌ **JANGAN login** ke Google saat scraping | Browser context selalu dibuat baru, no persistent profile |
| ❌ **JANGAN pakai profile Chrome utama** | Playwright pakai isolated browser context |
| ❌ **JANGAN simpan cookie session** | `storage_state` tidak di-save, fresh context tiap run |
| ✅ **Gunakan IP residential rumah** | IP rumah lebih aman daripada datacenter/VPS |
| ✅ **Random delay agresif** | 5-15 detik random antar request |
| ✅ **Stealth mode** | playwright-stealth menyamarkan automation fingerprint |
| ✅ **Detect CAPTCHA** | Auto-pause + alert kalau Google curiga |
| ✅ **Rate limit otomatis** | Exponential backoff jika kena throttle |

**Risiko ban akun = NOL**, karena tidak ada akun Google yang dipakai sama sekali. Yang berisiko hanya **IP block sementara** (recover sendiri dalam beberapa jam).

### Tambahan Safety

- Scraper akan **otomatis berhenti** kalau:
  - CAPTCHA muncul lebih dari 3× berturut-turut
  - Halaman "unusual traffic" terdeteksi
  - Network error berturut-turut > 5×
- Semua output disimpan **inkremental** → progress tidak hilang kalau berhenti mendadak

## Output Schema

### `data/output/{kelurahan}.json`
```json
{
  "kelurahan": "Cihapit",
  "kecamatan": "Bandung Wetan",
  "scraped_at": "2026-04-22T20:30:00+07:00",
  "barbershops": [
    {
      "place_id": "ChIJxxxxx",
      "name": "Barbershop ABC",
      "address": "Jl. ...",
      "lat": -6.9039,
      "lng": 107.6186,
      "rating": 4.7,
      "review_count": 234,
      "phone": "+62...",
      "website": "https://...",
      "hours": {"Senin": "09.00-21.00", ...},
      "about": {
        "Layanan": ["Haircut", "Shaving", "Hair coloring", ...],
        "Aksesibilitas": ["Pintu masuk ramah kursi roda", ...],
        "Fasilitas": ["Wi-Fi", "Toilet", ...],
        "Cara pembayaran": ["Kartu debit", "Tunai", ...]
      },
      "services": ["Haircut", "Shaving", "Hair coloring", ...],
      "reviews": [
        {
          "review_id": "Ch...",
          "author": "John D.",
          "rating": 5,
          "rating_aria": "Diberi 5 bintang",
          "text": "...",
          "time": "2 minggu lalu",
          "owner_response": null
        }
      ]
    }
  ]
}
```

## Catatan Legal

Project ini untuk **riset / penggunaan personal**. Scraping Google Maps melanggar Google ToS. Untuk penggunaan komersial, gunakan [Google Places API](https://developers.google.com/maps/documentation/places/web-service) yang resmi.

## Troubleshooting

| Masalah | Solusi |
|---|---|
| `playwright install` gagal | Pastikan koneksi internet, retry. Atau install manual: `python -m playwright install chromium --with-deps` |
| CAPTCHA terus muncul | Berhenti, tunggu 6-12 jam, ganti IP (restart router), atau coba VPN |
| Browser crash | Cek RAM, kurangi `MAX_REVIEWS_PER_SHOP` di `config.py` |
| Data kelurahan kosong | Mungkin tidak ada barbershop di kelurahan kecil — cek log |

## Roadmap

- [ ] v1: Scrape Bandung 1 kota (current)
- [ ] v2: Tambah proxy rotation (kalau butuh scale)
- [ ] v3: Expand ke kota lain (Jakarta, Surabaya)
- [ ] v4: Scheduling berkala (monitoring perubahan rating)

```
Terminal #1 — jalankan scraper:


cd "c:/Users/Hype G12/Desktop/project/scrapper"
python scraper.py --resume
Terminal #2 — monitor progress realtime (PowerShell):


cd "c:\Users\Hype G12\Desktop\project\scrapper"
Get-Content logs/scraper-*.log -Wait -Tail 30


python -c "import sqlite3; c=sqlite3.connect('progress.db'); print(dict(c.execute('SELECT status, COUNT(*) FROM kelurahan_progress GROUP BY status').fetchall()))"

```

# powercfg -change -standby-timeout-ac 30

# powercfg -change -standby-timeout-ac 0
# powercfg -change -hibernate-timeout-ac 0
# powercfg -change -monitor-timeout-ac 30

# Get-Content logs/scraper-*.log -Wait -Tail 30
# python scraper.py --resume