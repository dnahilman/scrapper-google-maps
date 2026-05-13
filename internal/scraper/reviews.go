package scraper

import (
	"context"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"

	"github.com/dnahilman/scrapper-go/internal/domain"
)

// ReviewsOptions controls the reviews scraping behaviour.
type ReviewsOptions struct {
	Max               int  // hard cap on reviews kept (0 = no cap)
	MaxAgeDays        int  // skip reviews older than this (0 = keep all)
	SortByNewest      bool // click sort → "Terbaru/Newest" before scraping
	SkipEmpty         bool // drop reviews whose text is < 3 chars
}

// reviewCardJS extracts every fielded payload from a single [data-review-id]
// card via Element.evaluate. Returns an object matching `reviewCardJSResult`.
const reviewCardJS = `el => {
  const txt = (sel) => el.querySelector(sel)?.innerText?.trim() || null;
  const attr = (sel, a) => el.querySelector(sel)?.getAttribute(a) || null;
  const isAvatar = url => /googleusercontent\.com\/a[-]?\//.test(url);
  const photos = new Set();
  el.querySelectorAll('button[style*="background-image"], div[style*="background-image"]').forEach(b => {
    const style = b.getAttribute('style') || '';
    const m = style.match(/url\("?(https:\/\/[^")]+googleusercontent[^")]+)"?\)/);
    if (m) {
      let url = m[1];
      if (isAvatar(url)) return;
      const eqIdx = url.indexOf('=');
      if (eqIdx > -1) url = url.substring(0, eqIdx);
      photos.add(url + '=w800-h600-k-no');
    }
  });
  el.querySelectorAll('img').forEach(img => {
    const src = img.src || img.dataset?.src || '';
    if (!src.includes('googleusercontent.com')) return;
    if (isAvatar(src)) return;
    if (/=s[1-9]\d?-/.test(src)) return;
    let base = src;
    const eqIdx = src.indexOf('=');
    if (eqIdx > -1) base = src.substring(0, eqIdx);
    photos.add(base + '=w800-h600-k-no');
  });
  return {
    review_id: el.getAttribute('data-review-id'),
    author: txt('div.d4r55') || txt('[class*="d4r55"]') || txt('button[jsaction*="reviewerLink"]'),
    author_avatar: attr('button[jsaction*="reviewerLink"] img', 'src'),
    rating_aria: attr('span[role="img"][aria-label*="bintang" i]', 'aria-label')
              || attr('span[role="img"][aria-label*="star" i]', 'aria-label'),
    text: txt('span.wiI7pd') || txt('[class*="wiI7pd"]'),
    time: txt('span.rsqaWe') || txt('[class*="rsqaWe"]'),
    owner_response: txt('div.CDe7pd') || txt('[class*="CDe7pd"]'),
    review_photos: Array.from(photos),
  };
}`

// rawReviewCard is the shape returned by reviewCardJS.
type rawReviewCard struct {
	ReviewID      string   `json:"review_id"`
	Author        string   `json:"author"`
	AuthorAvatar  string   `json:"author_avatar"`
	RatingAria    string   `json:"rating_aria"`
	Text          string   `json:"text"`
	Time          string   `json:"time"`
	OwnerResponse string   `json:"owner_response"`
	ReviewPhotos  []string `json:"review_photos"`
}

// ScrapeReviews opens the Reviews tab, optionally sorts by newest, scrolls the
// list until exhausted (or hard caps), expands truncated text, and returns
// gosom-style ReviewPayload entries.
func ScrapeReviews(ctx context.Context, page playwright.Page, opts ReviewsOptions) []domain.ReviewPayload {
	if !clickTab(ctx, page, "Ulasan", "Reviews") {
		return nil
	}
	if err := waitForAny(page, "div[data-review-id]", 10000); err != nil {
		return nil
	}
	if opts.SortByNewest {
		sortReviewsByNewest(ctx, page)
		// Let the list refresh after sort.
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(1500 * time.Millisecond):
		}
	}

	scrollContainer := acquireScrollHandle(page)
	scrollReviewsList(ctx, page, scrollContainer, opts)
	expandTruncated(page, opts.Max)

	cards, err := page.Locator("div[data-review-id]").All()
	if err != nil {
		return nil
	}

	out := make([]domain.ReviewPayload, 0, len(cards))
	seen := make(map[string]struct{}, len(cards))
	for _, card := range cards {
		raw, ok := evaluateReviewCard(card)
		if !ok || raw.ReviewID == "" {
			continue
		}
		if _, dup := seen[raw.ReviewID]; dup {
			continue
		}
		seen[raw.ReviewID] = struct{}{}

		text := CleanText(raw.Text)
		if opts.SkipEmpty && len(strings.TrimSpace(text)) < 3 {
			continue
		}

		rating := 0
		if r, ok := ParseRating(raw.RatingAria); ok {
			rating = int(r + 0.5)
			if rating > 5 {
				rating = 5
			}
		}

		ageDays := ParseAgeDays(raw.Time)
		if opts.MaxAgeDays > 0 && ageDays > 0 && ageDays > opts.MaxAgeDays {
			continue
		}

		rp := domain.ReviewPayload{
			ReviewID:       raw.ReviewID,
			Name:           CleanText(raw.Author),
			ProfilePicture: raw.AuthorAvatar,
			Rating:         rating,
			Description:    text,
			Images:         raw.ReviewPhotos,
			When:           CleanText(raw.Time),
			AgeDays:        ageDays,
		}
		if ownerResp := CleanText(raw.OwnerResponse); ownerResp != "" {
			rp.OwnerResponse = &domain.OwnerResponse{Text: ownerResp}
		}
		out = append(out, rp)
		if opts.Max > 0 && len(out) >= opts.Max {
			break
		}
	}
	return out
}

