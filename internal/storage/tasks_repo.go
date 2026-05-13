package storage

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/dnahilman/scrapper-go/internal/domain"
)

type TasksRepo struct{ db *gorm.DB }

func NewTasksRepo(db *gorm.DB) *TasksRepo { return &TasksRepo{db: db} }

func (r *TasksRepo) CreateMany(ctx context.Context, tasks []domain.Task) error {
	if len(tasks) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(tasks, 500).Error
}

func (r *TasksRepo) List(ctx context.Context, jobID *uuid.UUID, status string, limit int) ([]domain.Task, error) {
	q := r.db.WithContext(ctx).Preload("Kelurahan").Preload("Worker").Order("enqueued_at ASC")
	if jobID != nil {
		q = q.Where("job_id = ?", *jobID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if limit > 0 {
		q = q.Limit(limit)
	}
	var out []domain.Task
	err := q.Find(&out).Error
	return out, err
}

func (r *TasksRepo) Get(ctx context.Context, id uuid.UUID) (*domain.Task, error) {
	var t domain.Task
	err := r.db.WithContext(ctx).Preload("Kelurahan").Preload("Worker").First(&t, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *TasksRepo) Heartbeat(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&domain.Task{}).
		Where("id = ? AND status = 'in_progress'", id).
		Update("last_heartbeat", time.Now()).Error
}

func (r *TasksRepo) MarkDone(ctx context.Context, id uuid.UUID, placesCount int, resultPath string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&domain.Task{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":       domain.TaskStatusDone,
			"completed_at": now,
			"places_count": placesCount,
			"result_path":  resultPath,
			"last_error":   "",
		}).Error
}

// MarkFailed records the error and either re-queues (with backoff) if attempts remain,
// or marks the task as permanently failed.
func (r *TasksRepo) MarkFailed(ctx context.Context, id uuid.UUID, errMsg string) error {
	return r.db.WithContext(ctx).Exec(`
		UPDATE tasks SET
			status = CASE WHEN attempt < max_attempts THEN 'queued' ELSE 'failed' END,
			worker_id = NULL,
			last_error = ?,
			visible_after = NOW() + (attempt * INTERVAL '1 minute'),
			completed_at = CASE WHEN attempt >= max_attempts THEN NOW() ELSE NULL END
		WHERE id = ?
	`, errMsg, id).Error
}

func (r *TasksRepo) Reset(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&domain.Task{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":         domain.TaskStatusQueued,
			"worker_id":      nil,
			"attempt":        0,
			"visible_after":  time.Now(),
			"last_heartbeat": nil,
			"last_error":     "",
			"started_at":     nil,
			"completed_at":   nil,
		}).Error
}

func (r *TasksRepo) CountByStatus(ctx context.Context, jobID uuid.UUID) (map[string]int, error) {
	rows, err := r.db.WithContext(ctx).Raw(
		`SELECT status, COUNT(*) FROM tasks WHERE job_id = ? GROUP BY status`, jobID).Rows()
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
