# syntax=docker/dockerfile:1.7

# ---------- Stage 1: build Svelte UI ----------
FROM node:20-alpine AS ui
WORKDIR /ui
COPY web/package*.json ./
RUN npm install --no-audit --no-fund
COPY web/ ./
RUN npm run build

# ---------- Stage 2: build Go binary ----------
FROM golang:1.26.3-alpine AS build
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY server/go.mod server/go.sum* ./
RUN go mod download || true
COPY server/ .
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux \
    go build -ldflags="-s -w -X github.com/dnahilman/scrapper-go/internal/version.Version=${VERSION}" \
    -o /out/master ./cmd/master

# ---------- Stage 3: slim runtime ----------
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/master /app/master
COPY --from=ui /ui/dist /app/web/dist
COPY server/migrations /app/migrations
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/app/master"]
CMD ["serve"]
