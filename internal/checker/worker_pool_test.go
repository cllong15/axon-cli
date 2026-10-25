package checker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunPool_EmptyURLs(t *testing.T) {
	ctx := context.Background()

	resNil := RunPool(ctx, nil, 5, time.Second)
	if resNil != nil {
		t.Errorf("expected nil for nil urls, got %v", resNil)
	}

	resEmpty := RunPool(ctx, []string{}, 5, time.Second)
	if resEmpty != nil {
		t.Errorf("expected nil for empty urls, got %v", resEmpty)
	}
}

func TestRunPool_SuccessAndStatusCodes(t *testing.T) {
	s1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer s1.Close()

	s2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer s2.Close()

	s3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer s3.Close()

	urls := []string{s1.URL, s2.URL, s3.URL}
	ctx := context.Background()

	results := RunPool(ctx, urls, 3, 2*time.Second)

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	resultsMap := make(map[string]CheckResult)
	for _, res := range results {
		resultsMap[res.URL] = res
	}

	if r1, ok := resultsMap[s1.URL]; !ok || r1.StatusCode != http.StatusOK || r1.Error != "" {
		t.Errorf("expected 200 OK without error for s1, got %+v", r1)
	}

	if r2, ok := resultsMap[s2.URL]; !ok || r2.StatusCode != http.StatusNotFound || r2.Error != "" {
		t.Errorf("expected 404 NotFound without error for s2, got %+v", r2)
	}

	if r3, ok := resultsMap[s3.URL]; !ok || r3.StatusCode != http.StatusInternalServerError || r3.Error != "" {
		t.Errorf("expected 500 InternalError without error for s3, got %+v", r3)
	}

	for _, res := range results {
		if res.Latency <= 0 {
			t.Errorf("expected positive latency for %s, got %v", res.URL, res.Latency)
		}
	}
}

func TestRunPool_NetworkError(t *testing.T) {
	// Connect to closed/invalid local port
	badURL := "http://127.0.0.1:1"
	ctx := context.Background()

	results := RunPool(ctx, []string{badURL}, 1, 500*time.Millisecond)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Error == "" {
		t.Errorf("expected error for unreachable endpoint, got empty error")
	}
	if results[0].StatusCode != 0 {
		t.Errorf("expected status code 0 for network failure, got %d", results[0].StatusCode)
	}
}

func TestRunPool_InvalidURLSyntax(t *testing.T) {
	invalidURL := "http://invalid domain with spaces"
	ctx := context.Background()

	results := RunPool(ctx, []string{invalidURL}, 1, time.Second)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Error == "" {
		t.Errorf("expected error for malformed URL, got empty error")
	}
}

func TestRunPool_Timeout(t *testing.T) {
	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer slowServer.Close()

	ctx := context.Background()
	results := RunPool(ctx, []string{slowServer.URL}, 1, 50*time.Millisecond)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Error == "" {
		t.Errorf("expected timeout error, got nil")
	}
}

func TestRunPool_ContextCancellation(t *testing.T) {
	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer slowServer.Close()

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context quickly
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	results := RunPool(ctx, []string{slowServer.URL}, 1, 2*time.Second)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Error == "" {
		t.Errorf("expected error due to cancelled context, got nil")
	}
}

func TestRunPool_WorkerCountClamping(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer s.Close()

	ctx := context.Background()

	// workerCount <= 0 clamped to 1
	resZero := RunPool(ctx, []string{s.URL}, 0, time.Second)
	if len(resZero) != 1 || resZero[0].StatusCode != http.StatusOK {
		t.Errorf("expected success with clamped workerCount 0, got %+v", resZero)
	}

	// workerCount > len(urls) clamped to len(urls)
	resLarge := RunPool(ctx, []string{s.URL}, 50, time.Second)
	if len(resLarge) != 1 || resLarge[0].StatusCode != http.StatusOK {
		t.Errorf("expected success with clamped workerCount 50, got %+v", resLarge)
	}
}

func TestRunPool_Concurrency(t *testing.T) {
	var activeWorkers int32
	var maxConcurrent int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := atomic.AddInt32(&activeWorkers, 1)
		for {
			max := atomic.LoadInt32(&maxConcurrent)
			if current <= max || atomic.CompareAndSwapInt32(&maxConcurrent, max, current) {
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
		atomic.AddInt32(&activeWorkers, -1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	urls := make([]string, 10)
	for i := range urls {
		urls[i] = server.URL
	}

	ctx := context.Background()
	results := RunPool(ctx, urls, 4, 2*time.Second)

	if len(results) != 10 {
		t.Fatalf("expected 10 results, got %d", len(results))
	}

	peak := atomic.LoadInt32(&maxConcurrent)
	if peak < 2 {
		t.Errorf("expected concurrent execution, but peak concurrent was %d", peak)
	}
	if peak > 4 {
		t.Errorf("expected peak concurrent to be at most 4, got %d", peak)
	}
}

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "empty string", input: "", expected: ""},
		{name: "whitespace only", input: "   ", expected: ""},
		{name: "bare domain", input: "example.com", expected: "https://example.com"},
		{name: "bare domain with spaces", input: "  github.com  ", expected: "https://github.com"},
		{name: "bare IP", input: "192.168.1.1", expected: "https://192.168.1.1"},
		{name: "already http", input: "http://localhost:8080", expected: "http://localhost:8080"},
		{name: "already https", input: "https://api.github.com", expected: "https://api.github.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeURL(tt.input)
			if got != tt.expected {
				t.Errorf("NormalizeURL(%q) = %q, expected %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestRunPool_LargeResponseBody(t *testing.T) {
	// Server returns 10KB of payload, exceeding the 4KB LimitReader threshold
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(strings.Repeat("X", 10240)))
	}))
	defer server.Close()

	ctx := context.Background()
	results := RunPool(ctx, []string{server.URL}, 1, time.Second)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK for large response body, got %d", results[0].StatusCode)
	}
	if results[0].Error != "" {
		t.Errorf("expected no error for large response body, got: %s", results[0].Error)
	}
}
