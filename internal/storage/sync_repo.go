package storage

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/dnahilman/scrapper-go/internal/domain"
)

type SyncRepo struct{ db *gorm.DB }

func NewSyncRepo(db *gorm.DB) *SyncRepo { return &SyncRepo{db: db} }

func (r *SyncRepo) EnsurePending(ctx context.Context, taskID uuid.UUID) error {
	rec := &domain.SyncRecord{TaskID: taskID, Status: domain.SyncStatusPending}
	return r.db.WithContext(ctx).Where("task_id = ?", taskID).
		FirstOrCreate(rec).Error
}

func (r *SyncRepo) Stats(ctx context.Context) (map[string]int, error) {
	rows, err := r.db.WithContext(ctx).Raw(
		`SELECT status, COUNT(*) FROM sync_records GROUP BY status`).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var s string
		var n int
		if err := rows.Scan(&s, &n); err != nil {
			return nil, err
		}
		out[s] = n
	}
	return out, nil
}
