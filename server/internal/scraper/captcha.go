package scraper

import (
	"errors"
	"strings"

	"github.com/playwright-community/playwright-go"
)

// ErrCaptcha is returned when Google's anti-bot interstitial is detected.
var ErrCaptcha = errors.New("captcha or unusual-traffic page detected")

var captchaTriggers = []string{
	"unusual traffic",
	"captcha",
	"/sorry/",
	"verify you're human",
	"lalu lintas yang tidak biasa",
}

// CheckCaptcha returns ErrCaptcha if the current page content matches any known trigger.
func CheckCaptcha(page playwright.Page) error {
	content, err := page.Content()
	if err != nil {
		return err
	}
	lower := strings.ToLower(content)
	for _, t := range captchaTriggers {
		if strings.Contains(lower, t) {
			return ErrCaptcha
		}
	}
	return nil
}
