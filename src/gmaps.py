"""Google Maps scraping logic — search, place detail, reviews, hours, about."""
import asyncio
import random
import re
from urllib.parse import quote
from playwright.async_api import Page, Locator, TimeoutError as PWTimeout

from config import (
    GMAPS_BASE_URL,
    SEARCH_QUERY_TEMPLATE,
    MIN_DELAY_SEC,
    MAX_DELAY_SEC,
    MAX_REVIEWS_PER_SHOP,
    MAX_REVIEW_AGE_DAYS,
    SKIP_EMPTY_REVIEWS,
    SORT_REVIEWS_BY_NEWEST,
)
from src.logger import get_logger

log = get_logger("gmaps")


class CaptchaDetected(Exception):
    pass


# ============================================================================
# Helpers: delay, captcha detection, text cleaning, parsing
# ============================================================================

async def _human_delay(short: bool = False) -> None:
    lo, hi = (1.0, 2.5) if short else (MIN_DELAY_SEC, MAX_DELAY_SEC)
    await asyncio.sleep(random.uniform(lo, hi))


async def _check_captcha(page: Page) -> None:
    content = (await page.content()).lower()
    triggers = ("unusual traffic", "captcha", "/sorry/", "verify you're human")
    if any(t in content for t in triggers):
        raise CaptchaDetected("Google curiga — CAPTCHA / unusual traffic terdeteksi")


def _clean(s: str | None) -> str | None:
    """Strip whitespace + Material Symbols icon chars (Private Use Area Unicode)."""
    if not s:
        return None
    no_icons = "".join(c for c in s if not ("" <= c <= ""))
    cleaned = " ".join(no_icons.split())
    return cleaned or None


def _parse_rating(s: str | None) -> float | None:
    if not s:
        return None
    m = re.search(r"(\d+[.,]\d+|\d+)", s)
    return float(m.group(1).replace(",", ".")) if m else None


def _parse_review_count(s: str | None) -> int | None:
    if not s:
        return None
    m = re.search(r"([\d.,]+)", s)
    if not m:
        return None
    return int(m.group(1).replace(".", "").replace(",", ""))


def _extract_place_id(url: str) -> str | None:
    m = re.search(r"!1s([^!]+)", url) or re.search(r"placeid=([^&]+)", url)
    return m.group(1) if m else None


def _extract_coords(url: str) -> tuple[float | None, float | None]:
    m = re.search(r"!3d(-?\d+\.\d+)!4d(-?\d+\.\d+)", url) or re.search(
        r"@(-?\d+\.\d+),(-?\d+\.\d+)", url
    )
    if m:
        return float(m.group(1)), float(m.group(2))
    return None, None


_AGE_UNITS = {
    "tahun": 365, "year": 365,
    "bulan": 30, "month": 30,
    "minggu": 7, "week": 7,
    "hari": 1, "day": 1,
    "jam": 1 / 24, "hour": 1 / 24,
    "menit": 1 / (24 * 60), "minute": 1 / (24 * 60),
    "detik": 1 / (24 * 3600), "second": 1 / (24 * 3600),
}


def _parse_age_days(time_str: str | None) -> int | None:
    """Parse 'X tahun lalu' / 'X years ago' → jumlah hari."""
    if not time_str:
        return None
    s = time_str.lower().strip()
    s = re.sub(r"^diedit\s+", "", s)
    if "kemarin" in s or "yesterday" in s:
        return 1
    n: int | None = None
    if re.match(r"^se(tahun|bulan|minggu|hari|jam|menit|detik)\b", s):
        n = 1
    elif re.match(r"^(a|an)\s+(year|month|week|day|hour|minute|second)\b", s):
        n = 1
    else:
        m = re.match(r"^(\d+)", s)
        if m:
            n = int(m.group(1))
    if n is None:
        return None
    for unit, mult in _AGE_UNITS.items():
        if unit in s:
            return int(n * mult)
    return None


