package checker

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunPool_EmptyURLs(t *testing.T) {
	ctx := context.Background()

	resNil := RunPool(ctx, nil, 5, time.Second, "GET")
	if resNil != nil {
		t.Errorf("expected nil for nil urls, got %v", resNil)
	}

	resEmpty := RunPool(ctx, []string{}, 5, time.Second, "")
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

	results := RunPool(ctx, urls, 3, 2*time.Second, "GET")

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

	results := RunPool(ctx, []string{badURL}, 1, 500*time.Millisecond, "GET")

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

	results := RunPool(ctx, []string{invalidURL}, 1, time.Second, "GET")

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
	results := RunPool(ctx, []string{slowServer.URL}, 1, 50*time.Millisecond, "GET")

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

	results := RunPool(ctx, []string{slowServer.URL}, 1, 2*time.Second, "GET")

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
	resZero := RunPool(ctx, []string{s.URL}, 0, time.Second, "GET")
	if len(resZero) != 1 || resZero[0].StatusCode != http.StatusOK {
		t.Errorf("expected success with clamped workerCount 0, got %+v", resZero)
	}

	// workerCount > len(urls) clamped to len(urls)
	resLarge := RunPool(ctx, []string{s.URL}, 50, time.Second, "GET")
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
	results := RunPool(ctx, urls, 4, 2*time.Second, "GET")

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
	results := RunPool(ctx, []string{server.URL}, 1, time.Second, "GET")

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

func TestRunPool_CustomMethod(t *testing.T) {
	var receivedMethod string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx := context.Background()
	results := RunPool(ctx, []string{server.URL}, 1, time.Second, "HEAD")

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if receivedMethod != "HEAD" {
		t.Errorf("expected server to receive 'HEAD', got %q", receivedMethod)
	}
}

func TestDetectProtocol(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantProtocol string
		wantAddr     string
	}{
		{
			name:         "http scheme",
			input:        "http://example.com",
			wantProtocol: "http",
			wantAddr:     "http://example.com",
		},
		{
			name:         "https scheme",
			input:        "https://example.com:8443",
			wantProtocol: "http",
			wantAddr:     "https://example.com:8443",
		},
		{
			name:         "tcp scheme",
			input:        "tcp://127.0.0.1:8080",
			wantProtocol: "tcp",
			wantAddr:     "127.0.0.1:8080",
		},
		{
			name:         "bare host and port",
			input:        "127.0.0.1:9092",
			wantProtocol: "tcp",
			wantAddr:     "127.0.0.1:9092",
		},
		{
			name:         "bare ipv6 and port",
			input:        "[::1]:5432",
			wantProtocol: "tcp",
			wantAddr:     "[::1]:5432",
		},
		{
			name:         "bare domain defaults to https",
			input:        "google.com",
			wantProtocol: "http",
			wantAddr:     "https://google.com",
		},
		{
			name:         "domain with path containing colon",
			input:        "example.com/api:v1",
			wantProtocol: "http",
			wantAddr:     "https://example.com/api:v1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proto, addr := DetectProtocol(tt.input)
			if proto != tt.wantProtocol {
				t.Errorf("DetectProtocol(%q) proto = %q, want %q", tt.input, proto, tt.wantProtocol)
			}
			if addr != tt.wantAddr {
				t.Errorf("DetectProtocol(%q) addr = %q, want %q", tt.input, addr, tt.wantAddr)
			}
		})
	}
}

func TestRunPool_TCP_Success(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start mock TCP listener: %v", err)
	}
	defer ln.Close()

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
		}
	}()

	ctx := context.Background()
	results := RunPool(ctx, []string{ln.Addr().String(), "tcp://" + ln.Addr().String()}, 2, time.Second, "GET")

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	for _, res := range results {
		if res.Protocol != "tcp" {
			t.Errorf("expected protocol 'tcp', got %q", res.Protocol)
		}
		if res.Error != "" {
			t.Errorf("unexpected error for TCP target: %s", res.Error)
		}
	}
}

func TestRunPool_TCP_Failure(t *testing.T) {
	ctx := context.Background()
	results := RunPool(ctx, []string{"tcp://127.0.0.1:1"}, 1, 500*time.Millisecond, "GET")

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Protocol != "tcp" {
		t.Errorf("expected protocol 'tcp', got %q", results[0].Protocol)
	}
	if results[0].Error == "" {
		t.Errorf("expected error for closed TCP port, got nil")
	}
}

func TestExpandTargets(t *testing.T) {
	t.Run("IPv4 CIDR expansion", func(t *testing.T) {
		res, err := ExpandTargets([]string{"192.168.1.0/30:80"}, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := []string{
			"192.168.1.0:80",
			"192.168.1.1:80",
			"192.168.1.2:80",
			"192.168.1.3:80",
		}
		if len(res) != len(expected) {
			t.Fatalf("expected %d targets, got %d: %v", len(expected), len(res), res)
		}
		for i, v := range expected {
			if res[i] != v {
				t.Errorf("index %d: expected %s, got %s", i, v, res[i])
			}
		}
	})

	t.Run("IPv6 bracketed CIDR expansion", func(t *testing.T) {
		res, err := ExpandTargets([]string{"[2001:db8::/126]:22"}, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := []string{
			"[2001:db8::]:22",
			"[2001:db8::1]:22",
			"[2001:db8::2]:22",
			"[2001:db8::3]:22",
		}
		if len(res) != len(expected) {
			t.Fatalf("expected %d targets, got %d: %v", len(expected), len(res), res)
		}
		for i, v := range expected {
			if res[i] != v {
				t.Errorf("index %d: expected %s, got %s", i, v, res[i])
			}
		}
	})

	t.Run("passthrough non-CIDR targets", func(t *testing.T) {
		targets := []string{
			"https://api.example.com",
			"10.0.0.1:5432",
			"google.com",
			"192.168.1.0/24",
			"example.com:8080/healthz",
		}
		res, err := ExpandTargets(targets, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(res) != len(targets) {
			t.Fatalf("expected %d targets, got %d", len(targets), len(res))
		}
		for i, v := range targets {
			if res[i] != v {
				t.Errorf("index %d: expected %s, got %s", i, v, res[i])
			}
		}
	})

	t.Run("exceeds max targets limit", func(t *testing.T) {
		_, err := ExpandTargets([]string{"192.168.1.0/28:80"}, 4) // /28 is 16 IPs, limit is 4
		if err == nil {
			t.Fatal("expected error when CIDR exceeds maxTargets, got nil")
		}
		if !strings.Contains(err.Error(), "exceeds max targets limit") {
			t.Errorf("expected max targets error message, got: %v", err)
		}
	})

	t.Run("invalid CIDR syntax passes through as normal target", func(t *testing.T) {
		res, err := ExpandTargets([]string{"invalid/cidr/path:80"}, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(res) != 1 || res[0] != "invalid/cidr/path:80" {
			t.Errorf("expected passthrough for invalid CIDR, got %v", res)
		}
	})
}
