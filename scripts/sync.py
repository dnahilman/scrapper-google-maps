"""POST file JSON hasil scraping ke API sync-google-maps.

Sync progress tracked di SQLite (tabel sync_progress) — file yang sudah sukses
sync di-skip otomatis. Pakai --force untuk re-sync semua.

Usage:
    python scripts/sync.py --keyword cafe --kelurahan "Sukawarna"   # sync 1 file
    python scripts/sync.py --keyword cafe --all                      # sync semua (skip yang sudah)
    python scripts/sync.py --keyword cafe --all --force              # force re-sync semua
    python scripts/sync.py --keyword cafe --file path/to/file.json   # file spesifik
    python scripts/sync.py --keyword cafe --kelurahan "Sukawarna" --dry-run   # preview
"""
import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).parent.parent))

import argparse

import config
from config import APP_URL, SYNC_ENDPOINT
from src.logger import get_logger
from src.storage import init_db, is_synced, mark_synced, mark_sync_failed, sync_summary
from src.sync_client import post_file

log = get_logger("sync")


def main() -> None:
    parser = argparse.ArgumentParser(description="Sync JSON ke API sync-google-maps")
    parser.add_argument(
        "--keyword",
        default="cafe",
        help="Target keyword (folder data/<keyword>/). Default: cafe",
    )
    g = parser.add_mutually_exclusive_group(required=True)
    g.add_argument("--kelurahan", help="Sync file kelurahan spesifik (substring match)")
    g.add_argument("--file", help="Path absolut ke file JSON")
    g.add_argument("--all", action="store_true", help="Sync semua file di data/{keyword}/")
    parser.add_argument("--dry-run", action="store_true", help="Preview, tidak POST")
    parser.add_argument("--force", action="store_true", help="Re-sync file yang sudah pernah sukses")
    args = parser.parse_args()
    config.set_keyword(args.keyword)

    init_db()

    out_dir = config.output_dir()
    files: list[Path] = []
    if args.file:
        files = [Path(args.file)]
    elif args.kelurahan:
        needle = args.kelurahan.lower().replace(" ", "_")
        files = sorted(p for p in out_dir.glob("*.json") if needle in p.stem.lower())
        if not files:
            print(f"Tidak ada file matching '{args.kelurahan}' di {out_dir}", file=sys.stderr)
            sys.exit(1)
    else:
        files = sorted(out_dir.glob("*.json"))

    log.info(
        f"Sync {len(files)} file ke {APP_URL}{SYNC_ENDPOINT}"
        + (" (DRY-RUN)" if args.dry_run else "")
        + (" (FORCE)" if args.force else "")
    )

    success = 0
    failed = 0
    skipped = 0
    total_inserted = 0
    for f in files:
        stem = f.stem
        if not args.force and not args.dry_run and is_synced(stem):
            log.info(f"--- {f.name} --- [SKIP, sudah sync]")
            skipped += 1
            continue
        try:
            log.info(f"--- {f.name} ---")
            result = post_file(f, dry_run=args.dry_run)
            success += 1
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
            log.error(f"  GAGAL {f.name}: {e}")
            if not args.dry_run:
                mark_sync_failed(stem, str(e))

    log.info(
        f"Selesai: success={success} skipped={skipped} failed={failed}"
        + (f" total_inserted={total_inserted}" if not args.dry_run else "")
    )
    if not args.dry_run:
        log.info(f"DB summary: {sync_summary()}")


if __name__ == "__main__":
    main()
