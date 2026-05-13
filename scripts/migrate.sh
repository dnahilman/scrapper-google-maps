#!/usr/bin/env bash
# Run migrations against a running master container.
# Usage: ./scripts/migrate.sh
set -euo pipefail
docker compose -f deploy/docker-compose.yml exec master /app/master migrate
