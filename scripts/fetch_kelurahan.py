"""Fetch daftar kelurahan Bandung dari wilayah.id API.

Kode wilayah:
- 32       = Jawa Barat (provinsi)
- 32.73    = Kota Bandung
- 32.73.xx = Kecamatan Bandung
"""
import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).parent.parent))

import json
import httpx
from config import KELURAHAN_FILE

BASE = "https://wilayah.id/api"
KOTA_BANDUNG = "32.73"


def main() -> None:
    print("Mengambil daftar kecamatan Kota Bandung dari wilayah.id ...")
    with httpx.Client(timeout=30.0) as client:
        kec_resp = client.get(f"{BASE}/districts/{KOTA_BANDUNG}.json")
        kec_resp.raise_for_status()
        kecamatans = kec_resp.json()["data"]
        print(f"Ditemukan {len(kecamatans)} kecamatan.")

        result: list[dict] = []
        for kec in kecamatans:
            kec_code = kec["code"]
            kec_name = kec["name"]
            print(f"  - {kec_name} ...", end=" ", flush=True)
            try:
                resp = client.get(f"{BASE}/villages/{kec_code}.json")
                resp.raise_for_status()
                villages = resp.json()["data"]
                for v in villages:
                    result.append({
                        "code": v["code"],
                        "kelurahan": v["name"],
                        "kecamatan": kec_name,
                        "kota": "Bandung",
                    })
                print(f"{len(villages)} kelurahan")
            except Exception as e:
                print(f"GAGAL ({e})")

    KELURAHAN_FILE.parent.mkdir(parents=True, exist_ok=True)
    KELURAHAN_FILE.write_text(json.dumps(result, ensure_ascii=False, indent=2), encoding="utf-8")
    print(f"\nTersimpan: {KELURAHAN_FILE}  (total {len(result)} kelurahan)")


if __name__ == "__main__":
    try:
        main()
    except Exception as e:
        print(f"ERROR: {e}", file=sys.stderr)
        sys.exit(1)
