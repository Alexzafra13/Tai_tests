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
	Addr   string
	DBPath string
	// AdminUser and AdminPassword create the first administrator on first
	// start; afterwards accounts are managed from the app.
	AdminUser     string
	AdminPassword string
	CookieSecure  bool
	SessionTTL    time.Duration
	// Location is the time zone that defines a "day" in statistics.
	Location *time.Location
}

// Load reads the configuration shared by all subcommands.
func Load() (Config, error) {
	c := Config{
		Addr:          env("TAI_ADDR", ":8080"),
		DBPath:        env("TAI_DB_PATH", "tai.db"),
		AdminUser:     env("TAI_ADMIN_USER", "admin"),
		AdminPassword: os.Getenv("TAI_ADMIN_PASSWORD"),
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
	if c.Location, err = time.LoadLocation(env("TAI_TZ", "Europe/Madrid")); err != nil {
		return c, fmt.Errorf("TAI_TZ: %w", err)
	}
	return c, nil
}

func env(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}
