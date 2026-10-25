package cmd

import (
	"axon-cli/internal/checker"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExecute_NoTargets(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Execute(context.Background(), []string{}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error when no targets provided, got nil")
	}

	if !strings.Contains(err.Error(), "no target URLs provided") {
		t.Errorf("expected missing target error message, got: %v", err)
	}
}

func TestExecute_InvalidFormat(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Execute(context.Background(), []string{"-format", "unsupported", "https://example.com"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected validation error for invalid format, got nil")
	}

	if !strings.Contains(err.Error(), "output format must be table or json") {
		t.Errorf("expected format validation error, got: %v", err)
	}
}

func TestExecute_InvalidFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Execute(context.Background(), []string{"-unknown-flag"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected flag parsing error for unknown flag, got nil")
	}
}

func TestExecute_SuccessTableOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	err := Execute(context.Background(), []string{"-format", "table", server.URL}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected execution error: %v", err)
	}

	output := stdout.String()
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

func TestExecute_SuccessJSONOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	err := Execute(context.Background(), []string{"-f", "json", server.URL}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected execution error: %v", err)
	}

	var results []checker.CheckResult
	if err := json.Unmarshal(stdout.Bytes(), &results); err != nil {
		t.Fatalf("failed to decode JSON output: %v; raw: %s", err, stdout.String())
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result in JSON, got %d", len(results))
	}
	if results[0].URL != server.URL || results[0].StatusCode != http.StatusOK {
		t.Errorf("unexpected check result: %+v", results[0])
	}
}

func TestExecute_CustomFlags(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	err := Execute(context.Background(), []string{"-w", "2", "-t", "2s", server.URL}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected execution error with custom flags: %v", err)
	}
}

func TestExecute_TargetFile_Success(t *testing.T) {
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

	var stdout, stderr bytes.Buffer
	err := Execute(context.Background(), []string{"-file", filePath}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected execution error with target file: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, server.URL) || !strings.Contains(out, "200") {
		t.Errorf("expected target from file to be audited, got:\n%s", out)
	}
}

func TestExecute_TargetFile_NotFound(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Execute(context.Background(), []string{"-i", "nonexistent_file_xyz.txt"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error when target file does not exist, got nil")
	}
}

func TestExecute_TargetFile_ScannerError(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "huge_line.txt")
	longLine := strings.Repeat("x", 70000)
	if err := os.WriteFile(filePath, []byte(longLine), 0644); err != nil {
		t.Fatalf("failed to write huge line file: %v", err)
	}

	var stdout, stderr bytes.Buffer
	err := Execute(context.Background(), []string{"-i", filePath}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected scanner error for line exceeding buffer size, got nil")
	}
}

func TestExecute_Version(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Execute(context.Background(), []string{"-version"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error on -version: %v", err)
	}
	if !strings.Contains(stdout.String(), "axon version") {
		t.Errorf("expected version output, got: %q", stdout.String())
	}
}

func TestExecute_Version_Short(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Execute(context.Background(), []string{"-v"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error on -v: %v", err)
	}
	if !strings.Contains(stdout.String(), "axon version") {
		t.Errorf("expected version output, got: %q", stdout.String())
	}
}

func TestExecute_Method_Head(t *testing.T) {
	var receivedMethod string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	err := Execute(context.Background(), []string{"-m", "HEAD", server.URL}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error executing HEAD method: %v", err)
	}
	if receivedMethod != "HEAD" {
		t.Errorf("expected server to receive 'HEAD', got %q", receivedMethod)
	}
}

func TestExecute_InvalidMethod(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Execute(context.Background(), []string{"-method", "BAD_METHOD", "https://example.com"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error on invalid HTTP method, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported HTTP method") {
		t.Errorf("expected unsupported method error message, got: %v", err)
	}
}

func TestExecute_FailOnError_AllPassing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	err := Execute(context.Background(), []string{"-fail-on-error", server.URL}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("expected nil error when all targets pass, got: %v", err)
	}
}

func TestExecute_FailOnError_WithHttpFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	err := Execute(context.Background(), []string{"-fail", server.URL}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error with -fail flag when server returns 500, got nil")
	}
	if !strings.Contains(err.Error(), "one or more targets failed") {
		t.Errorf("expected failure message, got: %v", err)
	}
	// Verify that table output was still printed to stdout before error return
	if !strings.Contains(stdout.String(), "500") {
		t.Errorf("expected table output in stdout before failure return, got: %s", stdout.String())
	}
}

type errWriter struct{}

func (e *errWriter) Write(p []byte) (n int, err error) {
	return 0, errors.New("simulated output write failure")
}

func TestExecute_FormatError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	var stderr bytes.Buffer
	err := Execute(context.Background(), []string{server.URL}, &errWriter{}, &stderr)
	if err == nil {
		t.Fatal("expected error when output writer fails, got nil")
	}
}

func TestExecute_FailOnError_NetworkError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Execute(context.Background(), []string{"-fail", "http://127.0.0.1:1"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error with -fail flag on network error, got nil")
	}
}
