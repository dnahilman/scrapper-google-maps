package storage

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/dnahilman/scrapper-go/internal/domain"
)

type CitiesRepo struct{ db *gorm.DB }

func NewCitiesRepo(db *gorm.DB) *CitiesRepo { return &CitiesRepo{db: db} }

func (r *CitiesRepo) Upsert(ctx context.Context, c *domain.City) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "emsifa_regency_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "slug", "province_name", "emsifa_province_id", "updated_at"}),
	}).Create(c).Error
}

func (r *CitiesRepo) List(ctx context.Context) ([]domain.City, error) {
	var out []domain.City
	err := r.db.WithContext(ctx).Order("name ASC").Find(&out).Error
	return out, err
}

func (r *CitiesRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.City, error) {
	var c domain.City
	err := r.db.WithContext(ctx).First(&c, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *CitiesRepo) GetBySlug(ctx context.Context, slug string) (*domain.City, error) {
	var c domain.City
	err := r.db.WithContext(ctx).First(&c, "slug = ?", slug).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *CitiesRepo) Count(ctx context.Context) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&domain.City{}).Count(&n).Error
	return n, err
}
