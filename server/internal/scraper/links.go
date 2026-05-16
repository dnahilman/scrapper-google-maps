package scraper

import (
	"strings"

	"github.com/playwright-community/playwright-go"
)

// ReviewsLink returns the URL that opens the reviews tab for this place.
// Google Maps doesn't expose a separate URL for the tab; we synthesize one by
// appending the `reviews` parameter so anyone clicking it lands on the tab.
//
// If the page already exposes a direct reviews href on the tab button (rare),
// we prefer that.
func ReviewsLink(page playwright.Page, placeURL string) string {
	loc := page.Locator(`button[aria-label*="ulasan" i], button[aria-label*="review" i]`).First()
	if n, _ := loc.Count(); n > 0 {
		href, _ := loc.GetAttribute("href")
		if href != "" {
			return href
		}
	}
	if placeURL == "" {
		return ""
	}
	if strings.Contains(placeURL, "reviews") {
		return placeURL
	}
	return addQueryParam(placeURL, "reviews", "1")
}

// TimezoneForCity returns the IANA timezone string for a given city name.
// Indonesia spans WIB / WITA / WIT — we cover the most common cases and
// default to Asia/Jakarta for unknown cities.
func TimezoneForCity(cityName string) string {
	c := strings.ToLower(cityName)
	switch {
	case strings.Contains(c, "denpasar"),
		strings.Contains(c, "makassar"),
		strings.Contains(c, "manado"),
		strings.Contains(c, "balikpapan"),
		strings.Contains(c, "samarinda"),
		strings.Contains(c, "mataram"),
		strings.Contains(c, "kupang"):
		return "Asia/Makassar"
	case strings.Contains(c, "jayapura"),
		strings.Contains(c, "ambon"),
		strings.Contains(c, "ternate"),
		strings.Contains(c, "manokwari"),
		strings.Contains(c, "sorong"):
		return "Asia/Jayapura"
	default:
		return "Asia/Jakarta"
	}
}
