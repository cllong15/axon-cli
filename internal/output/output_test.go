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
		if strings.Contains(output, "Total:") {
			t.Errorf("did not expect summary row for empty results, got: %s", output)
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

		// Check header columns
		if !strings.Contains(out, "URL") || !strings.Contains(out, "STATUS") || !strings.Contains(out, "LATENCY") {
			t.Errorf("missing header columns in output: %s", out)
		}

		// Check 200 row
		if !strings.Contains(out, "https://api.github.com") || !strings.Contains(out, "200") {
			t.Errorf("expected URL and 200 in output: %s", out)
		}

		// Check ERR row
		if !strings.Contains(out, "http://bad.host") || !strings.Contains(out, "ERR") || !strings.Contains(out, "host unreachable") {
			t.Errorf("expected ERR and error message in output: %s", out)
		}

		// Check summary line
		expectedSummary := "Total: 2 | Passed: 1 | Failed: 1 | Slow: 0 | Avg Latency: 80ms"
		if !strings.Contains(out, expectedSummary) {
			t.Errorf("expected summary line %q in output, got:\n%s", expectedSummary, out)
		}
	})

	t.Run("results with slow markers", func(t *testing.T) {
		results := []checker.CheckResult{
			{URL: "https://api.github.com", StatusCode: 200, Latency: 300 * time.Millisecond, Slow: true},
		}

		var buf bytes.Buffer
		err := FormatTable(&buf, results)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		out := buf.String()
		if !strings.Contains(out, "[slow]") {
			t.Errorf("expected [slow] marker in output, got:\n%s", out)
		}
		if !strings.Contains(out, "Slow: 1") {
			t.Errorf("expected 'Slow: 1' in summary footer, got:\n%s", out)
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
