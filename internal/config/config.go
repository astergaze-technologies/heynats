package config

import (
	"flag"
	"fmt"
	"log/slog"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr  string
	LogLevel  slog.Level
	LogFormat string
	// MetricsInterval is how often each connection samples metrics; 0 = off.
	MetricsInterval time.Duration
}

// Load reads env vars, then command-line flags (flags win).
func Load(args []string, getenv func(string) string) (Config, error) {
	env := func(key, def string) string {
		if v := getenv(key); v != "" {
			return v
		}
		return def
	}

	fs := flag.NewFlagSet("heynats", flag.ContinueOnError)
	addr := fs.String("addr", env("HEYNATS_HTTP_ADDR", ":5000"), "HTTP listen address")
	level := fs.String("log-level", env("HEYNATS_LOG_LEVEL", "info"), "debug, info, warn or error")
	format := fs.String("log-format", env("HEYNATS_LOG_FORMAT", "text"), "text or json")
	interval := fs.Duration("metrics-interval", 0, "metrics sampling interval, 1s to 1m")
	enabled := fs.String("metrics-enabled", env("HEYNATS_METRICS_ENABLED", "true"), "sample metrics for charts")
	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	cfg := Config{HTTPAddr: *addr, LogFormat: *format, MetricsInterval: *interval}
	if cfg.MetricsInterval == 0 {
		d, err := time.ParseDuration(env("HEYNATS_METRICS_INTERVAL", "5s"))
		if err != nil {
			return Config{}, fmt.Errorf("invalid metrics interval: %w", err)
		}
		cfg.MetricsInterval = d
	}
	if cfg.MetricsInterval < time.Second || cfg.MetricsInterval > time.Minute {
		return Config{}, fmt.Errorf("metrics interval %s must be between 1s and 1m", cfg.MetricsInterval)
	}
	on, err := strconv.ParseBool(*enabled)
	if err != nil {
		return Config{}, fmt.Errorf("invalid metrics enabled %q", *enabled)
	}
	if !on {
		cfg.MetricsInterval = 0
	}
	if err := cfg.LogLevel.UnmarshalText([]byte(*level)); err != nil {
		return Config{}, fmt.Errorf("invalid log level %q", *level)
	}
	if cfg.LogFormat != "text" && cfg.LogFormat != "json" {
		return Config{}, fmt.Errorf("invalid log format %q", cfg.LogFormat)
	}
	return cfg, nil
}
