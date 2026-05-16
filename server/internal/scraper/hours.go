package scraper

import (
	"context"
	"strings"

	"github.com/playwright-community/playwright-go"
)

// hoursTableJS scans every <table> with 5-8 rows (typical opening-hours table)
// and returns {day: "time text"} flattened. Multi-range cells separated by
// newlines are collapsed to a single space.
const hoursTableJS = `() => {
  const out = {};
  const tables = document.querySelectorAll('table');
  for (const t of tables) {
    const rows = t.querySelectorAll('tr');
    if (rows.length < 5 || rows.length > 8) continue;
    const tmp = {};
    rows.forEach(r => {
      const cells = r.querySelectorAll('td, th');
      if (cells.length >= 2) {
        const day = cells[0].innerText.trim();
        const time = cells[1].innerText.replace(/\n+/g, ' ').trim();
        if (day && time) tmp[day] = time;
      }
    });
    if (Object.keys(tmp).length >= 5) return tmp;
  }
  return out;
}`

// ScrapeHours opens the hours toggle on the place panel and returns the table
// rendered as gosom-style {day: [time-range]} map. Closed days come back as
// the localized phrase from Google ("Tutup" / "Closed") in a single-item list.
func ScrapeHours(ctx context.Context, page playwright.Page) map[string][]string {
	// Expand the hours panel — the table is collapsed by default for some
	// places. The toggle button has data-item-id starting with "oh".
	toggle := page.Locator(
		`button[data-item-id*="oh"], button[aria-label*="jam" i], button[aria-label*="hours" i]`,
	).First()
	if n, _ := toggle.Count(); n > 0 {
		_ = toggle.Click(playwright.LocatorClickOptions{
			Timeout: playwright.Float(2000),
		})
		HumanDelay(ctx, 0, 0, true)
	}

	v, err := page.Evaluate(hoursTableJS)
	if err != nil || v == nil {
		return nil
	}
	m, ok := v.(map[string]any)
	if !ok || len(m) == 0 {
		return nil
	}
	out := make(map[string][]string, len(m))
	for day, raw := range m {
		s, ok := raw.(string)
		if !ok || s == "" {
			continue
		}
		s = CleanText(s)
		out[CleanText(day)] = splitTimeRanges(s)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// splitTimeRanges turns "09.00–14.00 17.00–21.00" into ["09.00–14.00", "17.00–21.00"].
// Single-range and "Tutup/Closed" values become a one-element slice.
func splitTimeRanges(s string) []string {
	if s == "" {
		return nil
	}
	// Quick path: contains exactly one en-dash → single range.
	parts := strings.Split(s, "–")
	if len(parts) <= 2 {
		return []string{s}
	}
	// Walk parts: every two consecutive halves form one range, separated by space.
	out := make([]string, 0, len(parts)-1)
	left := strings.TrimSpace(parts[0])
	for i := 1; i < len(parts); i++ {
		right := strings.TrimSpace(parts[i])
		// "right" looks like "14.00 17.00" when two ranges run together — split
		// at the rightmost space to peel off the start of the next range.
		fields := strings.Fields(right)
		if len(fields) >= 2 && i < len(parts)-1 {
			end := fields[0]
			nextStart := strings.Join(fields[1:], " ")
			out = append(out, left+"–"+end)
			left = nextStart
		} else {
			out = append(out, left+"–"+right)
		}
	}
	if len(out) == 0 {
		return []string{s}
	}
	return out
}
