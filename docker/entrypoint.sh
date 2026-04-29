#!/usr/bin/env bash
set -e

# Pastikan mount points ada (kalau host belum bikin folder)
mkdir -p /app/data /app/logs
KW="${KEYWORD:-cafe}"
mkdir -p "/app/data/${KW}"

# Sanity check env wajib
if [ -z "$GOOGLE_MAPS_SYNC_API_KEY" ]; then
  echo "WARNING: GOOGLE_MAPS_SYNC_API_KEY kosong — sync ke API akan gagal" >&2
fi
if [ -z "$APP_URL" ]; then
  echo "WARNING: APP_URL kosong" >&2
fi

echo "[entrypoint] KEYWORD=${KW}  output=/app/data/${KW}"

exec "$@"