def _is_review_text_empty(text: str | None) -> bool:
    if not text:
        return True
    return len(text.strip()) < 3


# ============================================================================
# DOM helpers
# ============================================================================

async def _safe_text(page: Page | Locator, selector: str) -> str | None:
    try:
        loc = page.locator(selector).first
        if await loc.count() == 0:
            return None
        return await loc.inner_text(timeout=3000)
    except Exception:
        return None


async def _safe_attr(page: Page | Locator, selector: str, attr: str) -> str | None:
    try:
        loc = page.locator(selector).first
        if await loc.count() == 0:
            return None
        return await loc.get_attribute(attr, timeout=3000)
    except Exception:
        return None


async def _click_tab(page: Page, *aria_patterns: str) -> bool:
    """Klik tab/button by aria-label (case-insensitive). Coba beberapa pattern."""
    for pattern in aria_patterns:
        for sel in (
            f'button[role="tab"][aria-label*="{pattern}" i]',
            f'button[aria-label*="{pattern}" i]',
        ):
            loc = page.locator(sel).first
            if await loc.count() > 0:
                try:
                    await loc.scroll_into_view_if_needed(timeout=2000)
                    await loc.click(timeout=3000)
                    await _human_delay(short=True)
                    return True
                except Exception:
                    continue
    return False


# ============================================================================
# Search: cari semua barbershop di kelurahan
# ============================================================================

async def search_barbershops(
    page: Page, kelurahan: str, kecamatan: str, limit: int | None = None
) -> list[str]:
    query = SEARCH_QUERY_TEMPLATE.format(kelurahan=kelurahan, kecamatan=kecamatan)
    url = f"{GMAPS_BASE_URL}/search/{quote(query)}?hl=id"
    log.info(f"Search: {query}")
    await page.goto(url, wait_until="domcontentloaded", timeout=60000)
    await _human_delay()
    await _check_captcha(page)

    try:
        await page.wait_for_selector('div[role="feed"], a[href*="/maps/place/"]', timeout=20000)
    except PWTimeout:
        log.warning(f"Tidak ada hasil untuk {kelurahan}")
        return []

    feed = page.locator('div[role="feed"]').first
    has_feed = await feed.count() > 0
    prev_count = 0
    stuck = 0
    while stuck < 3:
        if has_feed:
            await feed.evaluate("el => el.scrollTo(0, el.scrollHeight)")
        await asyncio.sleep(random.uniform(2, 4))
        cards = await page.locator('a[href*="/maps/place/"]').all()
        if limit and len(cards) >= limit:
            break
        if len(cards) == prev_count:
            stuck += 1
        else:
            stuck = 0
        prev_count = len(cards)
        body_text = (await page.content()).lower()
        if "you've reached the end" in body_text or "anda telah mencapai akhir" in body_text:
            break

    urls: list[str] = []
    for card in await page.locator('a[href*="/maps/place/"]').all():
        href = await card.get_attribute("href")
        if href and href not in urls:
            urls.append(href)
        if limit and len(urls) >= limit:
            break
    log.info(f"Ditemukan {len(urls)} barbershop di {kelurahan}" + (f" (limit {limit})" if limit else ""))
    return urls[:limit] if limit else urls


# ============================================================================
# Place detail: nama, alamat, rating, hours, about, reviews
# ============================================================================

