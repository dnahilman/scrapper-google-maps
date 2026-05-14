package main

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/playwright-community/playwright-go"

	"github.com/dnahilman/scrapper-go/internal/config"
	"github.com/dnahilman/scrapper-go/internal/logger"
	"github.com/dnahilman/scrapper-go/internal/scraper"
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
	log.Info().
		Str("version", version.Version).
		Str("master", cfg.MasterURL).
		Int("concurrency", cfg.MaxConcurrency).
		Bool("headless", cfg.Headless).
		Msg("worker starting")

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	var exec workeragent.Executor
	if strings.EqualFold(os.Getenv("SCRAPER"), "noop") {
		log.Info().Msg("using NoopExecutor (set SCRAPER!=noop for real Playwright)")
		exec = &workeragent.NoopExecutor{Delay: 2 * time.Second}
	} else {
		// Ensure the Playwright driver + Chromium are present. First boot
		// downloads ~150MB; subsequent boots are no-ops once the cache (or
		// the bundled image at PLAYWRIGHT_BROWSERS_PATH) is in place.
		log.Info().Msg("verifying Playwright driver…")
		if err := playwright.Install(&playwright.RunOptions{
			Browsers:            []string{"chromium"},
			SkipInstallBrowsers: strings.EqualFold(os.Getenv("PLAYWRIGHT_SKIP_BROWSER_INSTALL"), "1"),
		}); err != nil {
			log.Fatal().Err(err).Msg("playwright install failed")
		}
		exec = scraper.NewPlaywrightExecutor(cfg, logger.L())
	}

	agent := workeragent.New(cfg, exec)
	if err := agent.Run(ctx); err != nil {
		log.Fatal().Err(err).Msg("agent exited with error")
	}
	log.Info().Msg("worker stopped")
}
