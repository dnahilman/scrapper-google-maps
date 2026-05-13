package scraper

import (
	"github.com/playwright-community/playwright-go"

	"github.com/dnahilman/scrapper-go/internal/domain"
)

// galleryJS collects every googleusercontent.com image URL from the main panel,
// dropping avatars and tiny thumbnails, then normalizing to a fixed
// =w800-h600-k-no sizing. Mirrors server/src/gmaps.py::_scrape_photos.
const galleryJS = `(maxN) => {
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
}`

// ScrapeImages extracts the place's gallery photos. The first URL is treated
// as the cover thumbnail (returned separately so callers can populate both
// PlacePayload.Thumbnail and PlacePayload.Images).
func ScrapeImages(page playwright.Page, maxPhotos int) (thumbnail string, gallery []domain.Image) {
	if maxPhotos <= 0 {
		maxPhotos = 20
	}
	v, err := page.Evaluate(galleryJS, maxPhotos)
	if err != nil || v == nil {
		return "", nil
	}
	urls := asStringSlice(v)
	if len(urls) == 0 {
		return "", nil
	}
	gallery = make([]domain.Image, 0, len(urls))
	for _, u := range urls {
		// Google Maps doesn't expose reliable captions in the panel grid, so
		// the title stays empty. We keep the gosom shape for compatibility.
		gallery = append(gallery, domain.Image{Image: u})
	}
	return urls[0], gallery
}
