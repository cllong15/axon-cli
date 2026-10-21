package config

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("expected DefaultConfig to return non-nil config")
	}

	if cfg.WorkerCount != 10 {
		t.Errorf("expected default WorkerCount to be 10, got %d", cfg.WorkerCount)
	}

	if cfg.Timeout != 5*time.Second {
		t.Errorf("expected default Timeout to be 5s, got %v", cfg.Timeout)
	}

	if cfg.OutputFormat != "table" {
		t.Errorf("expected default OutputFormat to be 'table', got %q", cfg.OutputFormat)
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("expected default config to be valid, got error: %v", err)
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		expectErr bool
	}{
		{
			name: "valid table format",
			config: Config{
				WorkerCount:  5,
				Timeout:      3 * time.Second,
				OutputFormat: "table",
			},
			expectErr: false,
		},
		{
			name: "valid json format",
			config: Config{
				WorkerCount:  1,
				Timeout:      1 * time.Second,
				OutputFormat: "json",
			},
			expectErr: false,
		},
		{
			name: "zero worker count",
			config: Config{
				WorkerCount:  0,
				Timeout:      5 * time.Second,
				OutputFormat: "table",
			},
			expectErr: true,
		},
		{
			name: "negative worker count",
			config: Config{
				WorkerCount:  -2,
				Timeout:      5 * time.Second,
				OutputFormat: "table",
			},
			expectErr: true,
		},
		{
			name: "zero timeout",
			config: Config{
				WorkerCount:  5,
				Timeout:      0,
				OutputFormat: "table",
			},
			expectErr: true,
		},
		{
			name: "negative timeout",
			config: Config{
				WorkerCount:  5,
				Timeout:      -1 * time.Second,
				OutputFormat: "table",
			},
			expectErr: true,
		},
		{
			name: "invalid output format",
			config: Config{
				WorkerCount:  5,
				Timeout:      5 * time.Second,
				OutputFormat: "yaml",
			},
			expectErr: true,
		},
		{
			name: "empty output format",
			config: Config{
				WorkerCount:  5,
				Timeout:      5 * time.Second,
				OutputFormat: "",
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

