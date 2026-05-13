package scraper

import (
	"strings"

	"github.com/playwright-community/playwright-go"

	"github.com/dnahilman/scrapper-go/internal/domain"
)

// externalLinksJS scans every <a href> in the main panel and returns external
// (non-Google) URLs classified into "reservation" or "order" buckets based on
// known platform hostnames + Indonesian/English text hints.
//
// Returns {reservations: [{link, source_host}], order_online: [{link, source_host}]}
// — Source is finalised in Go using friendlyPlatformName so we can keep this
// regex list small.
const externalLinksJS = `() => {
  const RESERVATION_HOSTS = [
    'opentable', 'chope', 'resdiary', 'sevenrooms', 'quandoo',
    'tablein', 'eveve', 'tableagent', 'bookatable'
  ];
  const ORDER_HOSTS = [
    'gofood', 'grab', 'grabfood', 'shopeefood', 'shopee',
    'tokopedia', 'traveloka', 'pickup', 'doordash', 'ubereats',
    'foodpanda', 'halofresh'
  ];
  const RESERVATION_HINTS = ['reserve', 'pesan meja', 'book a table', 'reservasi'];
  const ORDER_HINTS = ['order online', 'pesan online', 'delivery', 'pesan antar', 'gofood', 'grabfood'];

  const hostnameOf = (href) => {
    try {
      return new URL(href).hostname.replace(/^www\./, '').toLowerCase();
    } catch (e) { return ''; }
  };
  const hostMatches = (host, list) => list.some(k => host.includes(k));
  const textMatches = (text, list) => list.some(k => text.includes(k));

  const main = document.querySelector('div[role="main"]') || document.body;
  const seenRes = new Set();
  const seenOrd = new Set();
  const reservations = [];
  const order_online = [];

  main.querySelectorAll('a[href]').forEach(a => {
    const href = a.href || '';
    if (!href || href.startsWith('javascript:')) return;
    const host = hostnameOf(href);
    if (!host) return;
    if (host.endsWith('google.com') || host.endsWith('googleusercontent.com') ||
        host.endsWith('youtube.com') || host.endsWith('goo.gl')) return;

    const aria = (a.getAttribute('aria-label') || '').toLowerCase();
    const text = (a.innerText || '').toLowerCase();
    const itemId = (a.getAttribute('data-item-id') || '').toLowerCase();
    const combined = aria + ' ' + text + ' ' + itemId;

    const isReservation = hostMatches(host, RESERVATION_HOSTS) ||
                          /reservation/.test(itemId) ||
                          textMatches(combined, RESERVATION_HINTS);
    const isOrder = hostMatches(host, ORDER_HOSTS) ||
                    /order/.test(itemId) ||
                    textMatches(combined, ORDER_HINTS);

    if (isReservation && !seenRes.has(href)) {
      seenRes.add(href);
      reservations.push({ link: href, source_host: host });
    } else if (isOrder && !seenOrd.has(href)) {
      seenOrd.add(href);
      order_online.push({ link: href, source_host: host });
    }
  });

  return { reservations, order_online };
}`

// ScrapeExternalLinks returns reservation + order-online link buckets.
// Both slices may be nil when nothing was found.
func ScrapeExternalLinks(page playwright.Page) (reservations, orderOnline []domain.LinkSource) {
	v, err := page.Evaluate(externalLinksJS)
	if err != nil || v == nil {
		return nil, nil
	}
	root, ok := v.(map[string]any)
	if !ok {
		return nil, nil
	}
	reservations = decodeLinkBucket(root["reservations"])
	orderOnline = decodeLinkBucket(root["order_online"])
	return
}

func decodeLinkBucket(raw any) []domain.LinkSource {
	xs, ok := raw.([]any)
	if !ok || len(xs) == 0 {
		return nil
	}
	out := make([]domain.LinkSource, 0, len(xs))
	for _, x := range xs {
		m, ok := x.(map[string]any)
		if !ok {
			continue
		}
		link := asString(m["link"])
		if link == "" {
			continue
		}
		out = append(out, domain.LinkSource{
			Link:   link,
			Source: friendlyPlatformName(asString(m["source_host"])),
		})
	}
	return out
}

// friendlyPlatformName converts "gofood.co.id" → "GoFood", "opentable.com" → "OpenTable".
// Unknown hosts get returned cleaned (sans TLD).
func friendlyPlatformName(host string) string {
	if host == "" {
		return ""
	}
	host = strings.ToLower(host)
	switch {
	case strings.Contains(host, "gofood"):
		return "GoFood"
	case strings.Contains(host, "grabfood"):
		return "GrabFood"
	case strings.Contains(host, "shopeefood"):
		return "ShopeeFood"
	case strings.Contains(host, "shopee"):
		return "Shopee"
	case strings.Contains(host, "tokopedia"):
		return "Tokopedia"
	case strings.Contains(host, "traveloka"):
		return "Traveloka Eats"
	case strings.Contains(host, "halofresh"):
		return "HaloFresh"
	case strings.Contains(host, "doordash"):
		return "DoorDash"
	case strings.Contains(host, "ubereats"):
		return "Uber Eats"
	case strings.Contains(host, "foodpanda"):
		return "Foodpanda"
	case strings.Contains(host, "opentable"):
		return "OpenTable"
	case strings.Contains(host, "chope"):
		return "Chope"
	case strings.Contains(host, "resdiary"):
		return "ResDiary"
	case strings.Contains(host, "sevenrooms"):
		return "SevenRooms"
	case strings.Contains(host, "quandoo"):
		return "Quandoo"
	case strings.Contains(host, "tablein"):
		return "Tablein"
	case strings.Contains(host, "bookatable"):
		return "Bookatable"
	}
	// Fallback: strip TLDs and title-case the remaining label.
	host = strings.TrimPrefix(host, "www.")
	parts := strings.Split(host, ".")
	if len(parts) == 0 {
		return host
	}
	label := parts[0]
	if label == "" {
		return host
	}
	return strings.ToUpper(label[:1]) + label[1:]
}
