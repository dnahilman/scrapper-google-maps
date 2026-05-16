package config

import (
	"strings"

	"github.com/spf13/viper"
)

type WorkerConfig struct {
	MasterURL          string
	MasterToken        string
	WorkerName         string
	MaxConcurrency     int
	HeartbeatInterval  int
	Headless           bool
	MinDelaySec        int
	MaxDelaySec        int
	MaxReviewsPerPlace int
	MaxReviewAgeDays   int
	SkipEmptyReviews   bool
	SortReviewsNewest  bool
	MaxCaptchaRetry    int
	EnableEmailCrawl   bool
	LogLevel           string
	LogFormat          string
}

func LoadWorker() (*WorkerConfig, error) {
	v := viper.New()
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	v.SetDefault("MASTER_URL", "http://master:8080")
	v.SetDefault("MAX_CONCURRENCY", 2)
	v.SetDefault("HEARTBEAT_INTERVAL_SEC", 15)
	v.SetDefault("HEADLESS", true)
	v.SetDefault("MIN_DELAY_SEC", 10)
	v.SetDefault("MAX_DELAY_SEC", 25)
	v.SetDefault("MAX_REVIEWS_PER_PLACE", 200)
	v.SetDefault("MAX_REVIEW_AGE_DAYS", 730)
	v.SetDefault("SKIP_EMPTY_REVIEWS", true)
	v.SetDefault("SORT_REVIEWS_BY_NEWEST", true)
	v.SetDefault("MAX_CAPTCHA_RETRY", 2)
	v.SetDefault("ENABLE_EMAIL_CRAWL", false)
	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("LOG_FORMAT", "json")

	return &WorkerConfig{
		MasterURL:          v.GetString("MASTER_URL"),
		MasterToken:        v.GetString("MASTER_TOKEN"),
		WorkerName:         v.GetString("WORKER_NAME"),
		MaxConcurrency:     v.GetInt("MAX_CONCURRENCY"),
		HeartbeatInterval:  v.GetInt("HEARTBEAT_INTERVAL_SEC"),
		Headless:           v.GetBool("HEADLESS"),
		MinDelaySec:        v.GetInt("MIN_DELAY_SEC"),
		MaxDelaySec:        v.GetInt("MAX_DELAY_SEC"),
		MaxReviewsPerPlace: v.GetInt("MAX_REVIEWS_PER_PLACE"),
		MaxReviewAgeDays:   v.GetInt("MAX_REVIEW_AGE_DAYS"),
		SkipEmptyReviews:   v.GetBool("SKIP_EMPTY_REVIEWS"),
		SortReviewsNewest:  v.GetBool("SORT_REVIEWS_BY_NEWEST"),
		MaxCaptchaRetry:    v.GetInt("MAX_CAPTCHA_RETRY"),
		EnableEmailCrawl:   v.GetBool("ENABLE_EMAIL_CRAWL"),
		LogLevel:           v.GetString("LOG_LEVEL"),
		LogFormat:          v.GetString("LOG_FORMAT"),
	}, nil
}
