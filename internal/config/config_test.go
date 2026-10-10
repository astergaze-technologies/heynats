package config

import (
	"log/slog"
	"testing"
)

func envOf(vars map[string]string) func(string) string {
	return func(key string) string { return vars[key] }
}

func TestLoad(t *testing.T) {
	tests := []struct {
		name string
		args []string
		env  map[string]string
		want Config
	}{
		{"defaults", nil, nil, Config{":5000", slog.LevelInfo, "text"}},
		{"env", nil, map[string]string{
			"HEYNATS_HTTP_ADDR":  ":8080",
			"HEYNATS_LOG_LEVEL":  "debug",
			"HEYNATS_LOG_FORMAT": "json",
		}, Config{":8080", slog.LevelDebug, "json"}},
		{"flag beats env", []string{"-addr", ":9090"}, map[string]string{"HEYNATS_HTTP_ADDR": ":8080"},
			Config{":9090", slog.LevelInfo, "text"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Load(tt.args, envOf(tt.env))
			if err != nil || got != tt.want {
				t.Fatalf("Load = %+v, %v; want %+v", got, err, tt.want)
			}
		})
	}
}

func TestLoadRejectsInvalidValues(t *testing.T) {
	for _, env := range []map[string]string{
		{"HEYNATS_LOG_LEVEL": "loud"},
		{"HEYNATS_LOG_FORMAT": "xml"},
	} {
		if _, err := Load(nil, envOf(env)); err == nil {
			t.Errorf("Load(%v) accepted invalid value", env)
		}
	}
}
