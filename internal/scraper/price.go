package scraper

import (
	"github.com/playwright-community/playwright-go"
)

// priceJS returns the raw price indicator as Google rendered it — no regex,
// no normalization beyond whitespace collapsing. Looks for any span whose
// aria-label mentions "harga" or "price" and returns the aria-label itself
// (most informative) falling back to inner text. Returns null when absent.
const priceJS = `() => {
  const norm = s => (s || '').replace(/\s+/g, ' ').trim();
  const candidates = document.querySelectorAll('span[aria-label], div[aria-label], button[aria-label]');
  for (const el of candidates) {
    const aria = (el.getAttribute('aria-label') || '');
    const ariaLow = aria.toLowerCase();
    if (ariaLow.includes('harga') || ariaLow.includes('price')) {
      return norm(aria) || norm(el.innerText);
    }
  }
  return null;
}`

// ScrapePrice returns the place's price indicator exactly as Google rendered
// it (e.g. "Rp 25.000–50.000", "Price: $$", "Harga: Rp10.000–25.000"), or ""
// when not detectable. No transformation is applied — consumers can parse as
// needed.
func ScrapePrice(page playwright.Page) string {
	v, err := page.Evaluate(priceJS)
	if err != nil || v == nil {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return CleanText(s)
}
