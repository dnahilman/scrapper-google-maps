"""One-time migration: transform file JSON lama (schema {kelurahan, barbershops})
ke schema baru (array of SyncItem). Replace file in-place.

Idempotent: file yang sudah dalam schema baru dilewati.

Usage:
    python scripts/migrate.py                            # migrate semua file
    python scripts/migrate.py --kelurahan "Sukawarna"   # spesifik
    python scripts/migrate.py --dry-run                  # preview
"""
import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).parent.parent))

import argparse
import json

from config import OUTPUT_DIR
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
    parser.add_argument("--kelurahan", help="Migrate file kelurahan spesifik (substring)")
    parser.add_argument("--dry-run", action="store_true", help="Preview, tidak overwrite")
    args = parser.parse_args()

    if args.kelurahan:
        needle = args.kelurahan.lower().replace(" ", "_")
        files = sorted(p for p in OUTPUT_DIR.glob("*.json") if needle in p.stem.lower())
    else:
        files = sorted(OUTPUT_DIR.glob("*.json"))

    if not files:
        print(f"Tidak ada file di {OUTPUT_DIR}", file=sys.stderr)
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
