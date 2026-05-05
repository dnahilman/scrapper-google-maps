"""Entry point CLI untuk scraping Google Maps per kelurahan, keyword configurable.

Keyword di-set via --keyword CLI flag (default: cafe). Output disimpan di
data/{keyword}/{kelurahan}.json, progress.db terpisah per keyword.

Contoh:
    python scripts/scraper.py --keyword cafe --resume
    python scripts/scraper.py --keyword barbershop --kelurahan "Cihapit"
"""
import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).parent.parent))

import argparse
import asyncio
from tqdm.asyncio import tqdm
import config
from config import MAX_CAPTCHA_RETRY, MAX_NETWORK_ERRORS
from src.browser import new_browser_session
from src.gmaps import search_places, scrape_place, CaptchaDetected
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
        urls = await search_places(page, name, kec, limit=limit)
        shops: list[dict] = []
        net_errors = 0
        for i, url in enumerate(urls, 1):
            try:
                data = await scrape_place(page, url)
                if not data:
                    log.info(f"  [{i}/{len(urls)}] SKIP")
                    continue
                log.info(f"  [{i}/{len(urls)}] {data.get('name')}")
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


async def run(
    kelurahan_filter: str | None,
    resume: bool,
    limit: int | None,
    auto_sync: bool = False,
    shard: str | None = None,
) -> None:
    init_db()
    items = filter_kelurahan(kelurahan_filter)
    if resume:
        items = [k for k in items if not is_done(k["kelurahan"])]
    if shard:
        n, m = (int(x) for x in shard.split("/"))
        if not (1 <= n <= m):
            raise ValueError(f"Shard {shard} invalid: butuh 1 <= K <= N")
        items = items[(n - 1) :: m]
    log.info(
        f"[keyword={config.get_keyword()}] Akan scrape {len(items)} kelurahan"
        + (f" (limit {limit} place/kelurahan)" if limit else "")
        + (f" [SHARD {shard}]" if shard else "")
        + (" [AUTO-SYNC]" if auto_sync else "")
        + f" → output: {config.output_dir()}"
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
                log.info(f"DONE {name}: {count} {config.get_keyword()}")

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
    parser = argparse.ArgumentParser(
        description="Google Maps Scraper (multi-keyword via --keyword)"
    )
    parser.add_argument(
        "--keyword",
        default="cafe",
        help="Target keyword scraping (mis. cafe, barbershop, kuliner). Default: cafe",
    )
    parser.add_argument("--kelurahan", help="Filter ke kelurahan tertentu (substring match)")
    parser.add_argument("--resume", action="store_true", help="Skip kelurahan yang sudah selesai")
    parser.add_argument("--dry-run", action="store_true", help="Cuma list kelurahan, tanpa scrape")
    parser.add_argument(
        "--limit",
        type=int,
        default=None,
        help="Maks jumlah place yang di-scrape per kelurahan (default: tidak terbatas).",
    )
    parser.add_argument(
        "--auto-sync",
        action="store_true",
        help="Setelah tiap kelurahan selesai scrape, auto-POST ke API backend.",
    )
    parser.add_argument(
        "--shard",
        default=None,
        help="Bagi kelurahan ke N shard, jalankan shard ke-K. Format: K/N (mis. 1/5). "
             "Round-robin: shard ambil setiap kelurahan ke-N mulai index K-1.",
    )
    args = parser.parse_args()
    config.set_keyword(args.keyword)

    if args.dry_run:
        items = filter_kelurahan(args.kelurahan)
        if args.resume:
            init_db()
            items = [k for k in items if not is_done(k["kelurahan"])]
        if args.shard:
            n, m = (int(x) for x in args.shard.split("/"))
            items = items[(n - 1) :: m]
        for k in items:
            print(f"  - {k['kelurahan']} ({k['kecamatan']})")
        print(f"\nTotal: {len(items)} kelurahan")
        return

    try:
        asyncio.run(run(args.kelurahan, args.resume, args.limit, args.auto_sync, args.shard))
    except KeyboardInterrupt:
        log.warning("Interrupted by user. Progress tersimpan, lanjut dengan --resume")
        sys.exit(130)


if __name__ == "__main__":
    main()
