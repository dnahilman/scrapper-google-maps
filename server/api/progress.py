"""Progress endpoints — query SQLite progress.db langsung (read-only).

Tidak pakai `src/storage.py` karena modul itu butuh `config.set_keyword(...)` global,
yang tidak aman di multi-request FastAPI (race antar request). Read direct dari path
file `data/<keyword>/progress.db` adalah cleaner + thread-safe.
"""
import sqlite3

from fastapi import APIRouter, HTTPException

from server.models import KelurahanProgress, ProgressSummary
from server.services.job_manager import DATA_DIR

router = APIRouter()


def _progress_db_path(keyword: str):
    if not keyword or "/" in keyword or "\\" in keyword or keyword.startswith("."):
        raise HTTPException(status_code=400, detail="invalid keyword")
    p = DATA_DIR / keyword / "progress.db"
    if not p.exists():
        raise HTTPException(status_code=404, detail="progress.db not found for keyword")
    return p


def _open_ro(path) -> sqlite3.Connection:
    """Open SQLite read-only (URI mode) — aman untuk concurrent dengan scraper writes."""
    uri = f"file:{path}?mode=ro"
    conn = sqlite3.connect(uri, uri=True, timeout=10)
    conn.row_factory = sqlite3.Row
    return conn


@router.get("/keywords/{keyword}/progress", response_model=ProgressSummary)
async def progress_summary(keyword: str) -> ProgressSummary:
    path = _progress_db_path(keyword)
    conn = _open_ro(path)
    try:
        rows = conn.execute(
            "SELECT kelurahan, kecamatan, status, shop_count, error, started_at, finished_at "
            "FROM kelurahan_progress ORDER BY finished_at DESC NULLS LAST, started_at DESC"
        ).fetchall()
        counts: dict[str, int] = {"done": 0, "in_progress": 0, "failed": 0}
        items: list[KelurahanProgress] = []
        for r in rows:
            d = dict(r)
            counts[d["status"]] = counts.get(d["status"], 0) + 1
            items.append(KelurahanProgress(**d))
        return ProgressSummary(
            keyword=keyword,
            counts=counts,
            total=sum(counts.values()),
            items=items,
        )
    finally:
        conn.close()


@router.get("/keywords/{keyword}/progress/failed", response_model=list[KelurahanProgress])
async def list_failed(keyword: str) -> list[KelurahanProgress]:
    path = _progress_db_path(keyword)
    conn = _open_ro(path)
    try:
        rows = conn.execute(
            "SELECT kelurahan, kecamatan, status, shop_count, error, started_at, finished_at "
            "FROM kelurahan_progress WHERE status='failed' ORDER BY finished_at DESC"
        ).fetchall()
        return [KelurahanProgress(**dict(r)) for r in rows]
    finally:
        conn.close()


@router.post("/keywords/{keyword}/progress/{kelurahan}/reset")
async def reset_kelurahan(keyword: str, kelurahan: str) -> dict:
    """Hapus row kelurahan dari progress.db supaya scraper next run akan re-scrape.
    Tidak hapus output JSON yang sudah ada (overwrite saat re-scrape).
    """
    path = _progress_db_path(keyword)
    # Re-open RW (bukan via _open_ro)
    conn = sqlite3.connect(path, timeout=10)
    conn.execute("PRAGMA busy_timeout=5000")
    try:
        cur = conn.execute(
            "DELETE FROM kelurahan_progress WHERE kelurahan=?", (kelurahan,)
        )
        conn.commit()
        deleted = cur.rowcount
    finally:
        conn.close()
    if deleted == 0:
        raise HTTPException(status_code=404, detail="kelurahan not found in progress")
    return {"ok": True, "keyword": keyword, "kelurahan": kelurahan, "deleted": deleted}
