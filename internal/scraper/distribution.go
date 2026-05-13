package scraper

import (
	"github.com/playwright-community/playwright-go"
)

// reviewDistributionJS scans every aria-label on the page for "X bintang/stars,
// N ulasan/reviews" patterns and returns max-count-per-star as a map keyed by
// integer star 1..5. Returns null when nothing matched.
const reviewDistributionJS = `() => {
  const out = {1: 0, 2: 0, 3: 0, 4: 0, 5: 0};
  let found = false;
  document.querySelectorAll('[aria-label]').forEach(el => {
    const label = el.getAttribute('aria-label') || '';
    let star = null, count = null, m;
    if ((m = label.match(/(\d)\s*bintang[^\d]*([\d.,]+)\s*ulasan/i))) {
      star = +m[1]; count = +m[2].replace(/[.,]/g, '');
    } else if ((m = label.match(/([\d.,]+)\s*ulasan[^\d]*(\d)\s*bintang/i))) {
      count = +m[1].replace(/[.,]/g, ''); star = +m[2];
    } else if ((m = label.match(/(\d)\s*stars?[^\d]*([\d.,]+)\s*reviews?/i))) {
      star = +m[1]; count = +m[2].replace(/[.,]/g, '');
    } else if ((m = label.match(/([\d.,]+)\s*reviews?[^\d]*(\d)\s*stars?/i))) {
      count = +m[1].replace(/[.,]/g, ''); star = +m[2];
    }
    if (star !== null && count !== null && !isNaN(count) && star >= 1 && star <= 5) {
      if (count > out[star]) out[star] = count;
      found = true;
    }
  });
  return found ? out : null;
}`

// ScrapeReviewsPerRating returns a map[1..5]int suitable for PlacePayload.ReviewsPerRating.
// Assumes the Reviews tab has been opened so the histogram aria-labels are present.
func ScrapeReviewsPerRating(page playwright.Page) map[int]int {
	v, err := page.Evaluate(reviewDistributionJS)
	if err != nil || v == nil {
		return nil
	}
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	out := make(map[int]int, 5)
	for k, raw := range m {
		// Keys come back as strings "1".."5" — convert.
		star := 0
		switch len(k) {
		case 1:
			star = int(k[0] - '0')
		}
		if star < 1 || star > 5 {
			continue
		}
		switch n := raw.(type) {
		case int:
			out[star] = n
		case int64:
			out[star] = int(n)
		case float64:
			out[star] = int(n)
		}
	}
	return out
}
