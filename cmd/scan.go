package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"

	"github.com/Mattel-Limbo/larasense-limbo/internal/config"
	"github.com/Mattel-Limbo/larasense-limbo/internal/output"
	"github.com/Mattel-Limbo/larasense-limbo/internal/reviewer"
	"github.com/Mattel-Limbo/larasense-limbo/internal/scanner"
)

var (
	flagScanPath    string
	flagScanJSON    bool
	flagScanFormat  string
	flagScanVerbose bool
	flagScanNoCache bool
	flagScanFix     bool
	flagScanApply   bool
	flagScanYes     bool
	flagScanDryRun  bool
	flagScanPatch   string
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Full codebase audit of all Laravel files",
	Long: `Scan all Laravel files in the project (not just changed files) and
send them to an AI provider for code review.

Useful for initial adoption on existing projects or periodic full audits.
Files are filtered using the same include/exclude patterns from config.`,
	RunE: runScan,
}

func init() {
	scanCmd.Flags().StringVar(&flagScanPath, "path", ".", "Root directory to scan")
	scanCmd.Flags().BoolVar(&flagScanJSON, "json", false, "Output results as JSON (shorthand for --format json)")
	scanCmd.Flags().StringVar(&flagScanFormat, "format", "human", "Output format: human, json, github")
	scanCmd.Flags().BoolVar(&flagScanVerbose, "verbose", false, "Show detailed request/response logs for debugging")
	scanCmd.Flags().BoolVar(&flagScanNoCache, "no-cache", false, "Skip cache and re-scan all files")
	scanCmd.Flags().BoolVar(&flagScanFix, "fix", false, "Generate code fix suggestions for issues")
	scanCmd.Flags().BoolVar(&flagScanApply, "apply", false, "Apply fixes interactively — prompts y/n per fix (requires --fix)")
	scanCmd.Flags().BoolVar(&flagScanYes, "yes", false, "Apply all fixes without prompting (requires --fix --apply)")
	scanCmd.Flags().BoolVar(&flagScanDryRun, "dry-run", false, "Show what fixes would be applied without writing to files (requires --fix)")
	scanCmd.Flags().StringVar(&flagScanPatch, "patch", "", "Write fixes as unified diff to file (requires --fix)")

	rootCmd.AddCommand(scanCmd)
}

func runScan(cmd *cobra.Command, args []string) error {
	if flagScanApply && !flagScanFix {
		return fmt.Errorf("--apply requires --fix")
	}
	if flagScanYes && !flagScanApply {
		return fmt.Errorf("--yes requires --apply")
	}
	if flagScanDryRun && !flagScanFix {
		return fmt.Errorf("--dry-run requires --fix")
	}
	if flagScanDryRun && flagScanApply {
		return fmt.Errorf("--dry-run and --apply cannot be used together")
	}
	if flagScanPatch != "" && !flagScanFix {
		return fmt.Errorf("--patch requires --fix")
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	log.Printf("Using AI provider: %s (model: %s)", cfg.Provider.BaseURL, cfg.Provider.Model)
	log.Printf("Scanning directory: %s", flagScanPath)

	s := scanner.New(cfg, flagScanPath)
	files, err := s.Scan()
	if err != nil {
		return fmt.Errorf("scanning files: %w", err)
	}

	if len(files) == 0 {
		fmt.Println("No Laravel files found matching config filters.")
		return nil
	}

	log.Printf("Found %d Laravel files to scan", len(files))

	rev := reviewer.New(cfg, flagScanVerbose, flagScanNoCache, flagScanFix)
	result, err := rev.RunScan(files)
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	format := flagScanFormat
	if flagScanJSON {
		format = "json"
	}

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

	if err := handleFixOutput(result, flagScanFix, flagScanApply, flagScanYes, flagScanDryRun, flagScanPatch); err != nil {
		return err
	}

	for _, issue := range result.Issues {
		if issue.Severity == "high" {
			os.Exit(1)
		}
	}

	return nil
}
