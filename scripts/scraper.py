"""Entry point CLI untuk scraping barbershop Bandung per kelurahan."""
import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).parent.parent))

import argparse
import asyncio
from tqdm.asyncio import tqdm
from config import MAX_CAPTCHA_RETRY, MAX_NETWORK_ERRORS
from src.browser import new_browser_session
from src.gmaps import search_barbershops, scrape_place, CaptchaDetected
from src.logger import get_logger
from src.seed import filter_kelurahan
from src.storage import (
    init_db, mark_started, mark_done, mark_failed,
    is_done, save_raw_json, progress_summary,
)
from src.sync_client import sync_one_kelurahan

log = get_logger("main")


async def scrape_kelurahan(context, kel: dict, limit: int | None = None) -> int:
    name, kec = kel["kelurahan"], kel["kecamatan"]
    page = await context.new_page()
    try:
        urls = await search_barbershops(page, name, kec, limit=limit)
        shops: list[dict] = []
        net_errors = 0
        for i, url in enumerate(urls, 1):
            try:
                data = await scrape_place(page, url)
                if not data:
                    log.info(f"  [{i}/{len(urls)}] SKIP")
                    continue
                log.info(f"  [{i}/{len(urls)}] {data.get('name')}")

                # Services = ambil dari About amenities (sumber lain belum reliable / CAPTCHA)
                data["services"] = list(data.get("services_about") or [])

                shops.append(data)
            except CaptchaDetected:
                raise
            except Exception as e:
                net_errors += 1
                log.warning(f"  [{i}/{len(urls)}] ERROR: {e}")
                if net_errors >= MAX_NETWORK_ERRORS:
                    log.error("Terlalu banyak network error, stop kelurahan ini")
                    break
        save_raw_json(name, kec, shops)
        return len(shops)
    finally:
        await page.close()


async def run(kelurahan_filter: str | None, resume: bool, limit: int | None, auto_sync: bool = False) -> None:
    init_db()
    items = filter_kelurahan(kelurahan_filter)
    if resume:
        items = [k for k in items if not is_done(k["kelurahan"])]
    log.info(
        f"Akan scrape {len(items)} kelurahan"
        + (f" (limit {limit} shop/kelurahan)" if limit else "")
        + (" [AUTO-SYNC]" if auto_sync else "")
    )

    captcha_streak = 0
    async with new_browser_session() as (browser, context):
        for kel in tqdm(items, desc="Kelurahan"):
            name = kel["kelurahan"]
            mark_started(name, kel["kecamatan"])
            try:
                count = await scrape_kelurahan(context, kel, limit=limit)
                mark_done(name, count)
                captcha_streak = 0
                log.info(f"DONE {name}: {count} barbershop")

                if auto_sync and count > 0:
                    file_stem = name.replace("/", "_").replace(" ", "_")
                    log.info(f"  Auto-sync {file_stem} ...")
                    sync_one_kelurahan(file_stem, force=True)
            except CaptchaDetected as e:
                captcha_streak += 1
                mark_failed(name, str(e))
                log.error(f"CAPTCHA #{captcha_streak} di {name}")
                if captcha_streak >= MAX_CAPTCHA_RETRY:
                    log.critical("CAPTCHA berturut-turut — STOP. Tunggu 6-12 jam, ganti IP, coba lagi dengan --resume")
                    break
            except Exception as e:
                mark_failed(name, str(e))
                log.exception(f"FAILED {name}: {e}")

    log.info(f"Summary: {progress_summary()}")


def main() -> None:
    parser = argparse.ArgumentParser(description="Bandung Barbershop Scraper")
    parser.add_argument("--kelurahan", help="Filter ke kelurahan tertentu (substring match)")
    parser.add_argument("--resume", action="store_true", help="Skip kelurahan yang sudah selesai")
    parser.add_argument("--dry-run", action="store_true", help="Cuma list kelurahan, tanpa scrape")
    parser.add_argument(
        "--limit",
        type=int,
        default=None,
        help="Maks jumlah barbershop yang di-scrape per kelurahan (default: tidak terbatas).",
    )
    parser.add_argument(
        "--auto-sync",
        action="store_true",
        help="Setelah tiap kelurahan selesai scrape, auto-POST ke API backend.",
    )
    args = parser.parse_args()

    if args.dry_run:
        items = filter_kelurahan(args.kelurahan)
        if args.resume:
            init_db()
            items = [k for k in items if not is_done(k["kelurahan"])]
        for k in items:
            print(f"  - {k['kelurahan']} ({k['kecamatan']})")
        print(f"\nTotal: {len(items)} kelurahan")
        return

    try:
        asyncio.run(run(args.kelurahan, args.resume, args.limit, args.auto_sync))
    except KeyboardInterrupt:
        log.warning("Interrupted by user. Progress tersimpan, lanjut dengan --resume")
        sys.exit(130)


if __name__ == "__main__":
    main()
