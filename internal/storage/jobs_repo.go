package storage

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/dnahilman/scrapper-go/internal/domain"
)

type JobsRepo struct{ db *gorm.DB }

func NewJobsRepo(db *gorm.DB) *JobsRepo { return &JobsRepo{db: db} }

func (r *JobsRepo) Create(ctx context.Context, j *domain.Job) error {
	return r.db.WithContext(ctx).Create(j).Error
}

func (r *JobsRepo) Get(ctx context.Context, id uuid.UUID) (*domain.Job, error) {
	var j domain.Job
	err := r.db.WithContext(ctx).Preload("City").First(&j, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &j, nil
}

func (r *JobsRepo) List(ctx context.Context, status string, limit int) ([]domain.Job, error) {
	q := r.db.WithContext(ctx).Preload("City").Order("created_at DESC")
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if limit > 0 {
		q = q.Limit(limit)
	}
	var out []domain.Job
	err := q.Find(&out).Error
	return out, err
}

func (r *JobsRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.JobStatus) error {
	updates := map[string]any{"status": status}
	switch status {
	case domain.JobStatusRunning:
		updates["started_at"] = time.Now()
	case domain.JobStatusCompleted, domain.JobStatusFailed, domain.JobStatusCancelled:
		updates["completed_at"] = time.Now()
	}
	return r.db.WithContext(ctx).Model(&domain.Job{}).Where("id = ?", id).Updates(updates).Error
}

// RefreshCounters recomputes done/failed counts and auto-completes job when all tasks are terminal.
func (r *JobsRepo) RefreshCounters(ctx context.Context, jobID uuid.UUID) error {
	return r.db.WithContext(ctx).Exec(`
		UPDATE jobs j SET
			done_count = (SELECT COUNT(*) FROM tasks t WHERE t.job_id = j.id AND t.status = 'done'),
			failed_count = (SELECT COUNT(*) FROM tasks t WHERE t.job_id = j.id AND t.status = 'failed'),
			status = CASE
				WHEN (SELECT COUNT(*) FROM tasks t WHERE t.job_id = j.id AND t.status IN ('queued','in_progress')) = 0
				 AND j.total_tasks > 0
				THEN 'completed'
				WHEN j.started_at IS NOT NULL THEN 'running'
				ELSE j.status
			END,
			completed_at = CASE
				WHEN (SELECT COUNT(*) FROM tasks t WHERE t.job_id = j.id AND t.status IN ('queued','in_progress')) = 0
				 AND j.total_tasks > 0
				THEN NOW()
				ELSE j.completed_at
			END
		WHERE j.id = ?
	`, jobID).Error
}
