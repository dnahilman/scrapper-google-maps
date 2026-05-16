package workeragent

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/dnahilman/scrapper-go/internal/config"
	"github.com/dnahilman/scrapper-go/internal/domain"
	"github.com/dnahilman/scrapper-go/internal/logger"
	"github.com/dnahilman/scrapper-go/internal/queue"
)

// Executor is the actual scraping engine. The agent calls Execute() per task.
// In Phase 1 we ship NoopExecutor that simulates work; Phase 3 plugs in Playwright-Go.
type Executor interface {
	Execute(ctx context.Context, task *queue.ClaimedTask) ([]domain.PlacePayload, error)
}

type Agent struct {
	cfg    *config.WorkerConfig
	client *Client
	exec   Executor

	workerID uuid.UUID
	wg       sync.WaitGroup
}

func New(cfg *config.WorkerConfig, exec Executor) *Agent {
	return &Agent{
		cfg:    cfg,
		client: NewClient(cfg.MasterURL, cfg.MasterToken),
		exec:   exec,
	}
}

func (a *Agent) Run(ctx context.Context) error {
	log := logger.L()

	hostname, _ := os.Hostname()
	name := a.cfg.WorkerName
	if name == "" {
		name = hostname
	}

	// Register with retries (master may not be ready yet).
	var (
		w   *domain.Worker
		err error
	)
	for attempt := 0; attempt < 60; attempt++ {
		w, err = a.client.Register(ctx, &RegisterRequest{
			Name:           name,
			Hostname:       hostname,
			MaxConcurrency: a.cfg.MaxConcurrency,
		})
		if err == nil {
			break
		}
		log.Warn().Err(err).Int("attempt", attempt+1).Msg("register failed, retrying")
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	if err != nil {
		return fmt.Errorf("register: %w", err)
	}
	a.workerID = w.ID
	log.Info().Str("worker_id", a.workerID.String()).Str("name", w.Name).Msg("worker registered")

	go a.heartbeatLoop(ctx)

	for i := 0; i < a.cfg.MaxConcurrency; i++ {
		a.wg.Add(1)
		go a.slotLoop(ctx, i)
	}

	a.wg.Wait()
	return nil
}

func (a *Agent) heartbeatLoop(ctx context.Context) {
	t := time.NewTicker(time.Duration(a.cfg.HeartbeatInterval) * time.Second)
	defer t.Stop()
	log := logger.L()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := a.client.Heartbeat(ctx, a.workerID); err != nil {
				log.Warn().Err(err).Msg("worker heartbeat failed")
			}
		}
	}
}

func (a *Agent) slotLoop(ctx context.Context, slot int) {
	defer a.wg.Done()
	log := logger.L().With().Int("slot", slot).Logger()
	backoff := time.Second
	for {
		if ctx.Err() != nil {
			return
		}
		ct, err := a.client.Claim(ctx, a.workerID)
		if err != nil {
			log.Warn().Err(err).Msg("claim failed")
			sleepCtx(ctx, backoff)
			backoff = nextBackoff(backoff)
			continue
		}
		if ct == nil {
			sleepCtx(ctx, 5*time.Second)
			backoff = time.Second
			continue
		}
		backoff = time.Second
		a.runTask(ctx, ct, &log)
	}
}

func (a *Agent) runTask(ctx context.Context, ct *queue.ClaimedTask, log *zerolog.Logger) {
	log.Info().
		Str("task", ct.TaskID.String()).
		Str("keyword", ct.Keyword).
		Str("kelurahan", ct.KelurahanName).
		Str("kecamatan", ct.KecamatanName).
		Msg("task claimed")

	taskCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go a.taskHeartbeatLoop(taskCtx, ct.TaskID, log)

	places, err := a.exec.Execute(taskCtx, ct)
	if err != nil {
		log.Error().Err(err).Msg("execute failed")
		if nerr := a.client.Nack(ctx, ct.TaskID, err.Error()); nerr != nil {
			log.Error().Err(nerr).Msg("nack failed")
		}
		return
	}

	if len(places) > 0 {
		if err := a.client.SubmitPlaces(ctx, &SubmitPlacesRequest{
			TaskID:      ct.TaskID,
			KelurahanID: ct.KelurahanID,
			Keyword:     ct.Keyword,
			Places:      places,
		}); err != nil {
			var httpErr *HTTPError
			if errors.As(err, &httpErr) && httpErr.Code == http.StatusGone {
				// Job/task was deleted while we were running — abandon silently.
				log.Warn().Str("task", ct.TaskID.String()).Msg("task gone (job deleted); dropping work")
				return
			}
			log.Error().Err(err).Msg("submit places failed; nacking")
			_ = a.client.Nack(ctx, ct.TaskID, "submit_places: "+err.Error())
			return
		}
	}

	if err := a.client.Ack(ctx, ct.TaskID, len(places), ""); err != nil {
		log.Error().Err(err).Msg("ack failed")
		return
	}
	log.Info().Int("places", len(places)).Str("task", ct.TaskID.String()).Msg("task done")
}

func (a *Agent) taskHeartbeatLoop(ctx context.Context, taskID uuid.UUID, log *zerolog.Logger) {
	t := time.NewTicker(time.Duration(a.cfg.HeartbeatInterval) * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := a.client.TaskHeartbeat(ctx, taskID); err != nil {
				log.Warn().Err(err).Msg("task heartbeat failed")
			}
		}
	}
}

func nextBackoff(d time.Duration) time.Duration {
	d *= 2
	jitter := time.Duration(rand.Int63n(int64(d/4) + 1))
	d += jitter
	if d > 30*time.Second {
		d = 30 * time.Second
	}
	return d
}

func sleepCtx(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}
