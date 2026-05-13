# syntax=docker/dockerfile:1.7

# ---------- Stage 1: build worker binary ----------
FROM golang:1.26.3-alpine AS build
WORKDIR /src
RUN apk add --no-cache git
COPY go.mod go.sum* ./
RUN go mod download || true
COPY . .
RUN CGO_ENABLED=0 GOOS=linux \
    go build -ldflags="-s -w" -o /out/worker ./cmd/worker

# ---------- Stage 2: Playwright base (Chromium preinstalled) ----------
FROM mcr.microsoft.com/playwright:v1.49.0-jammy
WORKDIR /app
COPY --from=build /out/worker /app/worker

RUN useradd -m -u 1000 worker && chown -R worker:worker /app
USER worker
ENTRYPOINT ["/app/worker"]
