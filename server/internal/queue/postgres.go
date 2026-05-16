package queue

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/dnahilman/scrapper-go/internal/storage"
)

type PostgresQueue struct {
	db    *gorm.DB
	tasks *storage.TasksRepo
	jobs  *storage.JobsRepo
}

func NewPostgresQueue(db *gorm.DB, tasks *storage.TasksRepo, jobs *storage.JobsRepo) *PostgresQueue {
	return &PostgresQueue{db: db, tasks: tasks, jobs: jobs}
}

// Claim atomically picks the highest-priority queued task using FOR UPDATE SKIP LOCKED.
// Returns nil if the queue is empty.
func (q *PostgresQueue) Claim(ctx context.Context, workerID uuid.UUID) (*ClaimedTask, error) {
	const query = `
WITH next AS (
  SELECT id FROM tasks
  WHERE status = 'queued' AND visible_after <= NOW()
  ORDER BY priority DESC, enqueued_at ASC
  LIMIT 1
  FOR UPDATE SKIP LOCKED
)
UPDATE tasks t SET
  status = 'in_progress',
  worker_id = $1,
  attempt = attempt + 1,
  started_at = NOW(),
  last_heartbeat = NOW()
FROM next
WHERE t.id = next.id
RETURNING
  t.id, t.job_id, t.kelurahan_id, t.attempt, t.max_attempts;
`
	var row struct {
		ID          uuid.UUID
		JobID       uuid.UUID
		KelurahanID uuid.UUID
		Attempt     int
		MaxAttempts int
	}
	err := q.db.WithContext(ctx).Raw(query, workerID).Row().
		Scan(&row.ID, &row.JobID, &row.KelurahanID, &row.Attempt, &row.MaxAttempts)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	// Hydrate with denormalized fields for the worker payload.
	const meta = `
SELECT j.keyword, k.name, k.kecamatan_name, c.name, j.options
FROM jobs j
JOIN kelurahan k ON k.id = $1
JOIN cities c ON c.id = j.city_id
WHERE j.id = $2`
	var keyword, kel, kec, city string
	var opts []byte
	if err := q.db.WithContext(ctx).Raw(meta, row.KelurahanID, row.JobID).
		Row().Scan(&keyword, &kel, &kec, &city, &opts); err != nil {
		return nil, err
	}

	return &ClaimedTask{
		TaskID:        row.ID,
		JobID:         row.JobID,
		Keyword:       keyword,
		KelurahanID:   row.KelurahanID,
		KelurahanName: kel,
		KecamatanName: kec,
		CityName:      city,
		Attempt:       row.Attempt,
		MaxAttempts:   row.MaxAttempts,
		Options:       opts,
	}, nil
}

func (q *PostgresQueue) Ack(ctx context.Context, taskID uuid.UUID, placesCount int, resultPath string) error {
	if err := q.tasks.MarkDone(ctx, taskID, placesCount, resultPath); err != nil {
		return err
	}
	// Find job id and refresh counters
	var jobID uuid.UUID
	if err := q.db.WithContext(ctx).Raw(
		`SELECT job_id FROM tasks WHERE id = ?`, taskID).Row().Scan(&jobID); err != nil {
		return err
	}
	return q.jobs.RefreshCounters(ctx, jobID)
}

func (q *PostgresQueue) Nack(ctx context.Context, taskID uuid.UUID, errMsg string) error {
	if err := q.tasks.MarkFailed(ctx, taskID, errMsg); err != nil {
		return err
	}
	var jobID uuid.UUID
	if err := q.db.WithContext(ctx).Raw(
		`SELECT job_id FROM tasks WHERE id = ?`, taskID).Row().Scan(&jobID); err != nil {
		return err
	}
	return q.jobs.RefreshCounters(ctx, jobID)
}

func (q *PostgresQueue) Heartbeat(ctx context.Context, taskID uuid.UUID) error {
	return q.tasks.Heartbeat(ctx, taskID)
}
