package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port           string
	DatabaseURL    string
	AllowedOrigins []string
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	IdleTimeout    time.Duration

	// JWTSecret signs session tokens. Required — the API has no other way to
	// authenticate a caller now that Supabase Auth is gone.
	JWTSecret string
	TokenTTL  time.Duration

	// UploadDir holds payment proofs, replacing Supabase Storage.
	UploadDir     string
	MaxUploadSize int64

	// Seed credentials create the first administrator on an empty users table,
	// so a fresh database is reachable without hand-writing a password hash.
	SeedAdminEmail    string
	SeedAdminPassword string
	SeedAdminName     string
}

// Load reads configuration from the environment, first pulling in a .env file
// if present (tiny loader — avoids a dependency just for local DX).
func Load() (Config, error) {
	loadDotEnv(".env")

	cfg := Config{
		Port:           envOr("PORT", "8080"),
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		AllowedOrigins: splitAndTrim(envOr("ALLOWED_ORIGINS", "*")),
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    60 * time.Second,

		JWTSecret: os.Getenv("JWT_SECRET"),
		TokenTTL:  durationOr("TOKEN_TTL", 12*time.Hour),

		UploadDir:     envOr("UPLOAD_DIR", "./uploads"),
		MaxUploadSize: bytesOr("MAX_UPLOAD_SIZE", 5<<20), // 5 MiB

		SeedAdminEmail:    strings.TrimSpace(os.Getenv("SEED_ADMIN_EMAIL")),
		SeedAdminPassword: os.Getenv("SEED_ADMIN_PASSWORD"),
		SeedAdminName:     envOr("SEED_ADMIN_NAME", "Administrator"),
	}

	if cfg.DatabaseURL == "" {
		return cfg, fmt.Errorf("DATABASE_URL wajib diisi (lihat .env.example)")
	}
	if cfg.JWTSecret == "" {
		return cfg, fmt.Errorf("JWT_SECRET wajib diisi — buat dengan: openssl rand -base64 48")
	}
	return cfg, nil
}

// HasSeedAdmin reports whether both seed credentials were supplied. Half a pair
// is a misconfiguration worth surfacing rather than silently ignoring.
func (c Config) HasSeedAdmin() bool {
	return c.SeedAdminEmail != "" && c.SeedAdminPassword != ""
}

func durationOr(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return fallback
	}
	return d
}

func bytesOr(key string, fallback int64) int64 {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// loadDotEnv sets any KEY=VALUE pairs from path that aren't already in the
// environment. Real env vars always win.
func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, value)
		}
	}
}
