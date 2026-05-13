package scraper

import (
	"context"
	"fmt"

	"github.com/playwright-community/playwright-go"
)

// clickTab tries to click a tab/button matched by any of the supplied
// aria-label substrings (case-insensitive). Returns true if a click succeeded.
//
// Mirrors server/src/gmaps.py::_click_tab — used to open the "Ulasan/Reviews"
// and "Tentang/About" panels.
func clickTab(ctx context.Context, page playwright.Page, patterns ...string) bool {
	for _, pat := range patterns {
		sels := []string{
			fmt.Sprintf(`button[role="tab"][aria-label*="%s" i]`, pat),
			fmt.Sprintf(`button[aria-label*="%s" i]`, pat),
		}
		for _, sel := range sels {
			loc := page.Locator(sel).First()
			n, err := loc.Count()
			if err != nil || n == 0 {
				continue
			}
			_ = loc.ScrollIntoViewIfNeeded(playwright.LocatorScrollIntoViewIfNeededOptions{
				Timeout: playwright.Float(2000),
			})
			if err := loc.Click(playwright.LocatorClickOptions{
				Timeout: playwright.Float(3000),
			}); err != nil {
				continue
			}
			HumanDelay(ctx, 0, 0, true)
			return true
		}
	}
	return false
}
