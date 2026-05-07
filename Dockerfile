# Google Maps Scraper — production image (multi-keyword via CLI flag --keyword)
# Base: Playwright official Python image (Chromium pre-installed)
#
# Image generic — TIDAK ada secret yang di-bake. APP_URL + GOOGLE_MAPS_SYNC_API_KEY
# di-inject saat runtime via env_file di docker-compose.{dev,prod}.yml. Itu artinya
# image bisa public/private di GHCR tanpa khawatir leak credential.
FROM mcr.microsoft.com/playwright/python:v1.59.0-jammy

# Tuning args (non-secret) — defaults bisa di-override saat build di compose dev.
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

# Persistent dirs (akan di-bind mount ke host saat runtime). Per-keyword folder
# data/<keyword>/ + progress.db auto-created saat scraper jalan.
RUN mkdir -p /app/data /app/logs

# OCI labels — link image ke GitHub repo, supaya GHCR auto-detect repo source
LABEL org.opencontainers.image.source="https://github.com/dnahilman/scrapper-google-maps"
LABEL org.opencontainers.image.description="Bandung Google Maps Scraper (multi-keyword)"
LABEL org.opencontainers.image.licenses="MIT"

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
# Default: idle daemon — scrape di-trigger via `docker exec`
CMD ["sleep", "infinity"]
