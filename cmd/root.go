package cmd

import (
	"axon-cli/internal/checker"
	"axon-cli/internal/config"
	"axon-cli/internal/output"
	"bufio"
	"context"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var cfg = config.DefaultConfig()

var targetFile string

var rootCmd = &cobra.Command{
	Use:     "axon [urls...]",
	Version: "1.0.1",
	Short:   "High-performance concurrent endpoint health auditor",
	RunE:    run,
}

func ExecuteContext(ctx context.Context) error {
	return rootCmd.ExecuteContext(ctx)
}

func init() {
	rootCmd.Flags().IntVarP(&cfg.WorkerCount, "workers", "w", 10, "Number of concurrent workers")
	rootCmd.Flags().DurationVarP(&cfg.Timeout, "timeout", "t", 5*time.Second, "Request timeout")
	rootCmd.Flags().StringVarP(&cfg.OutputFormat, "format", "f", "table", "Output format (table|json)")
	rootCmd.Flags().StringVarP(&targetFile, "file", "i", "", "path to file containing URLs (one per line)")
}

func run(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		cfg.Targets = args
	}
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
	results := checker.RunPool(cmd.Context(), cfg.Targets, cfg.WorkerCount, cfg.Timeout)
	if cfg.OutputFormat == "json" {
		return output.FormatJSON(cmd.OutOrStdout(), results)
	}
	return output.FormatTable(cmd.OutOrStdout(), results)
}
