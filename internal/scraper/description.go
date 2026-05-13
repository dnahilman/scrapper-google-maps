package scraper

import (
	"github.com/playwright-community/playwright-go"
)

// descriptionJS hunts for an editorial summary Google sometimes renders right
// under the place title. We probe a few stable hooks first (data-attrid /
// jsname markers) and fall back to a heuristic paragraph search bounded by
// length so we don't accidentally swallow address / phone blocks.
const descriptionJS = `() => {
  const norm = s => (s || '').replace(/\s+/g, ' ').trim();
  const main = document.querySelector('div[role="main"]') || document.body;
  // 1. Knowledge-graph style overview / description hooks.
  const hooks = [
    '[data-attrid="kj:overview"]',
    '[data-attrid*="description"]',
    'div.PYvSYb',
    'div[jsname="bN97Pc"]',
  ];
  for (const sel of hooks) {
    const el = main.querySelector(sel);
    if (el) {
      const txt = norm(el.innerText || '');
      if (txt.length >= 30 && txt.length <= 800) return txt;
    }
  }
  return null;
}`

// ScrapeDescription returns the editorial summary if Google exposes one,
// or "" when there is no clear description block (very common).
func ScrapeDescription(page playwright.Page) string {
	v, err := page.Evaluate(descriptionJS)
	if err != nil || v == nil {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return CleanText(s)
}
