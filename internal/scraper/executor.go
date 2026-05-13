package scraper

import (
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/dnahilman/scrapper-go/internal/config"
	"github.com/dnahilman/scrapper-go/internal/domain"
	"github.com/dnahilman/scrapper-go/internal/queue"
)

// PlaywrightExecutor implements workeragent.Executor by driving Chromium
// via playwright-go. It opens one fresh Session per Execute() call so each
// task gets a clean browser context (no cookie reuse across kelurahan).
type PlaywrightExecutor struct {
	cfg *config.WorkerConfig
	log *zerolog.Logger
}

func NewPlaywrightExecutor(cfg *config.WorkerConfig, log *zerolog.Logger) *PlaywrightExecutor {
	return &PlaywrightExecutor{cfg: cfg, log: log}
}

func (e *PlaywrightExecutor) Execute(ctx context.Context, task *queue.ClaimedTask) ([]domain.PlacePayload, error) {
	if task == nil {
		return nil, errors.New("nil task")
	}
	log := e.log.With().
		Str("task", task.TaskID.String()).
		Str("keyword", task.Keyword).
		Str("kelurahan", task.KelurahanName).
		Logger()

	sess, err := NewSession(SessionConfig{Headless: e.cfg.Headless})
	if err != nil {
		return nil, fmt.Errorf("new session: %w", err)
	}
	defer sess.Close()

	page, err := sess.NewPage()
	if err != nil {
		return nil, fmt.Errorf("new page: %w", err)
	}
	defer func() {
		_ = page.Close()
	}()

	urls, err := Search(ctx, page, task.Keyword,
		task.KelurahanName, task.KecamatanName, task.CityName,
		0, // no per-task URL cap by default
		e.cfg.MinDelaySec, e.cfg.MaxDelaySec)
	if err != nil {
		if errors.Is(err, ErrCaptcha) {
			log.Warn().Msg("captcha hit during search; aborting task")
		}
		return nil, fmt.Errorf("search: %w", err)
	}
	log.Info().Int("urls", len(urls)).Msg("search done")

	out := make([]domain.PlacePayload, 0, len(urls))
	for i, u := range urls {
		if ctx.Err() != nil {
			return out, ctx.Err()
		}
		p, err := ScrapePlace(ctx, page, u, e.cfg.MinDelaySec, e.cfg.MaxDelaySec)
		if err != nil {
			if errors.Is(err, ErrCaptcha) {
				log.Warn().Msg("captcha hit during place detail; aborting task")
				return out, err
			}
			log.Warn().Err(err).Str("url", u).Msg("place skipped")
			continue
		}
		if p == nil || p.Title == "" {
			continue
		}
		out = append(out, *p)
		if (i+1)%5 == 0 {
			log.Debug().Int("done", i+1).Int("total", len(urls)).Msg("progress")
		}
	}
	log.Info().Int("places", len(out)).Msg("execute done")
	return out, nil
}
