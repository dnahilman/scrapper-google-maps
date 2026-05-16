package scraper

import (
	"context"

	"github.com/playwright-community/playwright-go"

	"github.com/dnahilman/scrapper-go/internal/domain"
)

// menuItemsJS extracts dish-name + price pairs from the Menu tab. Strategy 1
// is heading-based (most reliable when Google renders a structured menu).
// Strategy 2 falls back to leaf elements containing "Rp NNN" patterns.
const menuItemsJS = `() => {
  const norm = s => (s || '').replace(/\s+/g, ' ').trim();
  const main = document.querySelector('div[role="main"]') || document.body;
  const out = [];
  const seen = new Set();
  const priceRe = /Rp\s*[\d.,]+(?:\s*[-–]\s*Rp?\s*[\d.,]+)?/i;
  const addItem = (name, price) => {
    const cleanName = norm(name);
    if (!cleanName || cleanName.length > 150) return;
    const cleanPrice = price ? norm(price) : null;
    const key = cleanName + '|' + (cleanPrice || '');
    if (seen.has(key)) return;
    seen.add(key);
    out.push({ name: cleanName, price: cleanPrice });
  };
  main.querySelectorAll('h3, [role="heading"]').forEach(h => {
    const name = norm(h.innerText);
    if (!name) return;
    const block = h.closest('[role="button"], [role="link"], li, div[jsaction]') || h.parentElement;
    if (!block) return;
    const blockText = norm(block.innerText);
    const m = blockText.match(priceRe);
    addItem(name, m ? m[0] : null);
  });
  if (out.length === 0) {
    main.querySelectorAll('li, div[jsaction]').forEach(el => {
      const text = norm(el.innerText);
      if (!text || text.length > 300) return;
      const m = text.match(priceRe);
      if (!m) return;
      const beforePrice = norm(text.split(m[0])[0]);
      if (!beforePrice) return;
      const lines = beforePrice.split('\n').map(s => s.trim()).filter(Boolean);
      addItem(lines[0], m[0]);
    });
  }
  return out.slice(0, 200);
}`

// menuPhotosJS collects food photos from the main panel, filtering out avatars
// and tiny placeholders. URLs are normalized to a fixed =w800-h600-k-no size.
const menuPhotosJS = `() => {
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
  return Array.from(out).slice(0, 30);
}`

// ScrapeMenu opens the in-place "Menu" tab and extracts items + photos.
// Returns nil when no Menu tab exists (e.g. barbershops).
func ScrapeMenu(ctx context.Context, page playwright.Page) *domain.MenuPayload {
	if !clickTab(ctx, page, "Menu") {
		return nil
	}
	if err := waitForAny(page, `div[role="main"]`, 5000); err != nil {
		return nil
	}
	HumanDelay(ctx, 0, 0, true)

	items := decodeMenuItems(page)
	photos := decodeMenuPhotos(page)
	if len(items) == 0 && len(photos) == 0 {
		return nil
	}
	return &domain.MenuPayload{Items: items, Photos: photos}
}

func decodeMenuItems(page playwright.Page) []domain.MenuItem {
	v, err := page.Evaluate(menuItemsJS)
	if err != nil || v == nil {
		return nil
	}
	xs, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]domain.MenuItem, 0, len(xs))
	for _, x := range xs {
		m, ok := x.(map[string]any)
		if !ok {
			continue
		}
		name := CleanText(asString(m["name"]))
		if name == "" {
			continue
		}
		out = append(out, domain.MenuItem{
			Name:  name,
			Price: CleanText(asString(m["price"])),
		})
	}
	return out
}

func decodeMenuPhotos(page playwright.Page) []string {
	v, err := page.Evaluate(menuPhotosJS)
	if err != nil || v == nil {
		return nil
	}
	return asStringSlice(v)
}
