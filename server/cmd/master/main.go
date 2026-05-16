package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dnahilman/scrapper-go/internal/api"
	"github.com/dnahilman/scrapper-go/internal/config"
	"github.com/dnahilman/scrapper-go/internal/emsifa"
	"github.com/dnahilman/scrapper-go/internal/logger"
	"github.com/dnahilman/scrapper-go/internal/logstream"
	"github.com/dnahilman/scrapper-go/internal/queue"
	"github.com/dnahilman/scrapper-go/internal/storage"
	"github.com/dnahilman/scrapper-go/internal/version"
)

func main() {
	cmd := "serve"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	cfg, err := config.LoadMaster()
	if err != nil {
		panic(err)
	}
	logger.Setup(cfg.LogLevel, cfg.LogFormat)
	log := logger.L()
	log.Info().Str("version", version.Version).Str("cmd", cmd).Msg("master starting")

	switch cmd {
	case "serve":
		run(cfg)
	case "migrate":
		if err := storage.RunMigrations(cfg.DatabaseURL, "./migrations"); err != nil {
			log.Fatal().Err(err).Msg("migrate failed")
		}
		log.Info().Msg("migrations applied")
	case "healthcheck":
		runHealthcheck(cfg)
	default:
		log.Fatal().Str("cmd", cmd).Msg("unknown command")
	}
}

func run(cfg *config.MasterConfig) {
	log := logger.L()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Run migrations on boot (idempotent).
	if err := storage.RunMigrations(cfg.DatabaseURL, "./migrations"); err != nil {
		log.Fatal().Err(err).Msg("migrate failed")
	}
	log.Info().Msg("migrations ok")

	db, err := storage.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("db open failed")
	}
	defer storage.Close(db)

	// Wire up repositories and services.
	citiesRepo := storage.NewCitiesRepo(db)
	kelRepo := storage.NewKelurahanRepo(db)
	workersRepo := storage.NewWorkersRepo(db)
	jobsRepo := storage.NewJobsRepo(db)
	tasksRepo := storage.NewTasksRepo(db)
	placesRepo := storage.NewPlacesRepo(db)
	reviewsRepo := storage.NewReviewsRepo(db)

	q := queue.NewPostgresQueue(db, tasksRepo, jobsRepo)
	em := emsifa.New(cfg.EmsifaBaseURL)
	seeder := emsifa.NewSeeder(em, citiesRepo, kelRepo)
	hub := logstream.NewHub()
	logger.AddHubWriter(logstream.NewHubWriter(hub))

	reaper := queue.NewReaper(db, workersRepo, hub,
		time.Duration(cfg.ReaperInterval)*time.Second,
		time.Duration(cfg.DeadAfter)*time.Minute)
	go reaper.Run(ctx)

	deps := &api.Deps{
		Cfg:       cfg,
		DB:        db,
		Cities:    citiesRepo,
		Kelurahan: kelRepo,
		Workers:   workersRepo,
		Jobs:      jobsRepo,
		Tasks:     tasksRepo,
		Places:    placesRepo,
		Reviews:   reviewsRepo,
		Queue:     q,
		Emsifa:    em,
		Seeder:    seeder,
		Hub:       hub,
	}
	r := api.NewRouter(deps)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Info().Str("addr", cfg.HTTPAddr).Msg("http listening")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Err(err).Msg("http server failed")
		}
	}()

	<-ctx.Done()
	log.Info().Msg("shutting down")
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	_ = srv.Shutdown(shutdownCtx)
}

func runHealthcheck(cfg *config.MasterConfig) {
	c := &http.Client{Timeout: 3 * time.Second}
	resp, err := c.Get("http://127.0.0.1" + cfg.HTTPAddr + "/api/v1/health")
	if err != nil || resp.StatusCode != http.StatusOK {
		os.Exit(1)
	}
}
