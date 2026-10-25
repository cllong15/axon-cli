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

var versionNum = "1.0.2"

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
	results := checker.RunPool(ctx, cfg.Targets, cfg.WorkerCount, cfg.Timeout)
	if cfg.OutputFormat == "json" {
		return output.FormatJSON(stdout, results)
	}
	return output.FormatTable(stdout, results)
}
