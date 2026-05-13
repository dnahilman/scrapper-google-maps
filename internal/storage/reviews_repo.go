package storage

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/dnahilman/scrapper-go/internal/domain"
)

type ReviewsRepo struct{ db *gorm.DB }

func NewReviewsRepo(db *gorm.DB) *ReviewsRepo { return &ReviewsRepo{db: db} }

// UpsertMany inserts reviews, ignoring duplicates on (place_id, review_id).
func (r *ReviewsRepo) UpsertMany(ctx context.Context, items []domain.Review) error {
	if len(items) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "place_id"}, {Name: "review_id"}},
		DoNothing: true,
	}).CreateInBatches(items, 500).Error
}

func (r *ReviewsRepo) ListByPlace(ctx context.Context, placeID string, limit int) ([]domain.Review, error) {
	q := r.db.WithContext(ctx).Where("place_id = ?", placeID).Order("created_at DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	var out []domain.Review
	err := q.Find(&out).Error
	return out, err
}
