package scraper

import (
	"context"
	"strings"

	"github.com/playwright-community/playwright-go"

	"github.com/dnahilman/scrapper-go/internal/domain"
)

// ScrapePlace navigates to a place URL and extracts the core gosom-style
// PlacePayload. Returns nil if the detail panel never loaded.
func ScrapePlace(ctx context.Context, page playwright.Page, rawURL string, minDelay, maxDelay int) (*domain.PlacePayload, error) {
	target := addQueryParam(rawURL, "hl", "id")
	if _, err := page.Goto(target, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		Timeout:   playwright.Float(60000),
	}); err != nil {
		return nil, err
	}
	HumanDelay(ctx, minDelay, maxDelay, false)
	if err := CheckCaptcha(page); err != nil {
		return nil, err
	}

	if err := waitForAny(page, SelTitle, 15000); err != nil {
		return nil, nil
	}

	p := &domain.PlacePayload{
		Link:    target,
		PlaceID: ExtractPlaceID(rawURL),
		Title:   safeText(page, SelTitle),
	}

	// Rating + count.
	if rating, ok := ParseRating(safeText(page, SelRatingText)); ok {
		p.ReviewRating = rating
	}
	rcAria := safeAttr(page, SelReviewCountIDAria, "aria-label")
	if rcAria == "" {
		rcAria = safeAttr(page, SelReviewCountEnAria, "aria-label")
	}
	if rcAria == "" {
		rcAria = safeText(page, SelReviewCountAria)
	}
	if n, ok := ParseReviewCount(rcAria); ok {
		p.ReviewCount = n
	}

	p.Address = safeText(page, SelAddressBtn)
	p.Phone = safeText(page, SelPhoneBtn)
	p.WebSite = safeAttr(page, SelWebsiteLink, "href")
	p.PlusCode = safeText(page, SelPlusCodeBtn)
	p.Category = safeText(page, SelCategoryBtn)

	if lat, lng, ok := ExtractCoords(rawURL); ok {
		p.Latitude = lat
		p.Longtitude = lng
	}

	p.Status = scrapeStatus(page)
	return p, nil
}

// scrapeStatus reads the main panel's innerText and matches known closure phrases.
func scrapeStatus(page playwright.Page) string {
	const js = `() => {
		const main = document.querySelector('div[role="main"]') || document.body;
		const text = (main.innerText || '').toLowerCase();
		if (text.includes('permanen ditutup') || text.includes('tutup permanen') ||
			text.includes('permanently closed')) return 'permanently_closed';
		if (text.includes('tutup sementara') || text.includes('ditutup sementara') ||
			text.includes('temporarily closed')) return 'temporarily_closed';
		return 'active';
	}`
	v, err := page.Evaluate(js)
	if err != nil {
		return "active"
	}
	s, _ := v.(string)
	s = strings.TrimSpace(s)
	if s == "" {
		return "active"
	}
	return s
}
