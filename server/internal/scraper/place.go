package scraper

import (
	"context"
	"strings"

	"github.com/playwright-community/playwright-go"

	"github.com/dnahilman/scrapper-go/internal/domain"
)

// PlaceOptions controls optional/expensive extraction (reviews etc).
type PlaceOptions struct {
	MaxReviewsPerPlace  int
	MinReviewAgeDays    int // skip reviews newer than this
	MaxReviewAgeDays    int // skip reviews older than this
	SortReviewsByNewest bool
	SkipEmptyReviews    bool
	// Task context used to derive PlacePayload.Timezone and CompleteAddress.
	CityName      string
	KelurahanName string
	KecamatanName string
	// EnableEmailCrawl turns on the post-scrape website fetch for emails.
	EnableEmailCrawl bool
}

// ScrapePlace navigates to a place URL and extracts the gosom-style PlacePayload.
// Returns nil if the detail panel never loaded.
func ScrapePlace(ctx context.Context, page playwright.Page, rawURL string, minDelay, maxDelay int, opts PlaceOptions) (*domain.PlacePayload, error) {
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
		DataID:  ExtractDataID(rawURL),
		Cid:     ExtractCID(rawURL),
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

	if ca := ParseCompleteAddress(p.Address, opts.KelurahanName, opts.KecamatanName, opts.CityName); ca != nil {
		p.CompleteAddress = ca
	}

	if lat, lng, ok := ExtractCoords(rawURL); ok {
		p.Latitude = lat
		p.Longtitude = lng
	}

	p.Status = scrapeStatus(page)

	// Small metadata scrapes that live on the default panel.
	p.Price = ScrapePrice(page)
	p.Description = ScrapeDescription(page)
	p.ReviewsLink = ReviewsLink(page, target)
	p.Timezone = TimezoneForCity(opts.CityName)

	// External booking + delivery links (still on the default panel).
	if res, ord := ScrapeExternalLinks(page); len(res) > 0 || len(ord) > 0 {
		p.Reservations = res
		p.OrderOnline = ord
	}

	// Default-panel scrapes — gallery / popular_times / owner / hours all live
	// on the place's first panel, so capture them before any tab switch.
	if thumb, gallery := ScrapeImages(page, 20); len(gallery) > 0 {
		p.Thumbnail = thumb
		p.Images = gallery
	}
	if pt := ScrapePopularTimes(page); len(pt) > 0 {
		p.PopularTimes = pt
	}
	if owner := ScrapeOwner(page); owner != nil {
		p.Owner = owner
	}
	if hours := ScrapeHours(ctx, page); len(hours) > 0 {
		p.OpenHours = hours
	}

	// Reviews tab — also unlocks the per-rating histogram aria-labels.
	if opts.MaxReviewsPerPlace != 0 {
		reviews := ScrapeReviews(ctx, page, ReviewsOptions{
			Max:          opts.MaxReviewsPerPlace,
			MinAgeDays:   opts.MinReviewAgeDays,
			MaxAgeDays:   opts.MaxReviewAgeDays,
			SortByNewest: opts.SortReviewsByNewest,
			SkipEmpty:    opts.SkipEmptyReviews,
		})
		p.UserReviews = reviews
		if dist := ScrapeReviewsPerRating(ctx, page); len(dist) > 0 {
			p.ReviewsPerRating = dist
		}
	}

	// About tab (amenities, payment, accessibility) — switching tabs is OK
	// because we've already captured everything from the default panel above.
	if about := ScrapeAbout(ctx, page); len(about) > 0 {
		p.About = about
	}

	// Menu tab (cafe / resto). Returns nil for places without a Menu tab.
	if menu := ScrapeMenu(ctx, page); menu != nil {
		p.Menu = menu
	}

	// Email crawl is opt-in because it adds an extra outbound HTTP fetch per
	// place — the work happens off the Playwright page so it doesn't lock up
	// the browser context.
	if opts.EnableEmailCrawl && p.WebSite != "" {
		if emails := CrawlEmails(ctx, p.WebSite); len(emails) > 0 {
			p.Emails = emails
		}
	}

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
