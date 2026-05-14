package scraper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/dnahilman/scrapper-go/internal/config"
	"github.com/dnahilman/scrapper-go/internal/domain"
	"github.com/dnahilman/scrapper-go/internal/queue"
)

// jobOptions mirrors the per-job knobs we expect inside ClaimedTask.Options
// (raw JSONB). All fields are optional; zero values fall back to worker env.
type jobOptions struct {
	LimitPerKelurahan  int   `json:"limit_per_kelurahan"`
	MaxReviewsPerPlace *int  `json:"max_reviews_per_place,omitempty"`
	MinReviewAgeDays   *int  `json:"min_review_age_days,omitempty"` // drop reviews newer than N days
	MaxReviewAgeDays   *int  `json:"max_review_age_days,omitempty"` // drop reviews older than N days
	EnableEmailCrawl   *bool `json:"enable_email_crawl,omitempty"`
}

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

	var jobOpts jobOptions
	if len(task.Options) > 0 {
		_ = json.Unmarshal(task.Options, &jobOpts)
	}

	maxReviews := e.cfg.MaxReviewsPerPlace
	if jobOpts.MaxReviewsPerPlace != nil {
		maxReviews = *jobOpts.MaxReviewsPerPlace
	}
	maxReviewAge := e.cfg.MaxReviewAgeDays
	if jobOpts.MaxReviewAgeDays != nil {
		maxReviewAge = *jobOpts.MaxReviewAgeDays
	}
	minReviewAge := 0
	if jobOpts.MinReviewAgeDays != nil {
		minReviewAge = *jobOpts.MinReviewAgeDays
	}
	enableEmails := e.cfg.EnableEmailCrawl
	if jobOpts.EnableEmailCrawl != nil {
		enableEmails = *jobOpts.EnableEmailCrawl
	}

	urls, err := Search(ctx, page, task.Keyword,
		task.KelurahanName, task.KecamatanName, task.CityName,
		jobOpts.LimitPerKelurahan,
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
		p, err := ScrapePlace(ctx, page, u, e.cfg.MinDelaySec, e.cfg.MaxDelaySec, PlaceOptions{
			MaxReviewsPerPlace:  maxReviews,
			MinReviewAgeDays:    minReviewAge,
			MaxReviewAgeDays:    maxReviewAge,
			SortReviewsByNewest: e.cfg.SortReviewsNewest,
			SkipEmptyReviews:    e.cfg.SkipEmptyReviews,
			CityName:            task.CityName,
			KelurahanName:       task.KelurahanName,
			KecamatanName:       task.KecamatanName,
			EnableEmailCrawl:    enableEmails,
		})
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
