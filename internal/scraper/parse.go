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
	reDataIDHex    = regexp.MustCompile(`(0x[0-9a-fA-F]+:0x[0-9a-fA-F]+)`)
	reCIDHexTail   = regexp.MustCompile(`0x[0-9a-fA-F]+:0x([0-9a-fA-F]+)`)
	reCIDQuery     = regexp.MustCompile(`[?&]cid=(\d+)`)
	reFTID         = regexp.MustCompile(`!1[678]s([^!]+)`)

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

// ExtractDataID returns the hex-form "0x...:0x..." identifier from a Google
// Maps place URL, or "" when absent. This matches gosom's `data_id` field.
func ExtractDataID(url string) string {
	if m := reDataIDHex.FindStringSubmatch(url); len(m) >= 2 {
		return m[1]
	}
	return ""
}

// ExtractCID returns Google's numeric Customer ID for the place. The CID is
// the second hex group of the data_id ("0x...:0xCID"), converted to decimal.
// Some URLs also expose it as `?cid=N` directly — we accept either form.
// Returns "" when none was found.
func ExtractCID(url string) string {
	if m := reCIDQuery.FindStringSubmatch(url); len(m) >= 2 {
		return m[1]
	}
	if m := reCIDHexTail.FindStringSubmatch(url); len(m) >= 2 {
		// Decode the hex tail into decimal.
		raw := strings.TrimPrefix(strings.ToLower(m[1]), "0x")
		var n uint64
		for _, c := range raw {
			n <<= 4
			switch {
			case c >= '0' && c <= '9':
				n |= uint64(c - '0')
			case c >= 'a' && c <= 'f':
				n |= uint64(c-'a') + 10
			default:
				return ""
			}
		}
		return strconv.FormatUint(n, 10)
	}
	return ""
}

// ExtractFTID returns Google's Feature/Knowledge-graph ID (e.g. "/g/11rkkkdf95")
// from a Maps URL — typically present as `!16s` or `!17s` in the data param.
// Useful for de-duplicating the same place across multiple URL formats.
func ExtractFTID(url string) string {
	if m := reFTID.FindStringSubmatch(url); len(m) >= 2 {
		// URL-decode "%2F" → "/" for the common path-style FTID.
		s := strings.ReplaceAll(m[1], "%2F", "/")
		s = strings.ReplaceAll(s, "%2f", "/")
		return s
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

// ParseAgeDays converts Google Maps relative time strings into approximate
// age in days. Handles Indonesian + English forms, including the no-number
// "seminggu lalu" / "a week ago" / "sebulan lalu" / "setahun lalu" variants.
// Returns -1 when nothing matches.
func ParseAgeDays(timeStr string) int {
	if timeStr == "" {
		return -1
	}
	s := strings.ToLower(strings.TrimSpace(timeStr))
	s = reEditedPrefix.ReplaceAllString(s, "")
	if strings.Contains(s, "baru saja") || strings.Contains(s, "just now") ||
		strings.Contains(s, "hari ini") || strings.Contains(s, "today") {
		return 0
	}
	if strings.Contains(s, "kemarin") || strings.Contains(s, "yesterday") {
		return 1
	}
	// No-number variants ("seminggu lalu" = "a week ago" = implicit n=1).
	for k, v := range map[string]int{
		"setahun": 365, "a year": 365, "one year": 365,
		"sebulan": 30, "a month": 30, "one month": 30,
		"seminggu": 7, "a week": 7, "one week": 7,
		"sehari": 1, "a day": 1, "one day": 1,
		"sejam": 1, "an hour": 1, "one hour": 1,
	} {
		if strings.Contains(s, k) {
			return v
		}
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
