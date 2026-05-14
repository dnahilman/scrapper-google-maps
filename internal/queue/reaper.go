package queue

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/dnahilman/scrapper-go/internal/logger"
	"github.com/dnahilman/scrapper-go/internal/logstream"
	"github.com/dnahilman/scrapper-go/internal/storage"
)

// Reaper re-queues tasks whose worker has stopped sending heartbeats.
type Reaper struct {
	db        *gorm.DB
	workers   *storage.WorkersRepo
	hub       *logstream.Hub
	interval  time.Duration
	deadAfter time.Duration
}

func NewReaper(db *gorm.DB, workers *storage.WorkersRepo, hub *logstream.Hub, interval, deadAfter time.Duration) *Reaper {
	return &Reaper{db: db, workers: workers, hub: hub, interval: interval, deadAfter: deadAfter}
}

// Run blocks until ctx is cancelled.
func (r *Reaper) Run(ctx context.Context) {
	log := logger.L().With().Str("svc", "reaper").Logger()
	t := time.NewTicker(r.interval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			n, err := r.sweepTasks(ctx)
			if err != nil {
				log.Error().Err(err).Msg("sweep tasks failed")
			} else if n > 0 {
				log.Info().Int64("requeued", n).Msg("reaper requeued dead tasks")
				if r.hub != nil {
					r.hub.Broadcast(logstream.EventTaskRequeued, map[string]any{"count": n})
				}
			}
			m, err := r.workers.MarkOfflineStale(ctx, r.deadAfter)
			if err != nil {
				log.Error().Err(err).Msg("mark offline failed")
			} else if m > 0 {
				log.Info().Int64("workers", m).Msg("marked stale workers offline")
				if r.hub != nil {
					r.hub.Broadcast(logstream.EventWorkerOffline, map[string]any{"count": m})
				}
			}
		}
	}
}

// sweepTasks finds in_progress tasks whose last_heartbeat is older than deadAfter
// and re-queues them (with exponential backoff) if attempts remain.
func (r *Reaper) sweepTasks(ctx context.Context) (int64, error) {
	const sql = `
UPDATE tasks SET
  status = CASE WHEN attempt < max_attempts THEN 'queued' ELSE 'failed' END,
  worker_id = NULL,
  visible_after = NOW() + (attempt * INTERVAL '1 minute'),
  last_error = COALESCE(NULLIF(last_error, ''), 'worker timeout'),
  completed_at = CASE WHEN attempt >= max_attempts THEN NOW() ELSE NULL END
WHERE status = 'in_progress'
  AND (last_heartbeat IS NULL OR last_heartbeat < NOW() - $1::interval);
`
	intervalSpec := r.deadAfter.String() // e.g. "2m0s"
	res := r.db.WithContext(ctx).Exec(sql, intervalSpec)
	return res.RowsAffected, res.Error
}
