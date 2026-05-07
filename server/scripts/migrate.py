"""One-time migration: transform file JSON lama (schema {kelurahan, barbershops})
ke schema baru (array of SyncItem). Replace file in-place.

Idempotent: file yang sudah dalam schema baru dilewati.

Usage:
    python scripts/migrate.py --keyword cafe                          # migrate semua file
    python scripts/migrate.py --keyword cafe --kelurahan "Sukawarna"  # spesifik
    python scripts/migrate.py --keyword cafe --dry-run                # preview
"""
import sys
from pathlib import Path
_server = Path(__file__).parent.parent
sys.path.insert(0, str(_server.parent))
sys.path.insert(0, str(_server))

import argparse
import json

import config
from src.transform import is_already_sync_schema, transform_payload


def migrate_file(path: Path, dry_run: bool = False) -> tuple[str, int]:
    """Return (status, item_count). Status: 'migrated' | 'skipped' | 'error'."""
    try:
        payload = json.loads(path.read_text(encoding="utf-8"))
    except Exception:
        return ("error", 0)

    if is_already_sync_schema(payload):
        return ("skipped", len(payload))

    items = transform_payload(payload)
    if not dry_run:
        path.write_text(json.dumps(items, ensure_ascii=False, indent=2), encoding="utf-8")
    return ("migrated", len(items))


def main() -> None:
    parser = argparse.ArgumentParser(description="Migrate file JSON lama ke SyncItem schema")
    parser.add_argument(
        "--keyword",
        default="cafe",
        help="Target keyword (folder data/<keyword>/). Default: cafe",
    )
    parser.add_argument("--kelurahan", help="Migrate file kelurahan spesifik (substring)")
    parser.add_argument("--dry-run", action="store_true", help="Preview, tidak overwrite")
    args = parser.parse_args()
    config.set_keyword(args.keyword)

    out_dir = config.output_dir()
    if args.kelurahan:
        needle = args.kelurahan.lower().replace(" ", "_")
        files = sorted(p for p in out_dir.glob("*.json") if needle in p.stem.lower())
    else:
        files = sorted(out_dir.glob("*.json"))

    if not files:
        print(f"Tidak ada file di {out_dir}", file=sys.stderr)
        sys.exit(1)

    print(f"Migrate {len(files)} file" + (" (DRY-RUN)" if args.dry_run else ""))
    counts = {"migrated": 0, "skipped": 0, "error": 0}
    for f in files:
        status, n = migrate_file(f, dry_run=args.dry_run)
        counts[status] += 1
        marker = {"migrated": "[OK]", "skipped": "[--]", "error": "[ER]"}[status]
        print(f"  {marker} {f.name:40} -> {status:8} ({n} items)")

    print(f"\nSelesai: {counts}")


if __name__ == "__main__":
    main()
