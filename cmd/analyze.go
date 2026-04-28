package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"

	"github.com/Mattel-Limbo/larasense-limbo/internal/config"
	"github.com/Mattel-Limbo/larasense-limbo/internal/output"
	"github.com/Mattel-Limbo/larasense-limbo/internal/reviewer"
)

var (
	flagBase    string
	flagHead    string
	flagJSON    bool
	flagFormat  string
	flagVerbose bool
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyze git diff for Laravel best practices",
	Long: `Analyze the git diff between two refs and send it to an AI provider
for code review focused on Laravel best practices.

The command reads the .larasense-limbo.yml config file from the current
directory and uses the configured AI provider to analyze changed files.`,
	RunE: runAnalyze,
}

func init() {
	analyzeCmd.Flags().StringVar(&flagBase, "base", "origin/main", "Base ref for diff comparison")
	analyzeCmd.Flags().StringVar(&flagHead, "head", "HEAD", "Head ref for diff comparison")
	analyzeCmd.Flags().BoolVar(&flagJSON, "json", false, "Output results as JSON (shorthand for --format json)")
	analyzeCmd.Flags().StringVar(&flagFormat, "format", "human", "Output format: human, json, github")
	analyzeCmd.Flags().BoolVar(&flagVerbose, "verbose", false, "Show detailed request/response logs for debugging")

	rootCmd.AddCommand(analyzeCmd)
}

func runAnalyze(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	log.Printf("Using AI provider: %s (model: %s)", cfg.Provider.BaseURL, cfg.Provider.Model)
	log.Printf("Comparing %s...%s", flagBase, flagHead)

	// Run the review pipeline
	rev := reviewer.New(cfg, flagVerbose)
	result, err := rev.Run(flagBase, flagHead)
	if err != nil {
		return fmt.Errorf("review failed: %w", err)
	}

	// Resolve output format (--json is shorthand for --format json)
	format := flagFormat
	if flagJSON {
		format = "json"
	}

	// Output results
	switch format {
	case "json":
		jsonOut, err := output.FormatJSON(result)
		if err != nil {
			return fmt.Errorf("formatting JSON output: %w", err)
		}
		fmt.Fprintln(os.Stdout, jsonOut)
	case "github":
		fmt.Fprint(os.Stdout, output.FormatGitHubAnnotations(result))
	default:
		fmt.Fprint(os.Stdout, output.FormatHuman(result))
	}

	// Exit with non-zero if high-severity issues found
	for _, issue := range result.Issues {
		if issue.Severity == "high" {
			os.Exit(1)
		}
	}

	return nil
}
