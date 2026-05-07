"""POST data cafe per kelurahan ke API JSON (bukan multipart).

Endpoint  : POST {CAFES_SYNC_URL}
Header    : x-api-key: {GOOGLE_MAPS_SYNC_API_KEY}
Payload   : { "cafes": [ ...array isi file .json... ] }

Usage:
    python scripts/post_cafes.py                        # 5 kelurahan pertama
    python scripts/post_cafes.py --limit 10             # 10 kelurahan
    python scripts/post_cafes.py --limit 0              # semua kelurahan
    python scripts/post_cafes.py --force                # re-sync walau sudah pernah sukses
    python scripts/post_cafes.py --dry-run              # preview tanpa POST
    python scripts/post_cafes.py --kelurahan Antapani   # filter by nama (substring)
"""
import sys
import json
import argparse
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent.parent))

import httpx
from dotenv import load_dotenv

load_dotenv(Path(__file__).parent.parent / ".env.local")

import os
from config import DATA_DIR
from src.logger import get_logger
from src.storage import init_db, is_synced, mark_synced, mark_sync_failed, sync_summary

log = get_logger("post_cafes")

CAFES_SYNC_URL = os.getenv("CAFES_SYNC_URL", "http://localhost:3000/api/v1/sync/cafes")
API_KEY = os.getenv("GOOGLE_MAPS_SYNC_API_KEY", "")
DEFAULT_LIMIT = 5


def _to_float(val, default: float = 0.0) -> float:
    try:
        return float(val)
    except (TypeError, ValueError):
        return default


def sanitize(cafes: list) -> tuple[list, int, int]:
    """Bersihkan payload sebelum POST.

    - rating (cafe & review): null / non-number → 0.0
    - address kosong/null: item di-drop

    Returns: (cleaned_list, dropped_count, fixed_rating_count)
    """
    cleaned = []
    dropped = fixed = 0
    for cafe in cafes:
        if not cafe.get("address"):
            dropped += 1
            continue
        r = cafe.get("rating")
        if r is None or not isinstance(r, (int, float)):
            cafe = {**cafe, "rating": _to_float(r)}
            fixed += 1
        if cafe.get("reviews"):
            new_reviews = []
            for rev in cafe["reviews"]:
                rr = rev.get("rating")
                if rr is None or not isinstance(rr, (int, float)):
                    rev = {**rev, "rating": _to_float(rr)}
                    fixed += 1
                new_reviews.append(rev)
            cafe = {**cafe, "reviews": new_reviews}
        cleaned.append(cafe)
    return cleaned, dropped, fixed


def post_cafes(file_path: Path, dry_run: bool = False) -> dict:
    """Baca file JSON, sanitize, POST sebagai { cafes: [...] }. Return response dict."""
    payload: list = json.loads(file_path.read_text(encoding="utf-8"))
    if not isinstance(payload, list):
        raise ValueError(f"{file_path.name}: isi bukan array")

    cleaned, dropped, fixed = sanitize(payload)
    log.info(
        f"  {file_path.name} — {len(payload)} item "
        f"(dropped={dropped} no-address, fixed_rating={fixed}), "
        f"{file_path.stat().st_size // 1024} KB"
    )

    if dry_run:
        log.info("  [DRY-RUN] skip POST")
        return {"dry_run": True, "items": len(payload)}

    if not API_KEY:
        raise RuntimeError("GOOGLE_MAPS_SYNC_API_KEY kosong — set di .env.local")

    headers = {
        "x-api-key": API_KEY,
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


def main() -> None:
    parser = argparse.ArgumentParser(description="POST cafe JSON ke API")
    parser.add_argument(
        "--keyword", default="cafe",
        help="Sub-folder data/<keyword>/. Default: cafe",
    )
    parser.add_argument(
        "--limit", type=int, default=DEFAULT_LIMIT,
        help=f"Jumlah kelurahan yg diproses. 0 = semua. Default: {DEFAULT_LIMIT}",
    )
    parser.add_argument(
        "--kelurahan", default="",
        help="Filter nama file (substring, case-insensitive)",
    )
    parser.add_argument("--force", action="store_true", help="Re-sync meski sudah pernah sukses")
    parser.add_argument("--dry-run", action="store_true", help="Preview tanpa POST")
    args = parser.parse_args()

    # Perlu set keyword sebelum init_db supaya pakai DB yang benar
    import config
    config.set_keyword(args.keyword)
    init_db()

    data_dir = DATA_DIR / args.keyword
    if not data_dir.exists():
        log.error(f"Folder tidak ada: {data_dir}")
        sys.exit(1)

    all_files = sorted(data_dir.glob("*.json"))

    if args.kelurahan:
        needle = args.kelurahan.lower().replace(" ", "_")
        all_files = [f for f in all_files if needle in f.stem.lower()]
        if not all_files:
            log.error(f"Tidak ada file matching '{args.kelurahan}' di {data_dir}")
            sys.exit(1)

    files = all_files if args.limit == 0 else all_files[: args.limit]

    log.info(
        f"Target: {len(files)}/{len(all_files)} file dari {data_dir}"
        + (f" | filter='{args.kelurahan}'" if args.kelurahan else "")
        + (f" | limit={args.limit}" if args.limit else " | limit=semua")
        + (" | DRY-RUN" if args.dry_run else "")
        + (" | FORCE" if args.force else "")
    )
    log.info(f"URL: {CAFES_SYNC_URL}")

    ok = failed = skipped = total_inserted = 0

    for f in files:
        stem = f.stem
        if not args.force and not args.dry_run and is_synced(stem):
            log.info(f"--- {f.name} [SKIP, sudah sync] ---")
            skipped += 1
            continue

        log.info(f"--- {f.name} ---")
        try:
            result = post_cafes(f, dry_run=args.dry_run)
            ok += 1
            if not args.dry_run:
                inserted = result.get("inserted") or 0
                total_inserted += inserted
                mark_synced(
                    stem,
                    inserted=inserted,
                    skipped=result.get("skipped") or 0,
                    errors_count=len(result.get("errors") or []),
                )
        except Exception as e:
            failed += 1
            log.error(f"  GAGAL: {e}")
            if not args.dry_run:
                mark_sync_failed(stem, str(e))

    log.info(
        f"\nSelesai — ok={ok} skipped={skipped} failed={failed}"
        + (f" total_inserted={total_inserted}" if not args.dry_run else "")
    )
    if not args.dry_run:
        log.info(f"DB summary: {sync_summary()}")


if __name__ == "__main__":
    main()