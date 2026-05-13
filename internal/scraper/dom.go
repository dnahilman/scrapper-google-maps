package scraper

import (
	"errors"

	"github.com/playwright-community/playwright-go"
)

// safeText returns the first matching element's text (trimmed) or "" if missing.
func safeText(page playwright.Page, selector string) string {
	loc := page.Locator(selector).First()
	n, err := loc.Count()
	if err != nil || n == 0 {
		return ""
	}
	txt, err := loc.TextContent(playwright.LocatorTextContentOptions{
		Timeout: playwright.Float(2000),
	})
	if err != nil {
		return ""
	}
	return CleanText(txt)
}

// safeAttr returns the first matching element's attribute or "" if missing.
func safeAttr(page playwright.Page, selector, attr string) string {
	loc := page.Locator(selector).First()
	n, err := loc.Count()
	if err != nil || n == 0 {
		return ""
	}
	val, err := loc.GetAttribute(attr, playwright.LocatorGetAttributeOptions{
		Timeout: playwright.Float(2000),
	})
	if err != nil || val == "" {
		return ""
	}
	return CleanText(val)
}

// waitForAny waits for the first selector that matches and returns nil.
// If none matched within timeout, returns the timeout error.
func waitForAny(page playwright.Page, selector string, timeoutMs float64) error {
	if page == nil {
		return errors.New("nil page")
	}
	_, err := page.WaitForSelector(selector, playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(timeoutMs),
	})
	return err
}
