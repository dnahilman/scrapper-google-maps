#!/usr/bin/env bash
set -e

# Pastikan mount points ada (kalau host belum bikin folder).
# Folder per-keyword (data/<keyword>/) dibuat otomatis oleh scraper saat dijalankan
# dengan flag --keyword.
mkdir -p /app/data /app/logs

# Sanity check env wajib
if [ -z "$GOOGLE_MAPS_SYNC_API_KEY" ]; then
  echo "WARNING: GOOGLE_MAPS_SYNC_API_KEY kosong — sync ke API akan gagal" >&2
fi
if [ -z "$CAFES_SYNC_URL" ]; then
  echo "WARNING: CAFES_SYNC_URL kosong — sync ke API akan gagal" >&2
fi

exec "$@"
