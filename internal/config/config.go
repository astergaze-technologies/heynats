package config

import (
	"flag"
	"fmt"
	"log/slog"
)

type Config struct {
	HTTPAddr  string
	LogLevel  slog.Level
	LogFormat string
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
	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	cfg := Config{HTTPAddr: *addr, LogFormat: *format}
	if err := cfg.LogLevel.UnmarshalText([]byte(*level)); err != nil {
		return Config{}, fmt.Errorf("invalid log level %q", *level)
	}
	if cfg.LogFormat != "text" && cfg.LogFormat != "json" {
		return Config{}, fmt.Errorf("invalid log format %q", cfg.LogFormat)
	}
	return cfg, nil
}
