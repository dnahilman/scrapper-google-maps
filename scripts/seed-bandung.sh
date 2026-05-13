#!/usr/bin/env bash
# Seed Bandung kelurahan via the master API.
# Usage: ./scripts/seed-bandung.sh [base_url]
set -euo pipefail

BASE_URL="${1:-http://localhost:8080}"

echo "==> Syncing all Indonesian cities from emsifa..."
curl -fsS -X POST "${BASE_URL}/api/v1/cities/sync" | jq .

echo "==> Resolving Kota Bandung..."
CITY_ID=$(curl -fsS "${BASE_URL}/api/v1/cities/bandung" | jq -r '.city.id')
echo "    city_id=${CITY_ID}"

echo "==> Syncing kelurahan for Kota Bandung..."
curl -fsS -X POST "${BASE_URL}/api/v1/cities/${CITY_ID}/kelurahan/sync" | jq .

echo "==> Done. Listing first 5 kelurahan:"
curl -fsS "${BASE_URL}/api/v1/cities/${CITY_ID}/kelurahan" | jq '.[0:5]'
