package scraper

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
)

const (
	gmapsBaseURL         = "https://www.google.com/maps"
	searchQueryTemplate  = "%s di %s, %s, %s"
	searchScrollMaxStuck = 3
)

// Search opens a Google Maps search for `<keyword> di <kelurahan>, <kecamatan>, <city>`
// and scrolls the results feed until no new entries appear, then returns
// every unique place URL it found.
//
// If limit > 0, the scroll loop stops once at least `limit` URLs are collected.
func Search(ctx context.Context, page playwright.Page, keyword, kelurahan, kecamatan, city string, limit int, minDelay, maxDelay int) ([]string, error) {
	query := fmt.Sprintf(searchQueryTemplate, keyword, kelurahan, kecamatan, city)
	searchURL := fmt.Sprintf("%s/search/%s?hl=id", gmapsBaseURL, url.PathEscape(query))

	if _, err := page.Goto(searchURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		Timeout:   playwright.Float(60000),
	}); err != nil {
		return nil, fmt.Errorf("goto search: %w", err)
	}
	HumanDelay(ctx, minDelay, maxDelay, false)
	if err := CheckCaptcha(page); err != nil {
		return nil, err
	}

	// Google auto-redirects exact-match queries (e.g. specific business names)
	// straight to the place detail panel — no result list. Detect this first.
	if final := page.URL(); strings.Contains(final, "/maps/place/") {
		return []string{final}, nil
	}

	if err := waitForAny(page, SelFeedOrCard, 20000); err != nil {
		// One more chance after the soft wait — Google sometimes finishes the
		// redirect after a brief loading delay.
		if final := page.URL(); strings.Contains(final, "/maps/place/") {
			return []string{final}, nil
		}
		return nil, nil
	}

	feed := page.Locator(SelFeed).First()
	feedCount, _ := feed.Count()
	hasFeed := feedCount > 0

	prev := 0
	stuck := 0
	for stuck < searchScrollMaxStuck {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if hasFeed {
			_, _ = feed.Evaluate("el => el.scrollTo(0, el.scrollHeight)", nil)
		}
		HumanDelay(ctx, 2, 4, false)

		cards, _ := page.Locator(SelPlaceCard).All()
		if limit > 0 && len(cards) >= limit {
			break
		}
		if len(cards) == prev {
			stuck++
		} else {
			stuck = 0
		}
		prev = len(cards)

		content, _ := page.Content()
		lower := strings.ToLower(content)
		if strings.Contains(lower, "you've reached the end") ||
			strings.Contains(lower, "anda telah mencapai akhir") {
			break
		}
	}

	cards, err := page.Locator(SelPlaceCard).All()
	if err != nil {
		return nil, fmt.Errorf("collect cards: %w", err)
	}
	seen := make(map[string]struct{}, len(cards))
	urls := make([]string, 0, len(cards))
	for _, card := range cards {
		href, _ := card.GetAttribute("href")
		if href == "" {
			continue
		}
		if _, ok := seen[href]; ok {
			continue
		}
		seen[href] = struct{}{}
		urls = append(urls, href)
		if limit > 0 && len(urls) >= limit {
			break
		}
	}
	return urls, nil
}

// addQueryParam appends `?hl=id` (or `&hl=id`) when missing — mirrors the
// Python implementation so behaviour stays consistent.
func addQueryParam(rawURL, key, value string) string {
	if strings.Contains(rawURL, key+"=") {
		return rawURL
	}
	sep := "?"
	if strings.Contains(rawURL, "?") {
		sep = "&"
	}
	return rawURL + sep + key + "=" + value
}

// Place navigates to a single Google Maps place URL and extracts the
// gosom-style PlacePayload.
//
// Phase 3 MVP: returns the *core* fields (title, address, phone, website,
// category, rating, count, lat/lng, place_id, plus_code, status).
// Reviews, popular_times, about, menu, owner are filled in later commits.
//
// Always-fail-soft: returns (nil, nil) when the detail panel doesn't load
// so a worker can skip the URL without aborting the whole task.

var _ = time.Second // keep `time` import alive — reserved for future timeouts