async def scrape_place(page: Page, url: str) -> dict | None:
    if "hl=" not in url:
        sep = "&" if "?" in url else "?"
        url = f"{url}{sep}hl=id"
    await page.goto(url, wait_until="domcontentloaded", timeout=60000)
    await _human_delay()
    await _check_captcha(page)

    try:
        await page.wait_for_selector("h1", timeout=15000)
    except PWTimeout:
        log.warning(f"Halaman detail gagal load: {url}")
        return None

    data: dict = {"url": url}
    data["place_id"] = _extract_place_id(url)
    data["name"] = _clean(await _safe_text(page, "h1"))

    rating_raw = _clean(await _safe_text(page, 'div.F7nice span[aria-hidden="true"]'))
    data["rating"] = _parse_rating(rating_raw)
    data["rating_raw"] = rating_raw

    rc_raw = _clean(
        await _safe_attr(page, 'div.F7nice span[aria-label*="ulasan" i]', "aria-label")
        or await _safe_attr(page, 'div.F7nice span[aria-label*="review" i]', "aria-label")
        or await _safe_text(page, 'div.F7nice span[aria-label]')
    )
    data["review_count"] = _parse_review_count(rc_raw)

    data["address"] = _clean(await _safe_text(page, 'button[data-item-id="address"]'))
    data["phone"] = _clean(await _safe_text(page, 'button[data-item-id^="phone"]'))
    data["website"] = await _safe_attr(page, 'a[data-item-id="authority"]', "href")
    data["plus_code"] = _clean(await _safe_text(page, 'button[data-item-id="oloc"]'))
    data["category"] = _clean(await _safe_text(page, 'button[jsaction*="category"]'))
    data["lat"], data["lng"] = _extract_coords(url)

    data["status"] = await _scrape_status(page)
    data["price_level"] = await _scrape_price_level(page)
    data["photos"] = await _scrape_photos(page)
    data["hours"] = await _scrape_hours(page)
    data["about"] = _clean_about(await _scrape_about(page))
    data["services_about"] = _flatten_services(data["about"])
    data["reviews"] = await _scrape_reviews(page)
    return data


# ============================================================================
# Status (active / temporarily_closed / permanently_closed)
# ============================================================================

async def _scrape_status(page: Page) -> str:
    try:
        result = await page.evaluate(
            r"""
            () => {
                const main = document.querySelector('div[role="main"]') || document.body;
                const text = (main.innerText || '').toLowerCase();
                if (text.includes('permanen ditutup') || text.includes('tutup permanen') ||
                    text.includes('permanently closed')) return 'permanently_closed';
                if (text.includes('tutup sementara') || text.includes('ditutup sementara') ||
                    text.includes('temporarily closed')) return 'temporarily_closed';
                return 'active';
            }
            """
        )
        return result or "active"
    except Exception:
        return "active"


# ============================================================================
# Price level
# ============================================================================

async def _scrape_price_level(page: Page) -> str | None:
    try:
        result = await page.evaluate(
            r"""
            () => {
                const norm = s => (s || '').replace(/\s+/g, ' ').trim();
                // Coba aria-label yang berkaitan dengan harga
                const candidates = document.querySelectorAll('span[aria-label]');
                for (const el of candidates) {
                    const aria = (el.getAttribute('aria-label') || '').toLowerCase();
                    if (aria.includes('harga') || aria.includes('price')) {
                        const txt = norm(el.innerText) || el.getAttribute('aria-label');
                        if (txt) return norm(txt);
                    }
                }
                // Cari simbol $ atau "Rp NNN-NNN" di header place
                const main = document.querySelector('div[role="main"]') || document.body;
                const headerText = main.innerText || '';
                const m = headerText.match(/\$+|Rp\s*[\d.]+(?:[-–]\s*Rp?\s*[\d.]+)?/);
                if (m) return m[0].trim();
                return null;
            }
            """
        )
        return _clean(result) if result else None
    except Exception:
        return None


# ============================================================================
# Photos (place-level cover/gallery photos)
# ============================================================================

def _is_avatar_url(url: str) -> bool:
    """Avatar URL pattern: googleusercontent.com/a/... atau /a-/..."""
    import re as _re
    return bool(_re.search(r"googleusercontent\.com/a[-]?/", url))


