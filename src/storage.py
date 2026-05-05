import json
import sqlite3
from datetime import datetime, timezone, timedelta
from pathlib import Path
from config import output_dir, progress_db, get_keyword
from src.transform import get_transformer

WIB = timezone(timedelta(hours=7))


def _connect() -> sqlite3.Connection:
    """Open SQLite connection dengan busy_timeout — wajib untuk concurrent writes."""
    conn = sqlite3.connect(progress_db(), timeout=10)
    conn.execute("PRAGMA busy_timeout=5000")
    return conn


def init_db() -> None:
    conn = _connect()
    conn.execute("PRAGMA journal_mode=WAL")
    conn.execute(
        """
        CREATE TABLE IF NOT EXISTS kelurahan_progress (
            kelurahan TEXT PRIMARY KEY,
            kecamatan TEXT,
            status TEXT NOT NULL,
            shop_count INTEGER DEFAULT 0,
            error TEXT,
            started_at TEXT,
            finished_at TEXT
        )
        """
    )
    conn.execute(
        """
        CREATE TABLE IF NOT EXISTS sync_progress (
            file_stem TEXT PRIMARY KEY,
            status TEXT NOT NULL,
            inserted INTEGER DEFAULT 0,
            skipped INTEGER DEFAULT 0,
            errors_count INTEGER DEFAULT 0,
            error TEXT,
            synced_at TEXT
        )
        """
    )
    conn.commit()
    conn.close()


def mark_started(kelurahan: str, kecamatan: str) -> None:
    conn = _connect()
    conn.execute(
        """
        INSERT INTO kelurahan_progress (kelurahan, kecamatan, status, started_at)
        VALUES (?, ?, 'in_progress', ?)
        ON CONFLICT(kelurahan) DO UPDATE SET
            status='in_progress', started_at=excluded.started_at, error=NULL
        """,
        (kelurahan, kecamatan, datetime.now(WIB).isoformat()),
    )
    conn.commit()
    conn.close()


def mark_done(kelurahan: str, shop_count: int) -> None:
    conn = _connect()
    conn.execute(
        """
        UPDATE kelurahan_progress
        SET status='done', shop_count=?, finished_at=?
        WHERE kelurahan=?
        """,
        (shop_count, datetime.now(WIB).isoformat(), kelurahan),
    )
    conn.commit()
    conn.close()


def mark_failed(kelurahan: str, error: str) -> None:
    conn = _connect()
    conn.execute(
        """
        UPDATE kelurahan_progress
        SET status='failed', error=?, finished_at=?
        WHERE kelurahan=?
        """,
        (error[:500], datetime.now(WIB).isoformat(), kelurahan),
    )
    conn.commit()
    conn.close()


def is_done(kelurahan: str) -> bool:
    conn = _connect()
    row = conn.execute(
        "SELECT status FROM kelurahan_progress WHERE kelurahan=?", (kelurahan,)
    ).fetchone()
    conn.close()
    return row is not None and row[0] == "done"


def save_raw_json(kelurahan: str, kecamatan: str, places: list[dict]) -> Path:
    """Simpan hasil scraping dalam SyncItem schema (langsung siap POST ke API).

    Transformer dipilih berdasarkan keyword aktif (lihat src/transform.get_transformer).
    Cafe/resto/kuliner pakai schema extended; lainnya pakai barbershop schema (default).
    """
    transform = get_transformer(get_keyword())
    items = [transform(s, kecamatan=kecamatan) for s in places]
    safe_name = kelurahan.replace("/", "_").replace(" ", "_")
    path = output_dir() / f"{safe_name}.json"
    path.write_text(json.dumps(items, ensure_ascii=False, indent=2), encoding="utf-8")
    return path


def progress_summary() -> dict:
    conn = _connect()
    rows = conn.execute(
        "SELECT status, COUNT(*) FROM kelurahan_progress GROUP BY status"
    ).fetchall()
    conn.close()
    return dict(rows)


# ============================================================================
# Sync progress tracking (anti-duplicate POST ke API)
# ============================================================================

def is_synced(file_stem: str) -> bool:
    conn = _connect()
    row = conn.execute(
        "SELECT status FROM sync_progress WHERE file_stem=?", (file_stem,)
    ).fetchone()
    conn.close()
    return row is not None and row[0] == "done"


def mark_synced(file_stem: str, inserted: int, skipped: int, errors_count: int) -> None:
    conn = _connect()
    conn.execute(
        """
        INSERT INTO sync_progress (file_stem, status, inserted, skipped, errors_count, synced_at)
        VALUES (?, 'done', ?, ?, ?, ?)
        ON CONFLICT(file_stem) DO UPDATE SET
            status='done', inserted=excluded.inserted, skipped=excluded.skipped,
            errors_count=excluded.errors_count, synced_at=excluded.synced_at, error=NULL
        """,
        (file_stem, inserted, skipped, errors_count, datetime.now(WIB).isoformat()),
    )
    conn.commit()
    conn.close()


def mark_sync_failed(file_stem: str, error: str) -> None:
    conn = _connect()
    conn.execute(
        """
        INSERT INTO sync_progress (file_stem, status, error, synced_at)
        VALUES (?, 'failed', ?, ?)
        ON CONFLICT(file_stem) DO UPDATE SET
            status='failed', error=excluded.error, synced_at=excluded.synced_at
        """,
        (file_stem, error[:500], datetime.now(WIB).isoformat()),
    )
    conn.commit()
    conn.close()


def sync_summary() -> dict:
    conn = _connect()
    rows = conn.execute(
        "SELECT status, COUNT(*), COALESCE(SUM(inserted),0) FROM sync_progress GROUP BY status"
    ).fetchall()
    conn.close()
    return {row[0]: {"count": row[1], "inserted": row[2]} for row in rows}
