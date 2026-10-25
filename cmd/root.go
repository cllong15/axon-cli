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

var versionNum = "1.0.3"

func Execute(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	cfg := config.DefaultConfig()
	var targetFile string
	var showVersion bool

	fs := flag.NewFlagSet("axon", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.IntVar(&cfg.WorkerCount, "workers", 10, "Number of concurrent workers")
	fs.IntVar(&cfg.WorkerCount, "w", 10, "number of concurrent workers (shorthand)")

	fs.DurationVar(&cfg.Timeout, "timeout", 5*time.Second, "Request timeout")
	fs.DurationVar(&cfg.Timeout, "t", 5*time.Second, "Request timeout (shorthand)")

	fs.StringVar(&cfg.OutputFormat, "format", "table", "Output format (table|json)")
	fs.StringVar(&cfg.OutputFormat, "f", "table", "Output format (table|json)")

	fs.StringVar(&targetFile, "file", "", "Path to file containing URLs")
	fs.StringVar(&targetFile, "i", "", "Path to file containing URLs")

	fs.BoolVar(&showVersion, "version", false, "Display version information")
	fs.BoolVar(&showVersion, "v", false, "Display version information (shorthand)")

	fs.StringVar(&cfg.Method, "method", "GET", "HTTP method (GET|HEAD|POST|PUT|DELETE|OPTIONS)")
	fs.StringVar(&cfg.Method, "m", "GET", "HTTP method (shorthand)")

	fs.BoolVar(&cfg.FailOnError, "fail-on-error", false, "Exit with non-zero code if any target fails")
	fs.BoolVar(&cfg.FailOnError, "fail", false, "Exit with non-zero code if any target fails (shorthand)")

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
	results := checker.RunPool(ctx, cfg.Targets, cfg.WorkerCount, cfg.Timeout, cfg.Method)
	var formatErr error
	if cfg.OutputFormat == "json" {
		formatErr = output.FormatJSON(stdout, results)
	} else {
		formatErr = output.FormatTable(stdout, results)
	}
	if formatErr != nil {
		return formatErr
	}
	if cfg.FailOnError {
		for _, r := range results {
			if r.Error != "" || r.StatusCode >= 400 {
				return errors.New("one or more targets failed health check")
			}
		}
	}
	return nil
}
