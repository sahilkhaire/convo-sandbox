package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port              string
	DatabaseURL       string
	MediaStoragePath  string
	EnableDataReset   bool
	CORSOrigins       []string
	DefaultWebhookURL string
	JWTSecret         string
	JWTExpiryHours    int
}

func Load() Config {
	return Config{
		Port:              getEnv("PORT", "8080"),
		DatabaseURL:       getEnv("DATABASE_URL", "postgres://simulator:simulator@localhost:5433/messaging_sim?sslmode=disable"),
		MediaStoragePath:  getEnv("MEDIA_STORAGE_PATH", "/data/media"),
		EnableDataReset:   getEnvBool("ENABLE_DATA_RESET", true),
		CORSOrigins:       splitCSV(getEnv("CORS_ORIGIN", "http://localhost:3000")),
		DefaultWebhookURL: getEnv("DEFAULT_WEBHOOK_URL", ""),
		JWTSecret:         getEnv("JWT_SECRET", ""),
		JWTExpiryHours:    getEnvInt("JWT_EXPIRY_HOURS", 24),
	}
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}
