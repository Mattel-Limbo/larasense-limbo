package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Mattel-Limbo/larasense-limbo/internal/config"
	"github.com/Mattel-Limbo/larasense-limbo/internal/context"
	"github.com/Mattel-Limbo/larasense-limbo/internal/git"
	"github.com/Mattel-Limbo/larasense-limbo/internal/output"
	"github.com/Mattel-Limbo/larasense-limbo/internal/reviewer"
	"github.com/Mattel-Limbo/larasense-limbo/internal/scanner"
	"github.com/Mattel-Limbo/larasense-limbo/internal/ui"
)

var (
	flagScanPath      string
	flagScanFiles     []string
	flagScanModified  bool
	flagScanJSON      bool
	flagScanFormat    string
	flagScanVerbose   bool
	flagScanNoCache   bool
	flagScanFresh     bool
	flagScanFix       bool
	flagScanApply     bool
	flagScanYes       bool
	flagScanAuto      bool
	flagScanDryRun    bool
	flagScanPreview   bool
	flagScanGitBranch string
	flagScanPatch     string
)

var scanCmd = &cobra.Command{
	Use:   "scan [path|file...]",
	Short: "Full codebase audit of all Laravel files",
	Long: `Scan all Laravel files in the project (not just changed files) and
send them to an AI provider for code review.

Useful for initial adoption on existing projects or periodic full audits.
Files are filtered using the same include/exclude patterns from config.

Arguments are auto-detected: directories are scanned recursively,
files are scanned individually.`,
	RunE: runScan,
}

func init() {
	scanCmd.Flags().StringVar(&flagScanPath, "path", ".", "Root directory to scan")
	scanCmd.Flags().StringSliceVar(&flagScanFiles, "file", nil, "Scan specific file(s) instead of directory (repeatable)")
	scanCmd.Flags().BoolVar(&flagScanModified, "modified", false, "Scan only modified/staged files from git status")
	scanCmd.Flags().BoolVar(&flagScanJSON, "json", false, "Output results as JSON (shorthand for --format json)")
	scanCmd.Flags().StringVar(&flagScanFormat, "format", "human", "Output format: human, json, github")
	scanCmd.Flags().BoolVar(&flagScanVerbose, "verbose", false, "Show detailed request/response logs for debugging")
	scanCmd.Flags().BoolVar(&flagScanNoCache, "no-cache", false, "Skip cache and re-scan all files")
	scanCmd.Flags().BoolVar(&flagScanFresh, "fresh", false, "Skip cache and re-scan all files (alias for --no-cache)")
	scanCmd.Flags().BoolVar(&flagScanFix, "fix", false, "Generate code fix suggestions for issues")
	scanCmd.Flags().BoolVar(&flagScanApply, "apply", false, "Apply fixes interactively — prompts y/n per fix (requires --fix)")
	scanCmd.Flags().BoolVar(&flagScanYes, "yes", false, "Apply all fixes without prompting (requires --fix --apply)")
	scanCmd.Flags().BoolVar(&flagScanAuto, "auto", false, "Generate fixes and apply all without prompting (alias for --fix --apply --yes)")
	scanCmd.Flags().BoolVar(&flagScanDryRun, "dry-run", false, "Show what fixes would be applied without writing to files (requires --fix)")
	scanCmd.Flags().BoolVar(&flagScanPreview, "preview", false, "Preview fixes without applying (alias for --fix --dry-run)")
	scanCmd.Flags().StringVar(&flagScanGitBranch, "git-branch", "", "Create a git branch before applying fixes for easy revert (requires --fix --apply)")
	scanCmd.Flags().StringVar(&flagScanPatch, "patch", "", "Write fixes as unified diff to file (requires --fix)")

	rootCmd.AddCommand(scanCmd)
}

func runScan(cmd *cobra.Command, args []string) error {
	if flagScanAuto {
		flagScanFix = true
		flagScanApply = true
		flagScanYes = true
	}
	if flagScanPreview {
		flagScanFix = true
		flagScanDryRun = true
	}
	if flagScanFresh {
		flagScanNoCache = true
	}

	if len(args) > 0 && !flagScanModified {
		var argFiles []string
		for _, arg := range args {
			info, err := os.Stat(arg)
			if err != nil {
				return fmt.Errorf("path not found: %s", arg)
			}
			if info.IsDir() {
				flagScanPath = arg
			} else {
				argFiles = append(argFiles, arg)
			}
		}
		if len(argFiles) > 0 {
			flagScanFiles = append(flagScanFiles, argFiles...)
		}
	}

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
		return fmt.Errorf("--dry-run/--preview and --apply/--auto cannot be used together")
	}
	if flagScanGitBranch != "" && !flagScanApply {
		return fmt.Errorf("--git-branch requires --apply")
	}
	if flagScanPatch != "" && !flagScanFix {
		return fmt.Errorf("--patch requires --fix")
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	s := scanner.New(cfg, flagScanPath)
	var files []context.FileContext
	var target string

	if flagScanModified {
		modifiedPaths, gitErr := git.GetModifiedFiles()
		if gitErr != nil {
			return fmt.Errorf("getting modified files: %w", gitErr)
		}
		if len(modifiedPaths) == 0 {
			fmt.Println("No modified files found in git working tree.")
			return nil
		}
		target = fmt.Sprintf("%d modified files from git status", len(modifiedPaths))
		files, err = s.ScanFiles(modifiedPaths)
		if err != nil {
			return fmt.Errorf("scanning modified files: %w", err)
		}
	} else if len(flagScanFiles) > 0 {
		target = fmt.Sprintf("%d specific file(s)", len(flagScanFiles))
		files, err = s.ScanFiles(flagScanFiles)
		if err != nil {
			return fmt.Errorf("scanning files: %w", err)
		}
	} else {
		target = fmt.Sprintf("directory: %s", flagScanPath)
		files, err = s.Scan()
		if err != nil {
			return fmt.Errorf("scanning files: %w", err)
		}
	}

	if len(files) == 0 {
		fmt.Println("No Laravel files found matching config filters.")
		return nil
	}

	mode := scanModeLabel()
	ui.PrintHeader("🔍 Larasense Limbo — Scan", map[string]string{
		"Provider": fmt.Sprintf("%s (%s)", cfg.Provider.Model, cfg.Provider.BaseURL),
		"Target":   target,
		"Files":    fmt.Sprintf("%d Laravel files", len(files)),
		"Mode":     mode,
	}, []string{"Provider", "Target", "Files", "Mode"})

	startTime := time.Now()

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

	elapsed := time.Since(startTime)
	if err := handleFixOutput(result, flagScanFix, flagScanApply, flagScanYes, flagScanDryRun, flagScanGitBranch, flagScanPatch, elapsed); err != nil {
		return err
	}

	for _, issue := range result.Issues {
		if issue.Severity == "high" {
			os.Exit(1)
		}
	}

	return nil
}

func scanModeLabel() string {
	var parts []string
	if flagScanAuto {
		parts = append(parts, "--auto")
	} else {
		if flagScanFix {
			parts = append(parts, "--fix")
		}
		if flagScanApply {
			parts = append(parts, "--apply")
		}
		if flagScanPreview {
			parts = append(parts, "--preview")
		}
	}
	if flagScanFresh || flagScanNoCache {
		parts = append(parts, "--fresh")
	}
	if flagScanModified {
		parts = append(parts, "--modified")
	}
	if len(parts) == 0 {
		return "scan (review only)"
	}
	return "scan " + fmt.Sprintf("%s", strings.Join(parts, " "))
}
