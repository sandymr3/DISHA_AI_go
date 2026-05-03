// Package config loads and validates environment variables for the DISHA API server.
package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all environment-sourced configuration values.
type Config struct {
	// GoogleAPIKey is the API key for Google Gemini. Optional — chat falls back to
	// keyword-matched responses when absent.
	GoogleAPIKey string

	// Port is the HTTP port to listen on. Defaults to "8080".
	Port string

	// Env is the deployment environment: "development" or "production".
	// Controls logging verbosity.
	Env string

	// RateLimitRequests is the max chat messages per student per window. Default 20.
	RateLimitRequests int

	// RateLimitWindowHours is the rate limit window duration in hours. Default 1.
	RateLimitWindowHours int

	// CORSAllowedOrigins is the allowed CORS origin. Default "*".
	CORSAllowedOrigins string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() Config {
	// Attempt to load .env file; ignore if not found
	_ = godotenv.Load()

	return Config{
		GoogleAPIKey:         os.Getenv("GOOGLE_API_KEY"),
		Port:                 envOrDefault("PORT", "8080"),
		Env:                  envOrDefault("ENV", "development"),
		RateLimitRequests:    envOrDefaultInt("RATE_LIMIT_REQUESTS", 20),
		RateLimitWindowHours: envOrDefaultInt("RATE_LIMIT_WINDOW_HOURS", 1),
		CORSAllowedOrigins:   envOrDefault("CORS_ALLOWED_ORIGINS", "*"),
	}
}

// IsDevelopment returns true if the server is running in development mode.
func (c Config) IsDevelopment() bool {
	return c.Env == "development"
}

// HasGeminiKey returns true if a Google API key is configured.
func (c Config) HasGeminiKey() bool {
	return c.GoogleAPIKey != ""
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envOrDefaultInt(key string, fallback int) int {
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
