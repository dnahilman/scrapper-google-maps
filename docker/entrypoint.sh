#!/usr/bin/env bash
set -e

# Pastikan mount points ada (kalau host belum bikin folder/file)
mkdir -p /app/data/output /app/logs
[ -f /app/progress.db ] || touch /app/progress.db

# Sanity check env wajib
if [ -z "$GOOGLE_MAPS_SYNC_API_KEY" ]; then
  echo "WARNING: GOOGLE_MAPS_SYNC_API_KEY kosong — sync ke API akan gagal" >&2
fi
if [ -z "$APP_URL" ]; then
  echo "WARNING: APP_URL kosong" >&2
fi

exec "$@"
