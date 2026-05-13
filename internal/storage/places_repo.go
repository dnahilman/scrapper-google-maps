package storage

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/dnahilman/scrapper-go/internal/domain"
)

type PlacesRepo struct{ db *gorm.DB }

func NewPlacesRepo(db *gorm.DB) *PlacesRepo { return &PlacesRepo{db: db} }

// Upsert a place by place_id (Google's natural key).
func (r *PlacesRepo) Upsert(ctx context.Context, p *domain.Place) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "place_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"task_id", "kelurahan_id", "keyword",
			"data_id", "cid",
			"title", "categories", "category", "address", "complete_address",
			"open_hours", "popular_times",
			"website", "phone", "plus_code",
			"review_count", "review_rating", "reviews_per_rating",
			"latitude", "longitude",
			"status", "description", "reviews_link", "thumbnail", "timezone", "price_range",
			"images", "reservations", "order_online", "menu", "owner", "about", "emails",
			"scraped_at",
		}),
	}).Create(p).Error
}

func (r *PlacesRepo) Get(ctx context.Context, placeID string) (*domain.Place, error) {
	var p domain.Place
	err := r.db.WithContext(ctx).First(&p, "place_id = ?", placeID).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PlacesRepo) ListByKeyword(ctx context.Context, keyword string, kelurahanID *uuid.UUID, limit, offset int) ([]domain.Place, error) {
	q := r.db.WithContext(ctx).Order("scraped_at DESC")
	if keyword != "" {
		q = q.Where("keyword = ?", keyword)
	}
	if kelurahanID != nil {
		q = q.Where("kelurahan_id = ?", *kelurahanID)
	}
	if limit > 0 {
		q = q.Limit(limit)
	}
	if offset > 0 {
		q = q.Offset(offset)
	}
	var out []domain.Place
	err := q.Find(&out).Error
	return out, err
}

func (r *PlacesRepo) CountByTask(ctx context.Context, taskID uuid.UUID) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&domain.Place{}).Where("task_id = ?", taskID).Count(&n).Error
	return n, err
}
