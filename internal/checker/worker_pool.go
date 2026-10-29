package checker

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type CheckResult struct {
	URL        string        `json:"url"`
	Protocol   string        `json:"protocol,omitempty"`
	StatusCode int           `json:"status_code"`
	Latency    time.Duration `json:"latency"`
	Error      string        `json:"error,omitempty"`
	Slow       bool          `json:"slow,omitempty"`
}

func RunPool(ctx context.Context, urls []string, workerCount int, timeout time.Duration, method string) []CheckResult {
	if method == "" {
		method = http.MethodGet
	}
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
				proto, addr := DetectProtocol(url)
				if proto == "tcp" {
					d := net.Dialer{Timeout: timeout}
					start := time.Now()
					conn, err := d.DialContext(ctx, proto, addr)
					latency := time.Since(start)
					if err != nil {
						resultsChan <- CheckResult{
							URL:      addr,
							Protocol: proto,
							Latency:  latency,
							Error:    err.Error(),
						}
						continue
					}
					_ = conn.Close()
					resultsChan <- CheckResult{
						URL:        addr,
						Protocol:   proto,
						StatusCode: 200,
						Latency:    latency,
					}
					continue
				}

				start := time.Now()
				req, err := http.NewRequestWithContext(ctx, method, addr, nil)
				if err != nil {
					resultsChan <- CheckResult{
						URL:      addr,
						Protocol: proto,
						Error:    err.Error(),
						Slow:     false,
					}
					continue
				}
				resp, err := client.Do(req)
				latency := time.Since(start)
				if err != nil {
					resultsChan <- CheckResult{
						URL:      addr,
						Protocol: proto,
						Latency:  latency,
						Error:    err.Error(),
					}
					continue
				}
				_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
				resp.Body.Close()
				resultsChan <- CheckResult{
					URL:        addr,
					Protocol:   proto,
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

func DetectProtocol(raw string) (protocol, addr string) {
	p := strings.TrimSpace(raw)
	if strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://") {
		return "http", p
	}
	if strings.HasPrefix(p, "tcp://") {
		return "tcp", strings.TrimPrefix(p, "tcp://")
	}
	if host, port, err := net.SplitHostPort(p); err == nil && port != "" && !strings.Contains(host, "/") {
		return "tcp", p
	}
	return "http", NormalizeURL(p)
}

func ExpandTargets(rawTargets []string, maxTargets int) ([]string, error) {
	var all []string
	for _, raw := range rawTargets {
		target := strings.TrimSpace(raw)
		if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") || !strings.Contains(target, "/") {
			all = append(all, target)
			continue
		}
		lastColon := strings.LastIndex(target, ":")
		if lastColon == -1 || lastColon < strings.LastIndex(target, "/") {
			all = append(all, target)
			continue
		}
		cidrPart := strings.Trim(target[:lastColon], "[]")
		port := target[lastColon+1:]
		_, ipNet, err := net.ParseCIDR(cidrPart)
		if err != nil {
			all = append(all, target)
			continue
		}
		for ip := append(net.IP(nil), ipNet.IP...); ipNet.Contains(ip); {
			if maxTargets > 0 && len(all) >= maxTargets {
				return nil, fmt.Errorf("target %q exceeds max targets limit (%d)", raw, maxTargets)
			}
			all = append(all, net.JoinHostPort(ip.String(), port))
			for j := len(ip) - 1; j >= 0; j-- {
				ip[j]++
				if ip[j] != 0 {
					break
				}
			}
		}
	}
	return all, nil
}
