package scraper

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

var (
	reRating       = regexp.MustCompile(`(\d+[.,]\d+|\d+)`)
	reReviewCount  = regexp.MustCompile(`([\d.,]+)`)
	rePlaceIDBang  = regexp.MustCompile(`!1s([^!]+)`)
	rePlaceIDQuery = regexp.MustCompile(`placeid=([^&]+)`)
	reCoordsBang   = regexp.MustCompile(`!3d(-?\d+\.\d+)!4d(-?\d+\.\d+)`)
	reCoordsAt     = regexp.MustCompile(`@(-?\d+\.\d+),(-?\d+\.\d+)`)

	reEditedPrefix = regexp.MustCompile(`(?i)^diedit\s+`)
)

// Material Symbols icon glyphs live in the Unicode Private Use Area.
const (
	privateUseStart rune = 0xE000
	privateUseEnd   rune = 0xF8FF
)

// CleanText strips whitespace + Material Symbols icon glyphs (Private Use Area).
func CleanText(s string) string {
	if s == "" {
		return ""
	}
	b := strings.Builder{}
	for _, r := range s {
		if r >= privateUseStart && r <= privateUseEnd {
			continue
		}
		b.WriteRune(r)
	}
	fields := strings.Fields(b.String())
	return strings.Join(fields, " ")
}

// ParseRating parses "4,8" / "4.8" / "4,8 stars" → 4.8.
func ParseRating(s string) (float64, bool) {
	if s == "" {
		return 0, false
	}
	m := reRating.FindStringSubmatch(s)
	if len(m) < 2 {
		return 0, false
	}
	v := strings.ReplaceAll(m[1], ",", ".")
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, false
	}
	return f, true
}

// ParseReviewCount handles "1.234 ulasan", "(1,234)", "1234 reviews".
func ParseReviewCount(s string) (int, bool) {
	if s == "" {
		return 0, false
	}
	m := reReviewCount.FindStringSubmatch(s)
	if len(m) < 2 {
		return 0, false
	}
	clean := strings.NewReplacer(".", "", ",", "", " ", "").Replace(m[1])
	n, err := strconv.Atoi(clean)
	if err != nil {
		return 0, false
	}
	return n, true
}

// ExtractPlaceID pulls the place ID from a Google Maps URL.
// Matches both `!1s0x...` form (modern) and legacy `placeid=` query param.
func ExtractPlaceID(url string) string {
	if m := rePlaceIDBang.FindStringSubmatch(url); len(m) >= 2 {
		return m[1]
	}
	if m := rePlaceIDQuery.FindStringSubmatch(url); len(m) >= 2 {
		return m[1]
	}
	return ""
}

// ExtractCoords pulls (lat, lng) from a Google Maps URL.
// Matches `!3d{lat}!4d{lng}` first, then falls back to `@lat,lng`.
func ExtractCoords(url string) (lat, lng float64, ok bool) {
	if m := reCoordsBang.FindStringSubmatch(url); len(m) >= 3 {
		lat, _ = strconv.ParseFloat(m[1], 64)
		lng, _ = strconv.ParseFloat(m[2], 64)
		return lat, lng, true
	}
	if m := reCoordsAt.FindStringSubmatch(url); len(m) >= 3 {
		lat, _ = strconv.ParseFloat(m[1], 64)
		lng, _ = strconv.ParseFloat(m[2], 64)
		return lat, lng, true
	}
	return 0, 0, false
}

// ParseAgeDays converts "X tahun lalu" / "X years ago" / "kemarin" / "yesterday"
// into approximate age in days. Returns -1 on parse failure.
func ParseAgeDays(timeStr string) int {
	if timeStr == "" {
		return -1
	}
	s := strings.ToLower(strings.TrimSpace(timeStr))
	s = reEditedPrefix.ReplaceAllString(s, "")
	if strings.Contains(s, "kemarin") || strings.Contains(s, "yesterday") {
		return 1
	}
	if strings.Contains(s, "baru saja") || strings.Contains(s, "just now") {
		return 0
	}
	digits := strings.Builder{}
	for _, r := range s {
		if unicode.IsDigit(r) {
			digits.WriteRune(r)
		} else if digits.Len() > 0 {
			break
		}
	}
	if digits.Len() == 0 {
		return -1
	}
	n, err := strconv.Atoi(digits.String())
	if err != nil {
		return -1
	}
	type unit struct {
		k string
		d float64
	}
	units := []unit{
		{"tahun", 365}, {"year", 365},
		{"bulan", 30}, {"month", 30},
		{"minggu", 7}, {"week", 7},
		{"hari", 1}, {"day", 1},
		{"jam", 1.0 / 24}, {"hour", 1.0 / 24},
		{"menit", 1.0 / (24 * 60)}, {"minute", 1.0 / (24 * 60)},
	}
	for _, u := range units {
		if strings.Contains(s, u.k) {
			return int(float64(n) * u.d)
		}
	}
	return -1
}
