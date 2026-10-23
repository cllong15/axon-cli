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
	for _, r := range results {
		status := fmt.Sprintf("%d", r.StatusCode)
		if r.StatusCode == 0 {
			status = "ERR"
		}
		fmt.Fprintf(tw,
			"%s\t%s\t%s\t%s\n",
			r.URL,
			status,
			r.Latency.Round(time.Millisecond),
			r.Error)
	}
	return tw.Flush()
}
