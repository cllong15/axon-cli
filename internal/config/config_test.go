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

	if cfg.OnlyErrors != false {
		t.Errorf("expected default OnlyErrors to be false, got %v", cfg.OnlyErrors)
	}

	if cfg.Threshold != 0 {
		t.Errorf("expected default Threshold to be 0, got %v", cfg.Threshold)
	}

	if cfg.NumericStatus != false {
		t.Errorf("expected default NumericStatus to be false, got %v", cfg.NumericStatus)
	}

	if cfg.MaxTargets != 1024 {
		t.Errorf("expected default MaxTargets to be 1024, got %d", cfg.MaxTargets)
	}

	if cfg.AutoConfirm != false {
		t.Errorf("expected default AutoConfirm to be false, got %v", cfg.AutoConfirm)
	}

	if cfg.BatchSize != 256 {
		t.Errorf("expected default BatchSize to be 256, got %d", cfg.BatchSize)
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
				BatchSize:    256,
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
				BatchSize:    256,
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
				BatchSize:    256,
			},
			expectErr: false,
		},
		{
			name: "valid with zero max targets (unlimited)",
			config: Config{
				WorkerCount:  1,
				Timeout:      1 * time.Second,
				OutputFormat: "json",
				Method:       "GET",
				BatchSize:    256,
				MaxTargets:   0,
			},
			expectErr: false,
		},
		{
			name: "zero batch size",
			config: Config{
				WorkerCount:  5,
				Timeout:      3 * time.Second,
				OutputFormat: "table",
				Method:       "GET",
				BatchSize:    0,
			},
			expectErr: true,
		},
		{
			name: "negative batch size",
			config: Config{
				WorkerCount:  5,
				Timeout:      3 * time.Second,
				OutputFormat: "table",
				Method:       "GET",
				BatchSize:    -5,
			},
			expectErr: true,
		},
		{
			name: "negative max targets",
			config: Config{
				WorkerCount:  5,
				Timeout:      3 * time.Second,
				OutputFormat: "table",
				Method:       "GET",
				BatchSize:    256,
				MaxTargets:   -1,
			},
			expectErr: true,
		},
		{
			name: "invalid http method",
			config: Config{
				WorkerCount:  1,
				Timeout:      1 * time.Second,
				OutputFormat: "json",
				Method:       "INVALID_METHOD",
				BatchSize:    256,
			},
			expectErr: true,
		},
		{
			name: "negative threshold",
			config: Config{
				WorkerCount:  5,
				Timeout:      5 * time.Second,
				OutputFormat: "table",
				Method:       "GET",
				Threshold:    -100 * time.Millisecond,
				BatchSize:    256,
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
				BatchSize:    256,
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
				BatchSize:    256,
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
				BatchSize:    256,
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
				BatchSize:    256,
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
				BatchSize:    256,
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
				BatchSize:    256,
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
