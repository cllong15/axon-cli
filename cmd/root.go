package cmd

import (
	"axon-cli/internal/checker"
	"axon-cli/internal/config"
	"axon-cli/internal/output"
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

var versionNum = "1.1.1"
var stdin io.Reader = os.Stdin

func Execute(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	cfg := config.DefaultConfig()
	var targetFile string
	var showVersion bool

	fs := flag.NewFlagSet("axon", flag.ContinueOnError)
	fs.SetOutput(stderr)

	fs.IntVar(&cfg.WorkerCount, "workers", 10, "Number of concurrent workers (default: 10)")
	fs.IntVar(&cfg.WorkerCount, "w", 10, "number of concurrent workers (shorthand)")

	fs.DurationVar(&cfg.Timeout, "timeout", 5*time.Second, "Request timeout (default: 5s)")
	fs.DurationVar(&cfg.Timeout, "t", 5*time.Second, "Request timeout (shorthand)")

	fs.StringVar(&cfg.OutputFormat, "format", "table", "Output format (table|json - default: table)")
	fs.StringVar(&cfg.OutputFormat, "f", "table", "Output format (table|json)")

	fs.StringVar(&targetFile, "file", "", "Path to file containing URLs (default: \"\")")
	fs.StringVar(&targetFile, "i", "", "Path to file containing URLs (shorthand)")

	fs.BoolVar(&showVersion, "version", false, "Display version information (default: false)")
	fs.BoolVar(&showVersion, "v", false, "Display version information (shorthand)")

	fs.StringVar(&cfg.Method, "method", "GET", "HTTP method (default: GET)(GET|HEAD|POST|PUT|DELETE|OPTIONS)")
	fs.StringVar(&cfg.Method, "m", "GET", "HTTP method (shorthand)")

	fs.BoolVar(&cfg.FailOnError, "fail-on-error", false, "Exit with non-zero code if any target fails (default: false)")
	fs.BoolVar(&cfg.FailOnError, "l", false, "Exit with non-zero code if any target fails (shorthand)")

	fs.BoolVar(&cfg.OnlyErrors, "only-errors", false, "display only failed or anomalous targets (default: false)")
	fs.BoolVar(&cfg.OnlyErrors, "e", false, "display only failed or anomalous targets (shorthand)")

	fs.DurationVar(&cfg.Threshold, "threshold", 0*time.Millisecond, "threshold for slow latency in ms (default: 0ms)")
	fs.DurationVar(&cfg.Threshold, "T", 0*time.Millisecond, "threshold for slow latency in ms(shorthand)")

	fs.BoolVar(&cfg.NumericStatus, "numeric-Status", false, "display http status codes for tcp connections (default: false)")
	fs.BoolVar(&cfg.NumericStatus, "num", false, "display http status codes for tcp connections (shorthand)")

	fs.IntVar(&cfg.MaxTargets, "maxTargets", 1024, "Maximum targets: default 1024")
	fs.IntVar(&cfg.MaxTargets, "max", 1024, "Maximum targets: default 1024 (shorthand)")

	fs.BoolVar(&cfg.AutoConfirm, "autoConfirm", false, "Auto confirm any prompts (default: false)")
	fs.BoolVar(&cfg.AutoConfirm, "y", false, "Auto confirm any prompts (shorthand)")

	fs.IntVar(&cfg.BatchSize, "batchSize", 256, "size of connection batches if any (default: 256)")
	fs.IntVar(&cfg.BatchSize, "b", 256, "size of connection batches if any (shorthand)")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if showVersion {
		fmt.Fprintf(stdout, "axon version %s\n", versionNum)
		return nil
	}

	cfg.Targets = append(cfg.Targets, fs.Args()...)
	if targetFile != "" {
		f, err := os.Open(targetFile)
		if err != nil {
			return err
		}
		defer f.Close()
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" && !strings.HasPrefix(line, "#") {
				cfg.Targets = append(cfg.Targets, line)
			}
		}
		if err := scanner.Err(); err != nil {
			return err
		}
	}
	if len(cfg.Targets) == 0 {
		return errors.New("no target URLs provided; usage: axon [urls...]")
	}
	if err := cfg.Validate(); err != nil {
		return err
	}
	exptargs, err := checker.ExpandTargets(cfg.Targets, cfg.MaxTargets)
	if err != nil {
		return err
	}
	cfg.Targets = exptargs
	if len(cfg.Targets) > 256 && !cfg.AutoConfirm {
		fmt.Fprintf(stderr, "Auditing %d targets. Recommend fewer targets or adjusting max targets...\nContinue? (y/N)", len(cfg.Targets))
		scanner := bufio.NewScanner(stdin)
		if !scanner.Scan() {
			return errors.New("audit canceled by user")
		}
		ans := strings.ToLower(strings.TrimSpace(scanner.Text()))
		if ans != "y" && ans != "yes" {
			return errors.New("audit cancelled by user")
		}
	}
	var raw []checker.CheckResult
	for i := 0; i < len(cfg.Targets); i += cfg.BatchSize {
		end := i + cfg.BatchSize
		if end > len(cfg.Targets) {
			end = len(cfg.Targets)
		}
		batch := cfg.Targets[i:end]
		raw = append(raw, checker.RunPool(ctx, batch, cfg.WorkerCount, cfg.Timeout, cfg.Method)...)
	}
	var results []checker.CheckResult
	var formatErr error
	for _, r := range raw {
		if cfg.Threshold > 0 && cfg.Threshold < r.Latency {
			r.Slow = true
		}
		if !cfg.OnlyErrors || r.Error != "" || r.StatusCode >= 400 || r.Slow {
			results = append(results, r)
		}
	}
	if cfg.OutputFormat == "json" {
		formatErr = output.FormatJSON(stdout, results)
	} else {
		formatErr = output.FormatTable(stdout, cfg, results)
	}
	if formatErr != nil {
		return formatErr
	}
	if cfg.FailOnError {
		for _, r := range results {
			if r.Error != "" || r.StatusCode >= 400 || r.Slow {
				return errors.New("one or more targets failed health check")
			}
		}
	}
	return nil
}
