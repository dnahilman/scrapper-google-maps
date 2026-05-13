package scraper

import (
	"regexp"
	"strings"

	"github.com/dnahilman/scrapper-go/internal/domain"
)

// postalRe matches Indonesian postal codes (5 digits).
var postalRe = regexp.MustCompile(`\b(\d{5})\b`)

// ParseCompleteAddress splits an Indonesian address into gosom-style components.
// It uses the supplied task context (kelurahan, kecamatan, city) as hints to
// backfill borough/city/state when they don't appear verbatim in the address.
//
// Example input:
//   "Jl. Tangkuban Perahu No.7, Lebakgede, Kecamatan Coblong, Kota Bandung,
//    Jawa Barat 40132"
// →  Street: "Jl. Tangkuban Perahu No.7"
//    Borough: "Lebakgede"
//    City: "Kota Bandung"
//    PostalCode: "40132"
//    State: "Jawa Barat"
//    Country: "Indonesia"
func ParseCompleteAddress(addr, kelurahan, kecamatan, city string) *domain.CompleteAddress {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return nil
	}

	out := &domain.CompleteAddress{Country: "Indonesia"}

	// Postal code anywhere.
	if m := postalRe.FindString(addr); m != "" {
		out.PostalCode = m
	}

	// City: prefer the task hint, fall back to "Kota X" / "Kabupaten X" tokens.
	if city != "" {
		out.City = city
	} else {
		for _, part := range strings.Split(addr, ",") {
			p := strings.TrimSpace(part)
			pLow := strings.ToLower(p)
			if strings.HasPrefix(pLow, "kota ") || strings.HasPrefix(pLow, "kabupaten ") {
				out.City = p
				break
			}
		}
	}

	// Borough: kelurahan name when known.
	if kelurahan != "" {
		out.Borough = kelurahan
	}

	// State: heuristic from city, or the last token (without postal) of the address.
	if state := stateForCity(city); state != "" {
		out.State = state
	} else {
		parts := strings.Split(addr, ",")
		if len(parts) > 0 {
			last := strings.TrimSpace(parts[len(parts)-1])
			last = strings.TrimSpace(postalRe.ReplaceAllString(last, ""))
			if last != "" && !strings.EqualFold(last, "Indonesia") {
				out.State = last
			}
		}
	}

	// Street: portion before the first kelurahan / kecamatan / city marker.
	out.Street = extractStreet(addr, kelurahan, kecamatan, city)

	return out
}

// extractStreet returns the portion of `addr` before any administrative marker.
// Falls back to the first comma chunk if no marker hits.
func extractStreet(addr, kelurahan, kecamatan, city string) string {
	street := addr
	for _, marker := range []string{kelurahan, kecamatan, city, "Kelurahan", "Kecamatan", "Kota ", "Kabupaten "} {
		if marker == "" {
			continue
		}
		if idx := strings.Index(street, marker); idx > 0 {
			candidate := strings.TrimRight(strings.TrimSpace(street[:idx]), ",")
			if candidate != "" {
				return strings.TrimSpace(candidate)
			}
		}
	}
	if idx := strings.Index(addr, ","); idx > 0 {
		return strings.TrimSpace(addr[:idx])
	}
	return strings.TrimSpace(addr)
}

// stateForCity maps known Indonesian city names → province name.
// Default empty so we can fall back to address-parsing logic.
func stateForCity(city string) string {
	c := strings.ToLower(city)
	switch {
	case containsAny(c, "bandung", "cimahi", "sumedang", "garut", "tasik",
		"cianjur", "bogor", "bekasi", "depok", "karawang", "subang",
		"purwakarta", "indramayu", "majalengka", "kuningan", "ciamis"):
		return "Jawa Barat"
	case containsAny(c, "jakarta"):
		return "DKI Jakarta"
	case containsAny(c, "semarang", "solo", "surakarta", "yogyakarta",
		"magelang", "purwokerto", "tegal", "pekalongan", "kudus", "salatiga"):
		switch {
		case strings.Contains(c, "yogya"):
			return "DI Yogyakarta"
		default:
			return "Jawa Tengah"
		}
	case containsAny(c, "surabaya", "malang", "kediri", "madiun", "jember",
		"sidoarjo", "gresik", "blitar", "pasuruan", "probolinggo"):
		return "Jawa Timur"
	case containsAny(c, "denpasar", "badung", "gianyar", "tabanan", "buleleng"):
		return "Bali"
	case containsAny(c, "medan", "binjai", "deli serdang", "asahan"):
		return "Sumatera Utara"
	case containsAny(c, "padang", "bukittinggi", "payakumbuh"):
		return "Sumatera Barat"
	case containsAny(c, "palembang", "lubuklinggau", "prabumulih"):
		return "Sumatera Selatan"
	case containsAny(c, "makassar", "parepare", "palopo"):
		return "Sulawesi Selatan"
	case containsAny(c, "manado", "bitung", "tomohon"):
		return "Sulawesi Utara"
	case containsAny(c, "balikpapan", "samarinda", "bontang"):
		return "Kalimantan Timur"
	case containsAny(c, "banjarmasin", "banjarbaru"):
		return "Kalimantan Selatan"
	default:
		return ""
	}
}

func containsAny(haystack string, needles ...string) bool {
	for _, n := range needles {
		if strings.Contains(haystack, n) {
			return true
		}
	}
	return false
}
