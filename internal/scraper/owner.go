package scraper

import (
	"github.com/playwright-community/playwright-go"

	"github.com/dnahilman/scrapper-go/internal/domain"
)

// ownerJS makes a best-effort attempt to find Business Profile owner info.
// Google Maps rarely surfaces this in the public panel — when present, it
// usually appears as a link with `data-item-id="claimed-listing-link"` or
// similar. For unclaimed listings we expect to return nil.
const ownerJS = `() => {
  // 1. Explicit "Profil bisnis pemilik" / "Business profile" link with a name.
  const ownerLink = document.querySelector('a[href*="/maps/contrib/"]');
  if (ownerLink) {
    const name = (ownerLink.innerText || ownerLink.getAttribute('aria-label') || '').trim();
    const href = ownerLink.getAttribute('href') || '';
    const idMatch = href.match(/\/maps\/contrib\/(\d+)/);
    return {
      id: idMatch ? idMatch[1] : '',
      name: name,
      link: href.startsWith('http') ? href : ('https://www.google.com' + href),
    };
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
	return &domain.Owner{
		ID:   asString(m["id"]),
		Name: name,
		Link: asString(m["link"]),
	}
}
