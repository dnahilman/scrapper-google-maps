# Google Maps Scraper — production image (multi-keyword via CLI flag --keyword)
# Base: Playwright official Python image (Chromium pre-installed)
#
# Image generic — TIDAK ada secret yang di-bake. APP_URL + GOOGLE_MAPS_SYNC_API_KEY
# di-inject saat runtime via env_file di docker-compose.{dev,prod}.yml. Itu artinya
# image bisa public/private di GHCR tanpa khawatir leak credential.
#
# Multi-stage:
#   Stage 1 (frontend-builder): build Svelte → static dist
#   Stage 2 (runtime):          Python + Playwright + FastAPI + dist (di-serve)

# ============================================================================
# Stage 1 — Frontend builder (Svelte → static)
# ============================================================================
FROM node:20-alpine AS frontend-builder

WORKDIR /build

# Cache deps layer
COPY web/package.json web/package-lock.json* ./
RUN npm ci --no-audit --no-fund

# Build static
COPY web/ ./
RUN npm run build

# ============================================================================
# Stage 2 — Runtime (Python + Playwright + FastAPI)
# ============================================================================
FROM mcr.microsoft.com/playwright/python:v1.59.0-jammy

ARG MIN_DELAY_SEC=10
ARG MAX_DELAY_SEC=25
ARG MAX_REVIEWS_PER_SHOP=200
ARG MAX_REVIEW_AGE_DAYS=730

ENV HEADLESS=true \
    MIN_DELAY_SEC=${MIN_DELAY_SEC} \
    MAX_DELAY_SEC=${MAX_DELAY_SEC} \
    MAX_REVIEWS_PER_SHOP=${MAX_REVIEWS_PER_SHOP} \
    MAX_REVIEW_AGE_DAYS=${MAX_REVIEW_AGE_DAYS} \
    SKIP_EMPTY_REVIEWS=true \
    SORT_REVIEWS_BY_NEWEST=true \
    MAX_CAPTCHA_RETRY=2 \
    LOG_LEVEL=INFO \
    TZ=Asia/Jakarta \
    PYTHONUNBUFFERED=1 \
    PYTHONDONTWRITEBYTECODE=1 \
    WEB_PORT=8000

WORKDIR /app

# dumb-init untuk reap zombies (uvicorn spawn child scrape via subprocess.Popen)
# + curl untuk healthcheck di compose
RUN apt-get update \
    && apt-get install -y --no-install-recommends dumb-init curl \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/* /var/cache/apt/archives/*

# Python deps (cache layer — copy requirements dulu sebelum source code)
COPY requirements.txt .
RUN pip install --no-cache-dir --no-compile -r requirements.txt \
    && rm -rf /root/.cache/pip

# Application code
COPY config.py ./
COPY server ./server
COPY docker/entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

# Static frontend dari stage 1 — hanya dist/ yang di-copy (node_modules di-discard)
COPY --from=frontend-builder /build/dist /app/server/static

# Persistent dirs (akan di-bind mount ke host saat runtime)
RUN mkdir -p /app/data /app/logs

LABEL org.opencontainers.image.source="https://github.com/dnahilman/scrapper-google-maps"
LABEL org.opencontainers.image.description="Bandung Google Maps Scraper + Web UI (FastAPI + Svelte)"
LABEL org.opencontainers.image.licenses="MIT"

EXPOSE 8000

# dumb-init sebagai PID 1 → reap zombie scrape children kalau uvicorn crash/restart
ENTRYPOINT ["dumb-init", "--"]

# Default mode: jalankan FastAPI + UI. Untuk legacy idle-daemon mode (sleep infinity),
# override entrypoint: docker run --entrypoint /usr/local/bin/entrypoint.sh ... sleep infinity
CMD ["uvicorn", "server.app:app", "--host", "0.0.0.0", "--port", "8000", "--workers", "1"]
