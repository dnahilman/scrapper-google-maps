package scraper

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// reEmail matches RFC-5322ish addresses. Conservative — no exotic characters.
var reEmail = regexp.MustCompile(`(?i)\b[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}\b`)

// emailNoiseSuffixes drops false positives like "logo@2x.png" or
// "sentry@1.0.0" picked up from minified JS bundles.
var emailNoiseSuffixes = []string{
	".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp",
	".css", ".js", ".map",
	".sentry.io", ".sentry-cdn.com",
	"@2x", "@3x",
}

// emailNoiseDomains drops common placeholder/example addresses.
var emailNoiseDomains = []string{
	"example.com", "example.org", "example.net",
	"yourdomain.com", "domain.com",
	"sentry.io", "wixpress.com",
}

// pathsToCrawl is checked in order. Stop early once we have addresses.
var pathsToCrawl = []string{"", "/contact", "/contact-us", "/kontak", "/about", "/about-us"}

// CrawlEmails fetches the place website and extracts email addresses with
// reasonable false-positive filtering. Bounded: max 3 pages, 8s per page,
// 256KB body cap. Returns nil on any failure or when crawl is disabled.
func CrawlEmails(ctx context.Context, website string) []string {
	if website == "" {
		return nil
	}
	base, err := url.Parse(website)
	if err != nil || base.Host == "" {
		return nil
	}
	if base.Scheme == "" {
		base.Scheme = "https"
	}

	client := &http.Client{
		Timeout: 8 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	seen := make(map[string]struct{}, 8)
	out := make([]string, 0, 8)
	pagesCrawled := 0
	const maxPages = 3

	for _, path := range pathsToCrawl {
		if pagesCrawled >= maxPages {
			break
		}
		if ctx.Err() != nil {
			break
		}
		u := *base
		u.Path = path
		u.RawQuery = ""
		u.Fragment = ""

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent",
			"Mozilla/5.0 (compatible; ScrapperGoBot/1.0; +https://example.com/bot)")
		req.Header.Set("Accept", "text/html,application/xhtml+xml")

		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		body := readCapped(resp.Body, 256*1024)
		resp.Body.Close()
		if body == "" {
			continue
		}
		pagesCrawled++

		for _, addr := range reEmail.FindAllString(body, -1) {
			addr = strings.ToLower(strings.TrimSpace(addr))
			if !isUsefulEmail(addr) {
				continue
			}
			if _, dup := seen[addr]; dup {
				continue
			}
			seen[addr] = struct{}{}
			out = append(out, addr)
		}
		if len(out) >= 8 {
			break
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func readCapped(r io.Reader, max int64) string {
	buf := make([]byte, 0, 32*1024)
	tmp := make([]byte, 8*1024)
	var n int64
	for n < max {
		k, err := r.Read(tmp)
		if k > 0 {
			buf = append(buf, tmp[:k]...)
			n += int64(k)
		}
		if err != nil {
			break
		}
	}
	return string(buf)
}

func isUsefulEmail(addr string) bool {
	if addr == "" || len(addr) > 254 {
		return false
	}
	for _, s := range emailNoiseSuffixes {
		if strings.Contains(addr, s) {
			return false
		}
	}
	domain := addr[strings.IndexByte(addr, '@')+1:]
	for _, d := range emailNoiseDomains {
		if domain == d {
			return false
		}
	}
	// Reject if the local part has consecutive dots or weird chars.
	local := addr[:strings.IndexByte(addr, '@')]
	if strings.Contains(local, "..") {
		return false
	}
	return true
}
