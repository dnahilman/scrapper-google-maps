"""HTTP client untuk POST hasil scraping ke API sync-google-maps.

Module ini di-share antara:
- scripts/sync.py (CLI manual sync)
- scripts/scraper.py --auto-sync (sync inline setelah tiap kelurahan selesai)
"""
import json
from pathlib import Path

import httpx

from config import APP_URL, GOOGLE_MAPS_SYNC_API_KEY, OUTPUT_DIR, SYNC_ENDPOINT
from src.logger import get_logger
from src.storage import is_synced, mark_synced, mark_sync_failed

log = get_logger("sync_client")


def post_file(file_path: Path, dry_run: bool = False) -> dict:
    """POST 1 file JSON ke API. Return result dict (sudah unwrap dari {data:{...}}).
    Raise httpx.HTTPStatusError kalau API return 4xx/5xx."""
    if not file_path.exists():
        raise FileNotFoundError(file_path)

    payload = json.loads(file_path.read_text(encoding="utf-8"))
    if not isinstance(payload, list):
        raise ValueError(
            f"{file_path.name}: payload bukan array. Mungkin masih schema lama — "
            "jalankan: python scripts/migrate.py"
        )

    log.info(f"  File: {file_path.name} ({len(payload)} items, {file_path.stat().st_size // 1024} KB)")

    if dry_run:
        log.info("  [DRY-RUN] Skip POST, hanya validate file")
        return {"dry_run": True, "items": len(payload)}

    if not GOOGLE_MAPS_SYNC_API_KEY:
        raise RuntimeError("GOOGLE_MAPS_SYNC_API_KEY kosong. Set di .env.local atau build args.")

    url = f"{APP_URL.rstrip('/')}{SYNC_ENDPOINT}"
    with file_path.open("rb") as f:
        files = {"file": (file_path.name, f, "application/json")}
        data = {"apiKey": GOOGLE_MAPS_SYNC_API_KEY}
        with httpx.Client(timeout=300) as client:
            r = client.post(url, files=files, data=data)

    log.info(f"  Status: {r.status_code}")
    try:
        body = r.json()
    except Exception:
        body = {"raw": r.text[:500]}

    if r.status_code >= 400:
        log.error(f"  ERROR: {body}")
        raise httpx.HTTPStatusError(
            f"API {r.status_code}: {body}", request=r.request, response=r
        )

    # Response dibungkus di {data: {...}}
    result = body.get("data") if isinstance(body, dict) and "data" in body else body
    log.info(
        f"  Result: total={result.get('total')} inserted={result.get('inserted')} "
        f"skipped={result.get('skipped')} errors={len(result.get('errors') or [])}"
    )
    if result.get("errors"):
        for err in result["errors"][:5]:
            log.warning(f"    - {err}")
    return result


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

    file_path = OUTPUT_DIR / f"{file_stem}.json"
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
