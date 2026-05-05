#!/usr/bin/env bash
set -e

# Pastikan mount points ada (kalau host belum bikin folder).
# Folder per-keyword (data/<keyword>/) dibuat otomatis oleh scraper saat dijalankan
# dengan flag --keyword.
mkdir -p /app/data /app/logs

# Sanity check — scrape mode tidak butuh API key, sync mode butuh.
if [ -z "$GOOGLE_MAPS_SYNC_API_KEY" ]; then
  echo "INFO: GOOGLE_MAPS_SYNC_API_KEY kosong — scrape OK, sync ke API akan error sampai di-set" >&2
fi

exec "$@"