async def _scrape_photos(page: Page, max_photos: int = 20) -> list[str]:
    """Extract photo URLs dari panel detail place. Normalize URL ke high-res, skip avatar."""
    try:
        urls = await page.evaluate(
            r"""
            (maxN) => {
                const isAvatar = url => /googleusercontent\.com\/a[-]?\//.test(url);
                const main = document.querySelector('div[role="main"]') || document.body;
                const out = new Set();
                main.querySelectorAll('img').forEach(img => {
                    const src = img.src || img.dataset?.src || '';
                    if (!src.includes('googleusercontent.com')) return;
                    if (isAvatar(src)) return;
                    if (/=s[1-9]\d?-/.test(src)) return;
                    if (/=w[1-9]\d?-h/.test(src)) return;
                    let base = src;
                    const eqIdx = src.indexOf('=');
                    if (eqIdx > -1) base = src.substring(0, eqIdx);
                    out.add(base + '=w800-h600-k-no');
                });
                main.querySelectorAll('button[style*="background-image"], div[style*="background-image"]').forEach(el => {
                    const style = el.getAttribute('style') || '';
                    const m = style.match(/url\("?(https:\/\/[^")]+googleusercontent[^")]+)"?\)/);
                    if (m) {
                        let url = m[1];
                        if (isAvatar(url)) return;
                        const eqIdx = url.indexOf('=');
                        if (eqIdx > -1) url = url.substring(0, eqIdx);
                        out.add(url + '=w800-h600-k-no');
                    }
                });
                return Array.from(out).slice(0, maxN);
            }
            """,
            max_photos,
        )
        return urls or []
    except Exception as e:
        log.debug(f"Photos scrape error: {e}")
        return []


# ============================================================================
# Hours
# ============================================================================

async def _scrape_hours(page: Page) -> dict:
    """Hours bisa muncul tanpa expand. Kalau perlu expand, klik button-nya."""
    try:
        toggle = page.locator(
            'button[data-item-id*="oh"], button[aria-label*="jam" i], button[aria-label*="hours" i]'
        ).first
        if await toggle.count() > 0:
            try:
                await toggle.click(timeout=2000)
                await _human_delay(short=True)
            except Exception:
                pass

        hours = await page.evaluate(
            r"""
            () => {
                const out = {};
                const tables = document.querySelectorAll('table');
                for (const t of tables) {
                    const rows = t.querySelectorAll('tr');
                    if (rows.length < 5 || rows.length > 8) continue;
                    const tmp = {};
                    rows.forEach(r => {
                        const cells = r.querySelectorAll('td, th');
                        if (cells.length >= 2) {
                            const day = cells[0].innerText.trim();
                            const time = cells[1].innerText.replace(/\n+/g, ' ').trim();
                            if (day && time) tmp[day] = time;
                        }
                    });
                    if (Object.keys(tmp).length >= 5) return tmp;
                }
                return out;
            }
            """
        )
        return hours or {}
    except Exception as e:
        log.debug(f"Hours scrape error: {e}")
        return {}


# ============================================================================
# About panel (amenities, payment options, accessibility, etc)
# ============================================================================

async def _scrape_about(page: Page) -> dict:
    if not await _click_tab(page, "Tentang", "About"):
        return {}
    try:
        await page.wait_for_selector('h2, [role="region"]', timeout=5000)
    except PWTimeout:
        return {}

    sections = await page.evaluate(
        r"""
        () => {
            const norm = s => (s || '').replace(/\s+/g, ' ').trim();
            const main = document.querySelector('div[role="main"]') || document.body;
            const h2s = Array.from(main.querySelectorAll('h2')).filter(h => h.innerText.trim());
            if (h2s.length === 0) return {};
            const container = h2s[0].closest('div[role="region"]') || main;

            const result = {};
            let currentSection = null;
            const items = new Map();

            const walker = document.createTreeWalker(container, NodeFilter.SHOW_ELEMENT);
            const flush = () => {
                if (currentSection && items.size > 0) {
                    result[currentSection] = Array.from(items.keys());
                }
                items.clear();
            };

            let node;
            while ((node = walker.nextNode())) {
                if (node.tagName === 'H2') {
                    const name = norm(node.innerText);
                    if (name) { flush(); currentSection = name; }
                } else if (currentSection && node.tagName === 'LI') {
                    const aria = node.getAttribute('aria-label');
                    const txt = aria || node.innerText || '';
                    const value = norm(txt);
                    if (value && value.length < 200 && value !== currentSection) {
                        items.set(value, true);
                    }
                }
            }
            flush();
            return result;
        }
        """
    )
    return sections or {}


