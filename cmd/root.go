package cmd

import (
	"axon-cli/internal/checker"
	"axon-cli/internal/config"
	"axon-cli/internal/output"
	"context"
	"errors"
	"time"

	"github.com/spf13/cobra"
)

var cfg = config.DefaultConfig()

var rootCmd = &cobra.Command{
	Use:   "axon [urls...]",
	Short: "High-performance concurrent endpoint health auditor",
	RunE:  run,
}

func ExecuteContext(ctx context.Context) error {
	return rootCmd.ExecuteContext(ctx)
}

func init() {
	rootCmd.Flags().IntVarP(&cfg.WorkerCount, "workers", "w", 10, "Number of concurrent workers")
	rootCmd.Flags().DurationVarP(&cfg.Timeout, "timeout", "t", 5*time.Second, "Request timeout")
	rootCmd.Flags().StringVarP(&cfg.OutputFormat, "format", "f", "table", "Output format (table|json)")
}

func run(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		cfg.Targets = args
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
