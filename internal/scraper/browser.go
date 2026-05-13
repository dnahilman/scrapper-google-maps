package scraper

import (
	"fmt"
	"math/rand"

	"github.com/playwright-community/playwright-go"
)

// Session bundles a Playwright runtime + Chromium browser + an isolated context.
// One Session per worker slot — caller is responsible for Close() when done.
type Session struct {
	pw      *playwright.Playwright
	browser playwright.Browser
	context playwright.BrowserContext
}

type SessionConfig struct {
	Headless bool
}

// NewSession launches Chromium with stealth applied. No cookie persistence.
func NewSession(cfg SessionConfig) (*Session, error) {
	pw, err := playwright.Run()
	if err != nil {
		return nil, fmt.Errorf("playwright.Run: %w", err)
	}

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(cfg.Headless),
		Args: []string{
			"--disable-blink-features=AutomationControlled",
			"--disable-dev-shm-usage",
			"--no-sandbox",
			"--disable-features=IsolateOrigins,site-per-process",
		},
	})
	if err != nil {
		_ = pw.Stop()
		return nil, fmt.Errorf("chromium.Launch: %w", err)
	}

	vp := viewportPool[rand.Intn(len(viewportPool))]
	ctx, err := browser.NewContext(playwright.BrowserNewContextOptions{
		Viewport: &playwright.Size{
			Width:  vp.Width,
			Height: vp.Height,
		},
		UserAgent:  playwright.String(userAgentPool[rand.Intn(len(userAgentPool))]),
		Locale:     playwright.String(localePool[rand.Intn(len(localePool))]),
		TimezoneId: playwright.String("Asia/Jakarta"),
		Geolocation: &playwright.Geolocation{
			Latitude:  -6.9175,
			Longitude: 107.6191,
		},
		Permissions: []string{"geolocation"},
	})
	if err != nil {
		_ = browser.Close()
		_ = pw.Stop()
		return nil, fmt.Errorf("browser.NewContext: %w", err)
	}

	if err := ctx.AddInitScript(playwright.Script{Content: playwright.String(stealthScript)}); err != nil {
		_ = ctx.Close()
		_ = browser.Close()
		_ = pw.Stop()
		return nil, fmt.Errorf("add stealth script: %w", err)
	}

	return &Session{pw: pw, browser: browser, context: ctx}, nil
}

// NewPage opens a fresh page in this session.
func (s *Session) NewPage() (playwright.Page, error) {
	return s.context.NewPage()
}

// Close releases all resources. Safe to call multiple times.
func (s *Session) Close() {
	if s == nil {
		return
	}
	if s.context != nil {
		_ = s.context.Close()
		s.context = nil
	}
	if s.browser != nil {
		_ = s.browser.Close()
		s.browser = nil
	}
	if s.pw != nil {
		_ = s.pw.Stop()
		s.pw = nil
	}
}