def _flatten_services(about: dict) -> list[str]:
    """Ekstrak items dari section bernama 'Layanan'/'Services' di About panel."""
    if not about:
        return []
    services: list[str] = []
    for k, items in about.items():
        kl = k.lower()
        if kl == "layanan" or kl == "services" or "service options" in kl or "opsi layanan" in kl:
            for item in items:
                clean = _clean(item)
                if clean and clean not in services:
                    services.append(clean)
    return services


def _clean_about(about: dict) -> dict:
    out: dict[str, list[str]] = {}
    for section, items in about.items():
        sec = _clean(section) or section
        cleaned_items: list[str] = []
        for it in items:
            c = _clean(it)
            if c and c not in cleaned_items:
                cleaned_items.append(c)
        if cleaned_items:
            out[sec] = cleaned_items
    return out


# ============================================================================
# Reviews
# ============================================================================

async def _sort_reviews_by_newest(page: Page) -> bool:
    """Klik tombol Sort lalu pilih 'Terbaru' / 'Newest'."""
    sort_btn = page.locator(
        'button[aria-label*="urutkan" i], button[aria-label*="sort" i], button[data-value*="Sort" i]'
    ).first
    if await sort_btn.count() == 0:
        return False
    try:
        await sort_btn.click(timeout=3000)
        await _human_delay(short=True)
    except Exception:
        return False
    for label in ("Terbaru", "Newest"):
        opt = page.locator(
            f'div[role="menuitemradio"]:has-text("{label}"), div[role="menuitem"]:has-text("{label}")'
        ).first
        if await opt.count() > 0:
            try:
                await opt.click(timeout=2000)
                await _human_delay(short=True)
                return True
            except Exception:
                continue
    return False


