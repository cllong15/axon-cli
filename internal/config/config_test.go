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

	if cfg.Method != "GET" {
		t.Errorf("expected default Method to be 'GET', got %q", cfg.Method)
	}

	if cfg.FailOnError != false {
		t.Errorf("expected default FailOnError to be false, got %v", cfg.FailOnError)
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
			name: "valid table format and GET method",
			config: Config{
				WorkerCount:  5,
				Timeout:      3 * time.Second,
				OutputFormat: "table",
				Method:       "GET",
			},
			expectErr: false,
		},
		{
			name: "valid json format and HEAD method",
			config: Config{
				WorkerCount:  1,
				Timeout:      1 * time.Second,
				OutputFormat: "json",
				Method:       "HEAD",
			},
			expectErr: false,
		},
		{
			name: "valid lowercase method normalized",
			config: Config{
				WorkerCount:  1,
				Timeout:      1 * time.Second,
				OutputFormat: "json",
				Method:       "post",
			},
			expectErr: false,
		},
		{
			name: "invalid http method",
			config: Config{
				WorkerCount:  1,
				Timeout:      1 * time.Second,
				OutputFormat: "json",
				Method:       "INVALID_METHOD",
			},
			expectErr: true,
		},
		{
			name: "zero worker count",
			config: Config{
				WorkerCount:  0,
				Timeout:      5 * time.Second,
				OutputFormat: "table",
				Method:       "GET",
			},
			expectErr: true,
		},
		{
			name: "negative worker count",
			config: Config{
				WorkerCount:  -2,
				Timeout:      5 * time.Second,
				OutputFormat: "table",
				Method:       "GET",
			},
			expectErr: true,
		},
		{
			name: "zero timeout",
			config: Config{
				WorkerCount:  5,
				Timeout:      0,
				OutputFormat: "table",
				Method:       "GET",
			},
			expectErr: true,
		},
		{
			name: "negative timeout",
			config: Config{
				WorkerCount:  5,
				Timeout:      -1 * time.Second,
				OutputFormat: "table",
				Method:       "GET",
			},
			expectErr: true,
		},
		{
			name: "invalid output format",
			config: Config{
				WorkerCount:  5,
				Timeout:      5 * time.Second,
				OutputFormat: "yaml",
				Method:       "GET",
			},
			expectErr: true,
		},
		{
			name: "empty output format",
			config: Config{
				WorkerCount:  5,
				Timeout:      5 * time.Second,
				OutputFormat: "",
				Method:       "GET",
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
