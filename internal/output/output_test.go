package output

import (
	"axon-cli/internal/checker"
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

// errWriter simulates an I/O error on Write
type errWriter struct{}

func (e *errWriter) Write(p []byte) (n int, err error) {
	return 0, errors.New("simulated write error")
}

func TestFormatJSON(t *testing.T) {
	t.Run("empty results", func(t *testing.T) {
		var buf bytes.Buffer
		err := FormatJSON(&buf, []checker.CheckResult{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var decoded []checker.CheckResult
		if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}
		if len(decoded) != 0 {
			t.Errorf("expected 0 decoded results, got %d", len(decoded))
		}
	})

	t.Run("valid results with success and error", func(t *testing.T) {
		results := []checker.CheckResult{
			{
				URL:        "https://example.com",
				StatusCode: 200,
				Latency:    125 * time.Millisecond,
			},
			{
				URL:        "http://invalid.local",
				StatusCode: 0,
				Latency:    5 * time.Millisecond,
				Error:      "connection refused",
			},
		}

		var buf bytes.Buffer
		err := FormatJSON(&buf, results)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var decoded []checker.CheckResult
		if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
			t.Fatalf("failed to decode JSON: %v", err)
		}

		if len(decoded) != 2 {
			t.Fatalf("expected 2 decoded items, got %d", len(decoded))
		}

		if decoded[0].URL != "https://example.com" || decoded[0].StatusCode != 200 {
			t.Errorf("unexpected decoded item 0: %+v", decoded[0])
		}

		if decoded[1].StatusCode != 0 || decoded[1].Error != "connection refused" {
			t.Errorf("unexpected decoded item 1: %+v", decoded[1])
		}
	})

	t.Run("write error propagation", func(t *testing.T) {
		results := []checker.CheckResult{
			{URL: "https://example.com", StatusCode: 200},
		}
		err := FormatJSON(&errWriter{}, results)
		if err == nil {
			t.Error("expected error from errWriter, got nil")
		}
	})
}

func TestFormatTable(t *testing.T) {
	t.Run("empty results", func(t *testing.T) {
		var buf bytes.Buffer
		err := FormatTable(&buf, []checker.CheckResult{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "URL") || !strings.Contains(output, "STATUS") {
			t.Errorf("expected header in table output, got: %s", output)
		}
	})

	t.Run("valid results with status codes and err mapping", func(t *testing.T) {
		results := []checker.CheckResult{
			{
				URL:        "https://api.github.com",
				StatusCode: 200,
				Latency:    150 * time.Millisecond,
			},
			{
				URL:        "http://bad.host",
				StatusCode: 0,
				Latency:    10 * time.Millisecond,
				Error:      "host unreachable",
			},
		}

		var buf bytes.Buffer
		err := FormatTable(&buf, results)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		out := buf.String()
		lines := strings.Split(strings.TrimSpace(out), "\n")
		if len(lines) != 3 {
			t.Fatalf("expected 3 lines (1 header + 2 rows), got %d lines: %q", len(lines), out)
		}

		// Check header
		if !strings.Contains(lines[0], "URL") || !strings.Contains(lines[0], "LATENCY") {
			t.Errorf("missing header columns: %s", lines[0])
		}

		// Check 200 row
		if !strings.Contains(lines[1], "https://api.github.com") || !strings.Contains(lines[1], "200") {
			t.Errorf("expected URL and 200 in line 1: %s", lines[1])
		}

		// Check ERR row (StatusCode == 0 mapped to ERR)
		if !strings.Contains(lines[2], "http://bad.host") || !strings.Contains(lines[2], "ERR") || !strings.Contains(lines[2], "host unreachable") {
			t.Errorf("expected ERR and error message in line 2: %s", lines[2])
		}
	})

	t.Run("write error propagation", func(t *testing.T) {
		results := []checker.CheckResult{
			{URL: "https://example.com", StatusCode: 200},
		}
		err := FormatTable(&errWriter{}, results)
		if err == nil {
			t.Error("expected error from errWriter, got nil")
		}
	})
}

