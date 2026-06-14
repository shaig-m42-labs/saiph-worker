package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port          string
	CoreURL       string
	NATSURL       string
	CheckInterval time.Duration
}

func Load() Config {
	return Config{
		Port:          env("PORT", "8084"),
		CoreURL:       env("CORE_URL", "http://localhost:8082"),
		NATSURL:       env("NATS_URL", "nats://localhost:4222"),
		CheckInterval: time.Duration(envInt("CHECK_INTERVAL_SECONDS", 30)) * time.Second,
	}
}

func env(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func envInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
