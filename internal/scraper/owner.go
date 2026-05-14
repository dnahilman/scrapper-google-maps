package scraper

import (
	"strings"

	"github.com/playwright-community/playwright-go"

	"github.com/dnahilman/scrapper-go/internal/domain"
)

// ownerJS hunts for the *business profile* owner — NOT a random reviewer
// contributor. Google renders the owner link in a few stable spots:
//
//   - a button with data-item-id="merchant" or text "Pemilik bisnis"
//   - an `<a>` under a heading containing "Pemilik bisnis" / "Business owner"
//
// We deliberately ignore /maps/contrib/ links that live inside review cards
// (i.e. inside a [data-review-id] element) because those are reviewers, not
// the business owner. Returns null for unclaimed listings (the common case).
const ownerJS = `() => {
  const norm = s => (s || '').replace(/\s+/g, ' ').trim();

  // 1. data-item-id="merchant" — most reliable when present.
  const merchant = document.querySelector('[data-item-id="merchant"] a, a[data-item-id="merchant"]');
  if (merchant) {
    const name = norm(merchant.innerText || merchant.getAttribute('aria-label') || '');
    const href = merchant.getAttribute('href') || '';
    if (name) {
      const m = href.match(/\/maps\/contrib\/(\d+)/);
      return {
        id: m ? m[1] : '',
        name,
        link: href.startsWith('http') ? href : ('https://www.google.com' + href),
      };
    }
  }

  // 2. Find a heading/aria-label hinting "Pemilik bisnis" / "Business profile"
  //    and pick the first <a> inside the same region.
  const headings = document.querySelectorAll('h2, h3, button, span[aria-label]');
  for (const h of headings) {
    const text = norm(h.innerText || h.getAttribute('aria-label') || '').toLowerCase();
    if (!text) continue;
    if (text.includes('pemilik bisnis') || text.includes('business profile') ||
        text.includes('business owner')) {
      const region = h.closest('div[role="region"], section, div') || h.parentElement;
      if (!region) continue;
      const link = region.querySelector('a[href*="/maps/contrib/"]');
      if (!link) continue;
      // Skip if the link sits inside a review card — that's a reviewer.
      if (link.closest('[data-review-id]')) continue;
      const name = norm(link.innerText || link.getAttribute('aria-label') || '');
      if (!name) continue;
      const href = link.getAttribute('href') || '';
      const m = href.match(/\/maps\/contrib\/(\d+)/);
      return {
        id: m ? m[1] : '',
        name,
        link: href.startsWith('http') ? href : ('https://www.google.com' + href),
      };
    }
  }

  return null;
}`

// ScrapeOwner returns the place owner's Business Profile reference when Google
// exposes one. nil for unclaimed or unverified listings (most places).
func ScrapeOwner(page playwright.Page) *domain.Owner {
	v, err := page.Evaluate(ownerJS)
	if err != nil || v == nil {
		return nil
	}
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	name := CleanText(asString(m["name"]))
	if name == "" {
		return nil
	}
	// Defensive: skip captions starting with "Foto pengulas" / "Reviewer photo"
	// that occasionally leak through when the heading match grabs a reviewer-
	// caption span sitting in the same region.
	low := strings.ToLower(name)
	for _, banned := range []string{"foto pengulas", "reviewer photo", "menulis", "wrote"} {
		if strings.Contains(low, banned) {
			return nil
		}
	}
	return &domain.Owner{
		ID:   asString(m["id"]),
		Name: name,
		Link: asString(m["link"]),
	}
}
