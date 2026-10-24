package cmd

import (
	"axon-cli/internal/checker"
	"axon-cli/internal/config"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func resetCommand() {
	*cfg = *config.DefaultConfig()
	targetFile = ""
	_ = rootCmd.Flags().Set("workers", "10")
	_ = rootCmd.Flags().Set("timeout", "5s")
	_ = rootCmd.Flags().Set("format", "table")
	_ = rootCmd.Flags().Set("file", "")
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
	if !strings.Contains(output, "Total: 1 | Passed: 1 | Failed: 0") {
		t.Errorf("expected summary line in output, got: %s", output)
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

func TestRootCmd_TargetFile_Success(t *testing.T) {
	resetCommand()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "targets.txt")
	content := "# comment line\n\n" + server.URL + "\n"
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp targets file: %v", err)
	}

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"--file", filePath})

	err := ExecuteContext(context.Background())
	if err != nil {
		t.Fatalf("unexpected execution error with target file: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, server.URL) || !strings.Contains(out, "200") {
		t.Errorf("expected target from file to be audited, got:\n%s", out)
	}
}

func TestRootCmd_TargetFile_NotFound(t *testing.T) {
	resetCommand()
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"-i", "nonexistent_file_xyz.txt"})

	err := ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected error when target file does not exist, got nil")
	}
}

func TestRootCmd_Version(t *testing.T) {
	resetCommand()
	if rootCmd.Version != "1.0.1" {
		t.Errorf("expected rootCmd.Version to be '1.0.1', got %q", rootCmd.Version)
	}
}

func TestRootCmd_TargetFile_ScannerError(t *testing.T) {
	resetCommand()
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "huge_line.txt")
	// bufio.Scanner default max token size is 64KB (65536 bytes). A line exceeding 64KB triggers bufio.ErrTooLong.
	longLine := strings.Repeat("x", 70000)
	if err := os.WriteFile(filePath, []byte(longLine), 0644); err != nil {
		t.Fatalf("failed to write huge line file: %v", err)
	}

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"--file", filePath})

	err := ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected scanner error for line exceeding buffer size, got nil")
	}
}
