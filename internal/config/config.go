// Package config loads runtime settings from environment variables.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Addr         string
	DBPath       string
	Password     string
	CookieSecure bool
	SessionTTL   time.Duration
}

// Load reads the configuration shared by all subcommands. It does not
// validate settings that only some subcommands need (see RequireServe).
func Load() (Config, error) {
	c := Config{
		Addr:     env("TAI_ADDR", ":8080"),
		DBPath:   env("TAI_DB_PATH", "tai.db"),
		Password: os.Getenv("TAI_PASSWORD"),
	}

	var err error
	if c.CookieSecure, err = strconv.ParseBool(env("TAI_COOKIE_SECURE", "false")); err != nil {
		return c, fmt.Errorf("TAI_COOKIE_SECURE: %w", err)
	}
	if c.SessionTTL, err = time.ParseDuration(env("TAI_SESSION_TTL", "720h")); err != nil {
		return c, fmt.Errorf("TAI_SESSION_TTL: %w", err)
	}
	if c.SessionTTL <= 0 {
		return c, errors.New("TAI_SESSION_TTL must be positive")
	}
	return c, nil
}

// RequireServe validates the settings needed to run the HTTP server.
func (c Config) RequireServe() error {
	if len(c.Password) < 8 {
		return errors.New("TAI_PASSWORD must be set and at least 8 characters long")
	}
	return nil
}

func env(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}
