package scraper

import (
	"github.com/playwright-community/playwright-go"
)

// priceLevelJS pulls a price indicator either from an aria-label containing
// "harga/price" or from a regex against the main panel ("$$$" / "Rp 50.000-100.000").
// Mirrors server/src/gmaps.py::_scrape_price_level.
const priceLevelJS = `() => {
  const norm = s => (s || '').replace(/\s+/g, ' ').trim();
  const candidates = document.querySelectorAll('span[aria-label]');
  for (const el of candidates) {
    const aria = (el.getAttribute('aria-label') || '').toLowerCase();
    if (aria.includes('harga') || aria.includes('price')) {
      const txt = norm(el.innerText) || el.getAttribute('aria-label');
      if (txt) return norm(txt);
    }
  }
  const main = document.querySelector('div[role="main"]') || document.body;
  const headerText = main.innerText || '';
  const m = headerText.match(/\$+|Rp\s*[\d.]+(?:[-–]\s*Rp?\s*[\d.]+)?/);
  if (m) return m[0].trim();
  return null;
}`

// ScrapePriceRange returns the place's price indicator ("Rp 25.000-50.000",
// "$$", "$$$") or "" when not detectable.
func ScrapePriceRange(page playwright.Page) string {
	v, err := page.Evaluate(priceLevelJS)
	if err != nil || v == nil {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return CleanText(s)
}
