package main

import (
	"context"
	"os/signal"
	"syscall"
	"time"

	"github.com/dnahilman/scrapper-go/internal/config"
	"github.com/dnahilman/scrapper-go/internal/logger"
	"github.com/dnahilman/scrapper-go/internal/version"
	"github.com/dnahilman/scrapper-go/internal/workeragent"
)

func main() {
	cfg, err := config.LoadWorker()
	if err != nil {
		panic(err)
	}
	logger.Setup(cfg.LogLevel, cfg.LogFormat)
	log := logger.L()
	log.Info().Str("version", version.Version).Str("master", cfg.MasterURL).Int("concurrency", cfg.MaxConcurrency).Msg("worker starting")

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Phase 1 uses NoopExecutor. Phase 3 will swap this for a Playwright-Go scraper.
	exec := &workeragent.NoopExecutor{Delay: 2 * time.Second}

	agent := workeragent.New(cfg, exec)
	if err := agent.Run(ctx); err != nil {
		log.Fatal().Err(err).Msg("agent exited with error")
	}
	log.Info().Msg("worker stopped")
}
