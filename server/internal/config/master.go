package config

import (
	"strings"

	"github.com/spf13/viper"
)

type MasterConfig struct {
	HTTPAddr       string
	LogLevel       string
	LogFormat      string
	DatabaseURL    string
	MasterToken    string
	EmsifaBaseURL  string
	CORSOrigins    []string
	ReaperInterval int // seconds
	DeadAfter      int // minutes
}

func LoadMaster() (*MasterConfig, error) {
	v := viper.New()
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	v.SetDefault("HTTP_ADDR", ":8080")
	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("LOG_FORMAT", "json")
	v.SetDefault("EMSIFA_BASE_URL", "https://emsifa.github.io/api-wilayah-indonesia/api")
	v.SetDefault("REAPER_INTERVAL_SEC", 30)
	v.SetDefault("DEAD_AFTER_MIN", 2)
	v.SetDefault("CORS_ORIGINS", "*")

	return &MasterConfig{
		HTTPAddr:       v.GetString("HTTP_ADDR"),
		LogLevel:       v.GetString("LOG_LEVEL"),
		LogFormat:      v.GetString("LOG_FORMAT"),
		DatabaseURL:    v.GetString("DATABASE_URL"),
		MasterToken:    v.GetString("MASTER_TOKEN"),
		EmsifaBaseURL:  v.GetString("EMSIFA_BASE_URL"),
		CORSOrigins:    strings.Split(v.GetString("CORS_ORIGINS"), ","),
		ReaperInterval: v.GetInt("REAPER_INTERVAL_SEC"),
		DeadAfter:      v.GetInt("DEAD_AFTER_MIN"),
	}, nil
}
