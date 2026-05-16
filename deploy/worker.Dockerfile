# syntax=docker/dockerfile:1.7

# ---------- Stage 1: build worker binary ----------
FROM golang:1.26.3-alpine AS build
WORKDIR /src
RUN apk add --no-cache git
COPY server/go.mod server/go.sum* ./
RUN go mod download || true
COPY server/ .
RUN CGO_ENABLED=0 GOOS=linux \
    go build -ldflags="-s -w" -o /out/worker ./cmd/worker

# ---------- Stage 2: Playwright base (Chromium preinstalled) ----------
# Image tag MUST match the playwright version embedded in playwright-go
# (currently v1.57). The Microsoft image ships browsers under /ms-playwright.
FROM mcr.microsoft.com/playwright:v1.57.0-jammy
WORKDIR /app
ENV PLAYWRIGHT_BROWSERS_PATH=/ms-playwright
COPY --from=build /out/worker /app/worker

# The base image already provides a 'pwuser' (uid 1000). Reuse it instead of
# trying to add a duplicate.
RUN chown -R 1000:1000 /app
USER 1000
ENTRYPOINT ["/app/worker"]
