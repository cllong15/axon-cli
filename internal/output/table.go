package output

import (
	"axon-cli/internal/checker"
	"fmt"
	"io"
	"text/tabwriter"
	"time"
)

func FormatTable(w io.Writer, results []checker.CheckResult) error {
	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
	fmt.Fprintln(tw, "URL\tSTATUS\tLATENCY\tERROR")
	var totalLatency time.Duration
	var passed = 0
	var slow = 0
	for _, r := range results {
		if r.Error == "" && r.StatusCode < 400 {
			passed++
		}
		totalLatency += r.Latency
		status := fmt.Sprintf("%d", r.StatusCode)
		if r.StatusCode == 0 {
			status = "ERR"
		}
		if r.Slow {
			if status != "" {
				status = status + " "
			}
			status = status + "[slow]"
			slow++
		}
		fmt.Fprintf(tw,
			"%s\t%s\t%s\t%s\n",
			r.URL,
			status,
			r.Latency.Round(time.Millisecond),
			r.Error)
	}
	if err := tw.Flush(); err != nil {
		return err
	}
	if len(results) == 0 {
		return nil
	}
	failed := len(results) - passed
	avgLatency := (totalLatency / time.Duration(len(results))).Round(time.Millisecond)
	_, err := fmt.Fprintf(w, "\nTotal: %d | Passed: %d | Failed: %d | Slow: %d | Avg Latency: %s\n", len(results), passed, failed, slow, avgLatency)
	return err
}
