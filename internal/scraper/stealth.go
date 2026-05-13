package scraper

// stealthScript patches navigator/window properties Google uses to fingerprint
// headless browsers. Identical to server/src/browser.py::_apply_stealth.
const stealthScript = `
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
`

// userAgentPool is a small curated set of Chrome/Edge desktop UAs.
// We rotate at session start instead of using a heavyweight `fake-useragent`-style lib.
var userAgentPool = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36 Edg/128.0.0.0",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36 Edg/131.0.0.0",
}

type viewport struct {
	Width  int
	Height int
}

var viewportPool = []viewport{
	{1366, 768},
	{1440, 900},
	{1536, 864},
	{1920, 1080},
}

var localePool = []string{"id-ID", "en-US"}
