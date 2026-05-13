package storage

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/dnahilman/scrapper-go/internal/domain"
)

type KelurahanRepo struct{ db *gorm.DB }

func NewKelurahanRepo(db *gorm.DB) *KelurahanRepo { return &KelurahanRepo{db: db} }

func (r *KelurahanRepo) UpsertMany(ctx context.Context, items []domain.Kelurahan) error {
	if len(items) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "emsifa_village_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "kecamatan_name", "emsifa_district_id", "city_id"}),
	}).CreateInBatches(items, 500).Error
}

func (r *KelurahanRepo) ListByCity(ctx context.Context, cityID uuid.UUID, search string) ([]domain.Kelurahan, error) {
	q := r.db.WithContext(ctx).Where("city_id = ?", cityID)
	if search != "" {
		q = q.Where("LOWER(name) LIKE LOWER(?)", "%"+search+"%")
	}
	var out []domain.Kelurahan
	err := q.Order("kecamatan_name ASC, name ASC").Find(&out).Error
	return out, err
}

func (r *KelurahanRepo) FindByCityAndNames(ctx context.Context, cityID uuid.UUID, names []string) ([]domain.Kelurahan, error) {
	var out []domain.Kelurahan
	err := r.db.WithContext(ctx).
		Where("city_id = ? AND name IN ?", cityID, names).
		Find(&out).Error
	return out, err
}

func (r *KelurahanRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Kelurahan, error) {
	var k domain.Kelurahan
	err := r.db.WithContext(ctx).First(&k, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &k, nil
}

func (r *KelurahanRepo) CountByCity(ctx context.Context, cityID uuid.UUID) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&domain.Kelurahan{}).Where("city_id = ?", cityID).Count(&n).Error
	return n, err
}
