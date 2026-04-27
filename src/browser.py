"""Playwright browser session dengan stealth + isolated context (no login, no cookie persist)."""
import random
from contextlib import asynccontextmanager
from playwright.async_api import async_playwright, BrowserContext, Browser
from fake_useragent import UserAgent
from config import HEADLESS, VIEWPORTS, LOCALES, TIMEZONE

_ua = UserAgent(browsers=["Chrome", "Edge"], os=["Windows"], min_percentage=1.0)


def _random_user_agent() -> str:
    try:
        return _ua.random
    except Exception:
        return (
            "Mozilla/5.0 (Windows NT 10.0; Win64; x64) "
            "AppleWebKit/537.36 (KHTML, like Gecko) "
            "Chrome/131.0.0.0 Safari/537.36"
        )


@asynccontextmanager
async def new_browser_session():
    """Yield (browser, desktop_context). Browser di-yield supaya caller bisa buat context tambahan kalau perlu."""
    async with async_playwright() as pw:
        browser = await pw.chromium.launch(
            headless=HEADLESS,
            args=[
                "--disable-blink-features=AutomationControlled",
                "--disable-dev-shm-usage",
                "--no-sandbox",
                "--disable-features=IsolateOrigins,site-per-process",
            ],
        )
        context = await _make_desktop_context(browser)
        try:
            yield browser, context
        finally:
            await context.close()
            await browser.close()


async def _make_desktop_context(browser: Browser) -> BrowserContext:
    context = await browser.new_context(
        viewport=random.choice(VIEWPORTS),
        user_agent=_random_user_agent(),
        locale=random.choice(LOCALES),
        timezone_id=TIMEZONE,
        geolocation={"latitude": -6.9175, "longitude": 107.6191},
        permissions=["geolocation"],
    )
    await _apply_stealth(context)
    return context


async def _apply_stealth(context: BrowserContext) -> None:
    """Patch navigator properties yang bocorkan automation."""
    await context.add_init_script(
        """
        Object.defineProperty(navigator, 'webdriver', {get: () => undefined});
        Object.defineProperty(navigator, 'languages', {get: () => ['id-ID', 'id', 'en-US']});
        Object.defineProperty(navigator, 'plugins', {get: () => [1, 2, 3, 4, 5]});
        window.chrome = { runtime: {} };
        const origQuery = window.navigator.permissions.query;
        window.navigator.permissions.query = (p) => (
            p.name === 'notifications'
                ? Promise.resolve({ state: Notification.permission })
                : origQuery(p)
        );
        """
    )
