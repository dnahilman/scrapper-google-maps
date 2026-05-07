"""FastAPI entry point — mount static UI + REST API.

Run dev (lokal, tanpa Docker):
    uvicorn web.app:app --reload --port 8000

Run production (di container, lihat Dockerfile CMD):
    uvicorn web.app:app --host 0.0.0.0 --port 8000 --workers 1
"""
from pathlib import Path

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import FileResponse, JSONResponse
from fastapi.staticfiles import StaticFiles

from web import __version__
from web.api import files as files_api
from web.api import health as health_api
from web.api import jobs as jobs_api
from web.api import logs as logs_api
from web.api import progress as progress_api

ROOT = Path(__file__).resolve().parents[1]
STATIC_DIR = ROOT / "web" / "static"

app = FastAPI(
    title="Google Maps Scraper Web UI",
    version=__version__,
    description="Control plane FastAPI + Svelte untuk scraper Bandung",
)

# Dev-only CORS — production deploy pada same-origin (FastAPI serve static).
# Kalau pakai dev mode dengan Vite di port 5173, izinkan localhost untuk hot reload.
app.add_middleware(
    CORSMiddleware,
    allow_origins=[
        "http://localhost:5173",
        "http://127.0.0.1:5173",
    ],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)


# ---------------------------------------------------------------------------
# REST API routers (prefix /api)
# ---------------------------------------------------------------------------
app.include_router(health_api.router, prefix="/api", tags=["health"])
app.include_router(jobs_api.router, prefix="/api", tags=["jobs"])
app.include_router(logs_api.router, prefix="/api", tags=["logs"])
app.include_router(files_api.router, prefix="/api", tags=["files"])
app.include_router(progress_api.router, prefix="/api", tags=["progress"])


# ---------------------------------------------------------------------------
# Static UI (Svelte build)
# ---------------------------------------------------------------------------
if STATIC_DIR.is_dir() and (STATIC_DIR / "index.html").is_file():
    # Mount semua asset (JS/CSS/font) di /assets, /favicon.svg, dst.
    app.mount("/assets", StaticFiles(directory=STATIC_DIR / "assets"), name="assets")

    @app.get("/", include_in_schema=False)
    async def root() -> FileResponse:
        return FileResponse(STATIC_DIR / "index.html")

    # SPA fallback — untuk hash-routing kita tidak butuh, tapi serve favicon dll.
    @app.get("/{path:path}", include_in_schema=False)
    async def spa_fallback(path: str):
        # path traversal guard
        if path.startswith("api/") or ".." in path:
            return JSONResponse({"detail": "not found"}, status_code=404)
        candidate = STATIC_DIR / path
        if candidate.is_file():
            return FileResponse(candidate)
        # Fallback ke index.html supaya hash routing work walau user reload halaman
        return FileResponse(STATIC_DIR / "index.html")
else:
    @app.get("/", include_in_schema=False)
    async def root_placeholder() -> JSONResponse:
        return JSONResponse(
            {
                "ok": True,
                "message": (
                    "FastAPI alive, tapi frontend belum di-build. "
                    "Run `cd frontend && npm install && npm run build` "
                    "atau gunakan Docker image yang sudah multi-stage build."
                ),
                "api_docs": "/docs",
            }
        )
