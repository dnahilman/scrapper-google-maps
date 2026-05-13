package storage

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/dnahilman/scrapper-go/internal/domain"
)

type WorkersRepo struct{ db *gorm.DB }

func NewWorkersRepo(db *gorm.DB) *WorkersRepo { return &WorkersRepo{db: db} }

// Register inserts new or updates by unique name.
func (r *WorkersRepo) Register(ctx context.Context, w *domain.Worker) error {
	now := time.Now()
	w.RegisteredAt = now
	w.Status = domain.WorkerStatusOnline
	w.LastHeartbeat = &now
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "name"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"hostname", "ip_addr", "max_concurrency", "capabilities", "status", "last_heartbeat", "metadata",
		}),
	}).Create(w).Error
}

func (r *WorkersRepo) Heartbeat(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&domain.Worker{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"last_heartbeat": time.Now(),
			"status":         domain.WorkerStatusOnline,
		}).Error
}

func (r *WorkersRepo) List(ctx context.Context) ([]domain.Worker, error) {
	var out []domain.Worker
	err := r.db.WithContext(ctx).Order("registered_at DESC").Find(&out).Error
	return out, err
}

func (r *WorkersRepo) SetStatus(ctx context.Context, id uuid.UUID, status domain.WorkerStatus) error {
	return r.db.WithContext(ctx).Model(&domain.Worker{}).
		Where("id = ?", id).
		Update("status", status).Error
}

func (r *WorkersRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.Worker{}, "id = ?", id).Error
}

// MarkOfflineStale sets stale workers to offline based on heartbeat age.
func (r *WorkersRepo) MarkOfflineStale(ctx context.Context, threshold time.Duration) (int64, error) {
	cutoff := time.Now().Add(-threshold)
	res := r.db.WithContext(ctx).Model(&domain.Worker{}).
		Where("status = ? AND (last_heartbeat IS NULL OR last_heartbeat < ?)", domain.WorkerStatusOnline, cutoff).
		Update("status", domain.WorkerStatusOffline)
	return res.RowsAffected, res.Error
}
