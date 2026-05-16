package scraper

import (
	"context"

	"github.com/playwright-community/playwright-go"

	"github.com/dnahilman/scrapper-go/internal/domain"
)

// aboutSectionsJS walks the main panel collecting every h2 → following <li>
// items as `{section_name: [item1, item2, ...]}`. Items use aria-label when
// present (which carries the enabled/disabled state in Google's UI).
const aboutSectionsJS = `() => {
  const norm = s => (s || '').replace(/\s+/g, ' ').trim();
  const main = document.querySelector('div[role="main"]') || document.body;
  const h2s = Array.from(main.querySelectorAll('h2')).filter(h => h.innerText.trim());
  if (h2s.length === 0) return {};
  const container = h2s[0].closest('div[role="region"]') || main;
  const result = {};
  let currentSection = null;
  const items = new Map();
  const walker = document.createTreeWalker(container, NodeFilter.SHOW_ELEMENT);
  const flush = () => {
    if (currentSection && items.size > 0) {
      result[currentSection] = Array.from(items.entries()).map(([name, enabled]) => ({name, enabled}));
    }
    items.clear();
  };
  let node;
  while ((node = walker.nextNode())) {
    if (node.tagName === 'H2') {
      const name = norm(node.innerText);
      if (name) { flush(); currentSection = name; }
    } else if (currentSection && node.tagName === 'LI') {
      const aria = node.getAttribute('aria-label');
      const txt = aria || node.innerText || '';
      const value = norm(txt);
      if (!value || value.length >= 200 || value === currentSection) continue;
      // Google prefixes negative items with "Tidak ada" / "No" — capture as enabled=false.
      const lower = value.toLowerCase();
      const negative = lower.startsWith('tidak ada') || lower.startsWith('no ') || lower.startsWith('tidak menyediakan');
      items.set(value, !negative);
    }
  }
  flush();
  return result;
}`

// ScrapeAbout opens the "Tentang/About" tab and returns gosom-style sections.
// Returns nil when the tab isn't available (e.g. barbershops without amenities).
func ScrapeAbout(ctx context.Context, page playwright.Page) []domain.About {
	if !clickTab(ctx, page, "Tentang", "About") {
		return nil
	}
	if err := waitForAny(page, `h2, [role="region"]`, 5000); err != nil {
		return nil
	}

	v, err := page.Evaluate(aboutSectionsJS)
	if err != nil || v == nil {
		return nil
	}
	sections, ok := v.(map[string]any)
	if !ok || len(sections) == 0 {
		return nil
	}

	out := make([]domain.About, 0, len(sections))
	for sectionName, raw := range sections {
		opts := decodeAboutOptions(raw)
		if len(opts) == 0 {
			continue
		}
		out = append(out, domain.About{
			ID:      slugifyAbout(sectionName),
			Name:    CleanText(sectionName),
			Options: opts,
		})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func decodeAboutOptions(raw any) []domain.AboutOption {
	xs, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]domain.AboutOption, 0, len(xs))
	seen := make(map[string]struct{}, len(xs))
	for _, x := range xs {
		m, ok := x.(map[string]any)
		if !ok {
			continue
		}
		name := CleanText(asString(m["name"]))
		if name == "" {
			continue
		}
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}
		enabled := true
		if b, ok := m["enabled"].(bool); ok {
			enabled = b
		}
		out = append(out, domain.AboutOption{Name: name, Enabled: enabled})
	}
	return out
}

// slugifyAbout produces a stable id like "service_options" from "Service options".
func slugifyAbout(name string) string {
	out := make([]rune, 0, len(name))
	prevDash := false
	for _, r := range name {
		switch {
		case r >= 'A' && r <= 'Z':
			out = append(out, r+32)
			prevDash = false
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			out = append(out, r)
			prevDash = false
		default:
			if !prevDash && len(out) > 0 {
				out = append(out, '_')
				prevDash = true
			}
		}
	}
	for len(out) > 0 && out[len(out)-1] == '_' {
		out = out[:len(out)-1]
	}
	return string(out)
}
