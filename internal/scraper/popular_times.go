package scraper

import (
	"github.com/playwright-community/playwright-go"
)

// popularTimesJS scans aria-label attributes for the popular-times bars Google
// renders inside the place panel. Each bar has a label such as:
//   "Senin pukul 14, biasanya cukup ramai (50%)"
//   "Monday at 2 PM, usually a little busy (50%)"
// We extract (day, hour 0-23, percent 0-100) and aggregate into
// gosom-style map[day]map[hour]percent.
//
// Bars without an explicit % digit are skipped (we don't try to map fuzzy
// phrases like "moderately busy" to a number).
const popularTimesJS = `() => {
  const DAYS_ID = {
    'minggu': 'Sunday', 'senin': 'Monday', 'selasa': 'Tuesday',
    'rabu': 'Wednesday', 'kamis': 'Thursday', 'jumat': 'Friday',
    "jum'at": 'Friday', 'sabtu': 'Saturday'
  };
  const DAYS_EN = new Set(['monday','tuesday','wednesday','thursday','friday','saturday','sunday']);
  const out = {};
  let found = false;

  // Convert "9 PM" / "1 AM" / "13" / "pukul 14" → 24h hour int, or null.
  const toHour24 = (raw) => {
    if (raw === undefined || raw === null) return null;
    const s = String(raw).trim().toLowerCase();
    // "9 pm" / "1 am"
    let m = s.match(/^(\d{1,2})\s*(am|pm)$/);
    if (m) {
      let h = +m[1] % 12;
      if (m[2] === 'pm') h += 12;
      return h;
    }
    // Plain integer 0..23.
    m = s.match(/^(\d{1,2})$/);
    if (m) {
      const h = +m[1];
      if (h >= 0 && h <= 23) return h;
    }
    return null;
  };

  document.querySelectorAll('[aria-label]').forEach(el => {
    const label = (el.getAttribute('aria-label') || '').trim();
    if (label.length < 5 || label.length > 200) return;
    const lower = label.toLowerCase();

    // Find day token.
    let dayName = null;
    for (const [id, en] of Object.entries(DAYS_ID)) {
      if (lower.includes(id)) { dayName = en; break; }
    }
    if (!dayName) {
      for (const en of DAYS_EN) {
        if (lower.includes(en)) { dayName = en.charAt(0).toUpperCase() + en.slice(1); break; }
      }
    }
    if (!dayName) return;

    // Percent.
    const pm = lower.match(/(\d{1,3})\s*%/);
    if (!pm) return;
    const percent = Math.max(0, Math.min(100, +pm[1]));

    // Hour — try Indonesian "pukul N" / English "at N AM/PM" / "at N:00".
    let hour = null;
    let hm = lower.match(/pukul\s*(\d{1,2})/);
    if (hm) hour = toHour24(hm[1]);
    if (hour === null) {
      hm = lower.match(/at\s*(\d{1,2})(?:\s*[:.]?\s*00)?\s*(am|pm)?/);
      if (hm) hour = toHour24(hm[1] + (hm[2] ? ' ' + hm[2] : ''));
    }
    if (hour === null) return;

    if (!out[dayName]) out[dayName] = {};
    // Take the highest value if multiple labels collide.
    if ((out[dayName][hour] || 0) < percent) {
      out[dayName][hour] = percent;
      found = true;
    }
  });
  return found ? out : null;
}`

// ScrapePopularTimes returns gosom-style popular_times — map[day]map[hour]percent.
// Returns nil when nothing recognisable was found (silently absent for many
// places, e.g. low-traffic spots without enough visit data).
func ScrapePopularTimes(page playwright.Page) map[string]map[int]int {
	v, err := page.Evaluate(popularTimesJS)
	if err != nil || v == nil {
		return nil
	}
	root, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	out := make(map[string]map[int]int, len(root))
	for day, rawHours := range root {
		hm, ok := rawHours.(map[string]any)
		if !ok {
			continue
		}
		day = CleanText(day)
		if day == "" {
			continue
		}
		hours := make(map[int]int, len(hm))
		for hStr, rawPct := range hm {
			// Hour key comes back as a string (e.g. "14") via JSON deser.
			h := atoiSafe(hStr)
			if h < 0 || h > 23 {
				continue
			}
			pct := toInt(rawPct)
			if pct < 0 || pct > 100 {
				continue
			}
			hours[h] = pct
		}
		if len(hours) > 0 {
			out[day] = hours
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func atoiSafe(s string) int {
	n := 0
	if s == "" {
		return -1
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return -1
		}
		n = n*10 + int(r-'0')
		if n > 23 {
			return -1
		}
	}
	return n
}

func toInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	}
	return -1
}
