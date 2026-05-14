package emsifa

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/dnahilman/scrapper-go/internal/domain"
	"github.com/dnahilman/scrapper-go/internal/logger"
	"github.com/dnahilman/scrapper-go/internal/storage"
)

type Seeder struct {
	emsifa     *Client
	citiesRepo *storage.CitiesRepo
	kelRepo    *storage.KelurahanRepo
}

func NewSeeder(c *Client, cr *storage.CitiesRepo, kr *storage.KelurahanRepo) *Seeder {
	return &Seeder{emsifa: c, citiesRepo: cr, kelRepo: kr}
}

// SyncAllCities fetches every province + regency from emsifa and upserts them as cities.
func (s *Seeder) SyncAllCities(ctx context.Context) (int, error) {
	log := logger.L().With().Str("svc", "emsifa.seeder").Logger()
	provs, err := s.emsifa.Provinces(ctx)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, p := range provs {
		regs, err := s.emsifa.Regencies(ctx, p.ID)
		if err != nil {
			log.Warn().Err(err).Str("province", p.Name).Msg("failed to fetch regencies")
			continue
		}
		for _, r := range regs {
			city := &domain.City{
				EmsifaRegencyID:  r.ID,
				EmsifaProvinceID: p.ID,
				Name:             r.Name,
				Slug:             slugify(r.Name),
				ProvinceName:     p.Name,
			}
			if err := s.citiesRepo.Upsert(ctx, city); err != nil {
				log.Warn().Err(err).Str("regency", r.Name).Msg("upsert city failed")
				continue
			}
			count++
		}
	}
	return count, nil
}

// SyncKelurahan fetches districts + villages for the given city and upserts kelurahan.
func (s *Seeder) SyncKelurahan(ctx context.Context, city *domain.City) (int, error) {
	districts, err := s.emsifa.Districts(ctx, city.EmsifaRegencyID)
	if err != nil {
		return 0, fmt.Errorf("districts: %w", err)
	}
	var batch []domain.Kelurahan
	for _, d := range districts {
		villages, err := s.emsifa.Villages(ctx, d.ID)
		if err != nil {
			return 0, fmt.Errorf("villages of %s: %w", d.Name, err)
		}
		for _, v := range villages {
			batch = append(batch, domain.Kelurahan{
				CityID:           city.ID,
				EmsifaVillageID:  v.ID,
				EmsifaDistrictID: d.ID,
				Name:             v.Name,
				KecamatanName:    d.Name,
				Code:             v.ID,
			})
		}
	}
	if err := s.kelRepo.UpsertMany(ctx, batch); err != nil {
		return 0, err
	}
	return len(batch), nil
}

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

// slugify converts a city name to a URL-safe slug, preserving the kota/kab
// prefix so "Kota Bandung" and "Kabupaten Bandung" produce distinct slugs
// ("kota-bandung" vs "kab-bandung"). Other adminstrative prefixes pass
// through unchanged.
func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	prefix := ""
	switch {
	case strings.HasPrefix(s, "kabupaten "):
		prefix = "kab-"
		s = strings.TrimPrefix(s, "kabupaten ")
	case strings.HasPrefix(s, "kota "):
		prefix = "kota-"
		s = strings.TrimPrefix(s, "kota ")
	}
	s = slugRe.ReplaceAllString(s, "-")
	return strings.Trim(prefix+s, "-")
}