// sortReviewsByNewest clicks the sort dropdown and picks "Terbaru/Newest".
func sortReviewsByNewest(ctx context.Context, page playwright.Page) bool {
	btn := page.Locator(
		`button[aria-label*="urutkan" i], button[aria-label*="sort" i], button[data-value*="Sort" i]`,
	).First()
	if n, _ := btn.Count(); n == 0 {
		return false
	}
	if err := btn.Click(playwright.LocatorClickOptions{
		Timeout: playwright.Float(3000),
	}); err != nil {
		return false
	}
	HumanDelay(ctx, 0, 0, true)

	for _, label := range []string{"Terbaru", "Newest"} {
		sel := `div[role="menuitemradio"]:has-text("` + label + `"), div[role="menuitem"]:has-text("` + label + `")`
		opt := page.Locator(sel).First()
		if n, _ := opt.Count(); n == 0 {
			continue
		}
		if err := opt.Click(playwright.LocatorClickOptions{
			Timeout: playwright.Float(2000),
		}); err == nil {
			HumanDelay(ctx, 0, 0, true)
			return true
		}
	}
	return false
}

// acquireScrollHandle finds the scrollable ancestor of the first review card.
// Returns nil JSHandle if not found — caller falls back to mouse wheel.
const findScrollAncestorJS = `() => {
  const card = document.querySelector('[data-review-id]');
  if (!card) return null;
  let el = card.parentElement;
  while (el) {
    const cs = getComputedStyle(el);
    if ((cs.overflowY === 'auto' || cs.overflowY === 'scroll') && el.scrollHeight > el.clientHeight) {
      return el;
    }
    el = el.parentElement;
  }
  return null;
}`

func acquireScrollHandle(page playwright.Page) playwright.JSHandle {
	h, err := page.EvaluateHandle(findScrollAncestorJS)
	if err != nil || h == nil {
		return nil
	}
	return h
}

// scrollReviewsList scrolls the reviews container until no new entries appear,
// the hard cap is reached, or the oldest visible review crosses MaxAgeDays.
func scrollReviewsList(ctx context.Context, page playwright.Page, handle playwright.JSHandle, opts ReviewsOptions) {
	prev := 0
	stuck := 0
	for {
		if ctx.Err() != nil {
			return
		}
		scrolled := false
		if handle != nil {
			if _, err := handle.Evaluate("el => el && el.scrollTo(0, el.scrollHeight)", nil); err == nil {
				scrolled = true
			}
		}
		if !scrolled {
			_ = page.Mouse().Wheel(0, 4000)
		}

		HumanDelay(ctx, 0, 0, true)

		count, _ := page.Locator("div[data-review-id]").Count()
		if opts.Max > 0 && count >= opts.Max {
			return
		}

		// Early stop on oldest-card-age (reviews sorted newest first).
		if opts.SortByNewest && opts.MaxAgeDays > 0 && count > 0 {
			lastTime := lastReviewTime(page, count-1)
			if age := ParseAgeDays(lastTime); age > 0 && age > opts.MaxAgeDays {
				return
			}
		}

		if count == prev {
			stuck++
			if stuck >= 3 {
				return
			}
		} else {
			stuck = 0
		}
		prev = count
	}
}

const lastReviewTimeJS = `idx => {
  const cards = document.querySelectorAll('[data-review-id]');
  const c = cards[idx];
  if (!c) return null;
  const t = c.querySelector('span.rsqaWe, [class*="rsqaWe"]');
  return t?.innerText?.trim() || null;
}`

func lastReviewTime(page playwright.Page, idx int) string {
	v, err := page.Evaluate(lastReviewTimeJS, idx)
	if err != nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// expandTruncated clicks every "Lainnya / Selengkapnya / More" button to reveal
// hidden review text. Best-effort: failures are ignored.
func expandTruncated(page playwright.Page, maxClicks int) {
	if maxClicks <= 0 {
		maxClicks = 500
	}
	selectors := []string{
		`button:has-text("Lainnya")`,
		`button:has-text("Selengkapnya")`,
		`button:has-text("More")`,
	}
	for _, sel := range selectors {
		btns, _ := page.Locator(sel).All()
		for i, btn := range btns {
			if i >= maxClicks {
				break
			}
			_ = btn.Click(playwright.LocatorClickOptions{
				Timeout: playwright.Float(800),
			})
		}
	}
}

// evaluateReviewCard runs reviewCardJS on a single locator and decodes the
// result into rawReviewCard. Returns (zero, false) on any failure.
func evaluateReviewCard(card playwright.Locator) (rawReviewCard, bool) {
	v, err := card.Evaluate(reviewCardJS, nil)
	if err != nil || v == nil {
		return rawReviewCard{}, false
	}
	// Playwright returns an interface{} backed by map[string]interface{}.
	m, ok := v.(map[string]any)
	if !ok {
		return rawReviewCard{}, false
	}
	r := rawReviewCard{
		ReviewID:      asString(m["review_id"]),
		Author:        asString(m["author"]),
		AuthorAvatar:  asString(m["author_avatar"]),
		RatingAria:    asString(m["rating_aria"]),
		Text:          asString(m["text"]),
		Time:          asString(m["time"]),
		OwnerResponse: asString(m["owner_response"]),
		ReviewPhotos:  asStringSlice(m["review_photos"]),
	}
	return r, true
}

func asString(v any) string {
	if v == nil {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

func asStringSlice(v any) []string {
	if v == nil {
		return nil
	}
	xs, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(xs))
	for _, x := range xs {
		if s, ok := x.(string); ok && s != "" {
			out = append(out, s)
		}
	}
	return out
}
