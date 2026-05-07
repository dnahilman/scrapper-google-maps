import json
from config import KELURAHAN_FILE


def load_kelurahan() -> list[dict]:
    if not KELURAHAN_FILE.exists():
        raise FileNotFoundError(
            f"Seed kelurahan tidak ditemukan: {KELURAHAN_FILE}\n"
            "Jalankan dulu: python fetch_kelurahan.py"
        )
    return json.loads(KELURAHAN_FILE.read_text(encoding="utf-8"))


def filter_kelurahan(name: str | None) -> list[dict]:
    items = load_kelurahan()
    if not name:
        return items
    needle = name.lower()
    return [k for k in items if needle in k["kelurahan"].lower()]
