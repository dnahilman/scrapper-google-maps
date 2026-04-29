# Google Maps Scraper — production image (keyword configurable)
# Base: Playwright official Python image (Chromium pre-installed)
FROM mcr.microsoft.com/playwright/python:v1.58.0-jammy

# Build-time secrets (di-bake ke image — image WAJIB private di GHCR)
ARG APP_URL=https://api.hilman.imola.ai/api
ARG GOOGLE_MAPS_SYNC_API_KEY=""
ARG KEYWORD=cafe
ARG MIN_DELAY_SEC=10
ARG MAX_DELAY_SEC=25
ARG MAX_REVIEWS_PER_SHOP=200
ARG MAX_REVIEW_AGE_DAYS=730

ENV APP_URL=${APP_URL} \
    GOOGLE_MAPS_SYNC_API_KEY=${GOOGLE_MAPS_SYNC_API_KEY} \
    KEYWORD=${KEYWORD} \
    HEADLESS=true \
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
    PYTHONDONTWRITEBYTECODE=1

WORKDIR /app

# Dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Application code
COPY config.py ./
COPY src ./src
COPY scripts ./scripts
COPY docker/entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

# Persistent dirs (akan di-bind mount ke host saat runtime). progress.db tinggal di
# /app/data/<keyword>/progress.db — auto-created saat scraper start.
RUN mkdir -p /app/data /app/logs

# OCI labels — link image ke GitHub repo, supaya GHCR auto-detect repo source
LABEL org.opencontainers.image.source="https://github.com/dnahilman/scrapper-google-maps"
LABEL org.opencontainers.image.description="Bandung Google Maps Scraper (multi-keyword)"
LABEL org.opencontainers.image.licenses="MIT"

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
# Default: idle daemon — scrape di-trigger via `docker exec`
CMD ["sleep", "infinity"]
