"""HTTP client untuk POST hasil scraping ke API sync.

Format: JSON body `{ cafes: [...] }` dengan header `x-api-key`.
Endpoint: `CAFES_SYNC_URL` dari `.env.local`.

Sanitize otomatis sebelum POST:
- Item tanpa `address` di-drop.
- `rating` (item & nested reviews) yang null / non-number → 0.0.

Module ini di-share antara:
- scripts/sync.py (CLI manual sync)
- scripts/scraper.py --auto-sync (sync inline setelah tiap kelurahan selesai)
"""
import json
from pathlib import Path

import httpx

from config import CAFES_SYNC_URL, GOOGLE_MAPS_SYNC_API_KEY, output_dir
from src.logger import get_logger
from src.storage import is_synced, mark_synced, mark_sync_failed

log = get_logger("sync_client")


def _to_float(val, default: float = 0.0) -> float:
    try:
        return float(val)
    except (TypeError, ValueError):
        return default


def _sanitize(items: list) -> tuple[list, int, int]:
    cleaned: list = []
    dropped = fixed = 0
    for item in items:
        if not item.get("address"):
            dropped += 1
            continue
        r = item.get("rating")
        if r is None or not isinstance(r, (int, float)):
            item = {**item, "rating": _to_float(r)}
            fixed += 1
        if item.get("reviews"):
            new_reviews = []
            for rev in item["reviews"]:
                rr = rev.get("rating")
                if rr is None or not isinstance(rr, (int, float)):
                    rev = {**rev, "rating": _to_float(rr)}
                    fixed += 1
                new_reviews.append(rev)
            item = {**item, "reviews": new_reviews}
        cleaned.append(item)
    return cleaned, dropped, fixed


def post_file(file_path: Path, dry_run: bool = False) -> dict:
    """Baca file JSON, sanitize, POST sebagai `{ cafes: [...] }`. Return result dict."""
    if not file_path.exists():
        raise FileNotFoundError(file_path)

    payload = json.loads(file_path.read_text(encoding="utf-8"))
    if not isinstance(payload, list):
        raise ValueError(
            f"{file_path.name}: payload bukan array. Mungkin masih schema lama — "
            "jalankan: python scripts/migrate.py"
        )

    cleaned, dropped, fixed = _sanitize(payload)
    log.info(
        f"  {file_path.name} — {len(payload)} item "
        f"(dropped={dropped} no-address, fixed_rating={fixed}), "
        f"{file_path.stat().st_size // 1024} KB"
    )

    if dry_run:
        log.info("  [DRY-RUN] Skip POST, hanya validate file")
        return {"dry_run": True, "items": len(payload)}

    if not GOOGLE_MAPS_SYNC_API_KEY:
        raise RuntimeError("GOOGLE_MAPS_SYNC_API_KEY kosong. Set di .env.local.")
    if not CAFES_SYNC_URL:
        raise RuntimeError("CAFES_SYNC_URL kosong. Set di .env.local.")

    headers = {
        "x-api-key": GOOGLE_MAPS_SYNC_API_KEY,
        "Content-Type": "application/json",
    }
    body = {"cafes": cleaned}

    with httpx.Client(timeout=300) as client:
        r = client.post(CAFES_SYNC_URL, headers=headers, json=body)

    log.info(f"  HTTP {r.status_code}")
    try:
        resp = r.json()
    except Exception:
        resp = {"raw": r.text[:500]}

    if r.status_code >= 400:
        log.error(f"  ERROR: {resp}")
        raise httpx.HTTPStatusError(
            f"API {r.status_code}: {resp}", request=r.request, response=r
        )

    data = resp.get("data", resp)
    log.info(
        f"  inserted={data.get('inserted')} skipped={data.get('skipped')} "
        f"errors={len(data.get('errors') or [])}"
    )
    if data.get("errors"):
        for err in (data["errors"] or [])[:5]:
            log.warning(f"    - {err}")
    return data


def sync_one_kelurahan(file_stem: str, force: bool = False) -> bool:
    """Sync 1 file by stem. Cek is_synced, mark setelah sukses/gagal.

    Args:
        file_stem: nama file tanpa .json, mis. 'Sukawarna' atau 'Antapani_Kidul'
        force: kalau True, sync ulang walau sudah marked done

    Returns:
        True kalau sukses (atau di-skip karena sudah synced). False kalau gagal POST.
    """
    if not force and is_synced(file_stem):
        log.info(f"  Skip {file_stem}: sudah ter-sync sebelumnya")
        return True

    file_path = output_dir() / f"{file_stem}.json"
    if not file_path.exists():
        log.warning(f"  File tidak ada: {file_path}")
        return False

    try:
        result = post_file(file_path)
        mark_synced(
            file_stem,
            inserted=result.get("inserted") or 0,
            skipped=result.get("skipped") or 0,
            errors_count=len(result.get("errors") or []),
        )
        return True
    except Exception as e:
        log.error(f"  Sync {file_stem} GAGAL: {e}")
        mark_sync_failed(file_stem, str(e))
        return False
