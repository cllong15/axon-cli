package cmd

import (
	"axon-cli/internal/checker"
	"axon-cli/internal/config"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func resetCommand() {
	*cfg = *config.DefaultConfig()
	_ = rootCmd.Flags().Set("workers", "10")
	_ = rootCmd.Flags().Set("timeout", "5s")
	_ = rootCmd.Flags().Set("format", "table")
}

func TestRootCmd_NoTargets(t *testing.T) {
	resetCommand()
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{})

	err := ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected error when no targets provided, got nil")
	}

	if !strings.Contains(err.Error(), "no target URLs provided") {
		t.Errorf("expected missing target error message, got: %v", err)
	}
}

func TestRootCmd_InvalidFormat(t *testing.T) {
	resetCommand()
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"--format", "unsupported", "https://example.com"})

	err := ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected validation error for invalid format, got nil")
	}

	if !strings.Contains(err.Error(), "output format must be table or json") {
		t.Errorf("expected format validation error, got: %v", err)
	}
}

func TestRootCmd_SuccessTableOutput(t *testing.T) {
	resetCommand()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"--format", "table", server.URL})

	err := ExecuteContext(context.Background())
	if err != nil {
		t.Fatalf("unexpected execution error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "URL") || !strings.Contains(output, "STATUS") {
		t.Errorf("expected table header in output, got: %s", output)
	}
	if !strings.Contains(output, server.URL) || !strings.Contains(output, "200") {
		t.Errorf("expected server URL and 200 status in output, got: %s", output)
	}
}

func TestRootCmd_SuccessJSONOutput(t *testing.T) {
	resetCommand()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"-f", "json", server.URL})

	err := ExecuteContext(context.Background())
	if err != nil {
		t.Fatalf("unexpected execution error: %v", err)
	}

	var results []checker.CheckResult
	if err := json.Unmarshal(buf.Bytes(), &results); err != nil {
		t.Fatalf("failed to decode JSON output: %v; raw: %s", err, buf.String())
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result in JSON, got %d", len(results))
	}
	if results[0].URL != server.URL || results[0].StatusCode != http.StatusOK {
		t.Errorf("unexpected check result: %+v", results[0])
	}
}

func TestRootCmd_CustomFlags(t *testing.T) {
	resetCommand()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"-w", "2", "-t", "2s", server.URL})

	err := ExecuteContext(context.Background())
	if err != nil {
		t.Fatalf("unexpected execution error with custom flags: %v", err)
	}

	if cfg.WorkerCount != 2 {
		t.Errorf("expected WorkerCount to be 2, got %d", cfg.WorkerCount)
	}
	if cfg.Timeout != 2*time.Second {
		t.Errorf("expected Timeout to be 2s, got %v", cfg.Timeout)
	}
}

