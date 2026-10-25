package checker

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

type CheckResult struct {
	URL        string        `json:"url"`
	StatusCode int           `json:"status_code"`
	Latency    time.Duration `json:"latency"`
	Error      string        `json:"error,omitempty"`
}

func RunPool(ctx context.Context, urls []string, workerCount int, timeout time.Duration) []CheckResult {
	if len(urls) == 0 {
		return nil
	}
	if workerCount > len(urls) {
		workerCount = len(urls)
	}
	if workerCount <= 0 {
		workerCount = 1
	}

	jobsChan := make(chan string, len(urls))
	resultsChan := make(chan CheckResult, len(urls))
	client := NewClient(timeout)

	var wg sync.WaitGroup
	for w := 0; w < workerCount; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for url := range jobsChan {
				url = NormalizeURL(url)
				start := time.Now()
				req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
				if err != nil {
					resultsChan <- CheckResult{URL: url,
						Error: err.Error()}
					continue
				}
				resp, err := client.Do(req)
				latency := time.Since(start)
				if err != nil {
					resultsChan <- CheckResult{URL: url,
						Latency: latency,
						Error:   err.Error()}
					continue
				}
				_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
				resp.Body.Close()
				resultsChan <- CheckResult{
					URL:        url,
					StatusCode: resp.StatusCode,
					Latency:    latency,
				}
			}

		}()
	}
	for _, u := range urls {
		jobsChan <- u
	}
	close(jobsChan)

	wg.Wait()
	close(resultsChan)

	var results []CheckResult
	for res := range resultsChan {
		results = append(results, res)
	}
	return results
}

func NormalizeURL(raw string) string {
	u := strings.TrimSpace(raw)
	if u == "" {
		return ""
	}
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		return "https://" + u
	}
	return u
}