async def _scrape_reviews(page: Page) -> list[dict]:
    if not await _click_tab(page, "Ulasan", "Reviews"):
        return []

    try:
        await page.wait_for_selector("div[data-review-id]", timeout=10000)
    except PWTimeout:
        return []

    if SORT_REVIEWS_BY_NEWEST:
        await _sort_reviews_by_newest(page)
        await asyncio.sleep(1.5)

    scroll_handle = await page.evaluate_handle(
        r"""
        () => {
            const card = document.querySelector('[data-review-id]');
            if (!card) return null;
            let el = card.parentElement;
            while (el) {
                const cs = getComputedStyle(el);
                if ((cs.overflowY === 'auto' || cs.overflowY === 'scroll') && el.scrollHeight > el.clientHeight) {
                    return el;
                }
                el = el.parentElement;
            }
            return null;
        }
        """
    )

    prev = 0
    stuck = 0
    while True:
        try:
            await scroll_handle.evaluate("el => el && el.scrollTo(0, el.scrollHeight)")
        except Exception:
            await page.mouse.wheel(0, 4000)
        await asyncio.sleep(random.uniform(1.2, 2.5))
        count = await page.locator("div[data-review-id]").count()
        if count >= MAX_REVIEWS_PER_SHOP:
            break

        # Early stop kalau review terakhir sudah lebih tua dari threshold (sorted newest first)
        if SORT_REVIEWS_BY_NEWEST and count > 0:
            last_time = await page.evaluate(
                r"""idx => {
                    const cards = document.querySelectorAll('[data-review-id]');
                    const c = cards[idx];
                    if (!c) return null;
                    const t = c.querySelector('span.rsqaWe, [class*="rsqaWe"]');
                    return t?.innerText?.trim() || null;
                }""",
                count - 1,
            )
            age = _parse_age_days(last_time)
            if age is not None and age > MAX_REVIEW_AGE_DAYS:
                break

        if count == prev:
            stuck += 1
            if stuck >= 3:
                break
        else:
            stuck = 0
        prev = count

    # Expand truncated review text
    for sel in (
        'button:has-text("Lainnya")',
        'button:has-text("Selengkapnya")',
        'button:has-text("More")',
    ):
        for btn in (await page.locator(sel).all())[:MAX_REVIEWS_PER_SHOP]:
            try:
                await btn.click(timeout=800)
            except Exception:
                pass

    cards = await page.locator("div[data-review-id]").all()
    reviews: list[dict] = []
    seen_ids: set[str] = set()
    skipped_empty = 0
    skipped_old = 0
    for card in cards:
        try:
            r = await card.evaluate(
                r"""
                el => {
                    const txt = (sel) => el.querySelector(sel)?.innerText?.trim() || null;
                    const attr = (sel, a) => el.querySelector(sel)?.getAttribute(a) || null;
                    // Review photos (skip avatar profile)
                    const isAvatar = url => /googleusercontent\.com\/a[-]?\//.test(url);
                    const photos = new Set();
                    el.querySelectorAll('button[style*="background-image"], div[style*="background-image"]').forEach(b => {
                        const style = b.getAttribute('style') || '';
                        const m = style.match(/url\("?(https:\/\/[^")]+googleusercontent[^")]+)"?\)/);
                        if (m) {
                            let url = m[1];
                            if (isAvatar(url)) return;
                            const eqIdx = url.indexOf('=');
                            if (eqIdx > -1) url = url.substring(0, eqIdx);
                            photos.add(url + '=w800-h600-k-no');
                        }
                    });
                    el.querySelectorAll('img').forEach(img => {
                        const src = img.src || img.dataset?.src || '';
                        if (!src.includes('googleusercontent.com')) return;
                        if (isAvatar(src)) return;
                        if (/=s[1-9]\d?-/.test(src)) return;
                        let base = src;
                        const eqIdx = src.indexOf('=');
                        if (eqIdx > -1) base = src.substring(0, eqIdx);
                        photos.add(base + '=w800-h600-k-no');
                    });
                    return {
                        review_id: el.getAttribute('data-review-id'),
                        author: txt('div.d4r55') || txt('[class*="d4r55"]') || txt('button[jsaction*="reviewerLink"]'),
                        rating_aria: attr('span[role="img"][aria-label*="bintang" i]', 'aria-label')
                                  || attr('span[role="img"][aria-label*="star" i]', 'aria-label'),
                        text: txt('span.wiI7pd') || txt('[class*="wiI7pd"]'),
                        time: txt('span.rsqaWe') || txt('[class*="rsqaWe"]'),
                        owner_response: txt('div.CDe7pd') || txt('[class*="CDe7pd"]'),
                        review_photos: Array.from(photos),
                    };
                }
                """
            )
            rid = r.get("review_id")
            if not rid or rid in seen_ids:
                continue
            seen_ids.add(rid)

            for k in ("author", "text", "time", "owner_response"):
                r[k] = _clean(r.get(k))
            r["rating"] = _parse_rating(r.get("rating_aria"))
            r["age_days"] = _parse_age_days(r.get("time"))

            if SKIP_EMPTY_REVIEWS and _is_review_text_empty(r.get("text")):
                skipped_empty += 1
                continue
            if r["age_days"] is not None and r["age_days"] > MAX_REVIEW_AGE_DAYS:
                skipped_old += 1
                continue

            reviews.append(r)
            if len(reviews) >= MAX_REVIEWS_PER_SHOP:
                break
        except Exception as e:
            log.debug(f"Review parse error: {e}")

    log.info(
        f"  Reviews: {len(reviews)} kept "
        f"(skip {skipped_empty} kosong, {skipped_old} >{MAX_REVIEW_AGE_DAYS}d)"
    )
    return reviews
