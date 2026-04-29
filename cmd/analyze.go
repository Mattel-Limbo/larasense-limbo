package cmd

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Mattel-Limbo/larasense-limbo/internal/config"
	gh "github.com/Mattel-Limbo/larasense-limbo/internal/github"
	"github.com/Mattel-Limbo/larasense-limbo/internal/output"
	"github.com/Mattel-Limbo/larasense-limbo/internal/reviewer"
)

var (
	flagBase     string
	flagHead     string
	flagJSON     bool
	flagFormat   string
	flagVerbose  bool
	flagNoCache  bool
	flagGitHubPR string
	flagFix      bool
	flagApply    bool
	flagYes      bool
	flagPatch    string
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
	analyzeCmd.Flags().BoolVar(&flagNoCache, "no-cache", false, "Skip cache and re-review all files")
	analyzeCmd.Flags().StringVar(&flagGitHubPR, "github-pr", "", "Post results as PR comment (format: owner/repo#number)")
	analyzeCmd.Flags().BoolVar(&flagFix, "fix", false, "Generate code fix suggestions for issues")
	analyzeCmd.Flags().BoolVar(&flagApply, "apply", false, "Apply fixes interactively — prompts y/n per fix (requires --fix)")
	analyzeCmd.Flags().BoolVar(&flagYes, "yes", false, "Apply all fixes without prompting (requires --fix --apply)")
	analyzeCmd.Flags().StringVar(&flagPatch, "patch", "", "Write fixes as unified diff to file (requires --fix)")

	rootCmd.AddCommand(analyzeCmd)
}

func runAnalyze(cmd *cobra.Command, args []string) error {
	if flagApply && !flagFix {
		return fmt.Errorf("--apply requires --fix")
	}
	if flagYes && !flagApply {
		return fmt.Errorf("--yes requires --apply")
	}
	if flagPatch != "" && !flagFix {
		return fmt.Errorf("--patch requires --fix")
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	log.Printf("Using AI provider: %s (model: %s)", cfg.Provider.BaseURL, cfg.Provider.Model)
	log.Printf("Comparing %s...%s", flagBase, flagHead)

	rev := reviewer.New(cfg, flagVerbose, flagNoCache, flagFix)
	result, err := rev.Run(flagBase, flagHead)
	if err != nil {
		return fmt.Errorf("review failed: %w", err)
	}

	format := flagFormat
	if flagJSON {
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

	if err := handleFixOutput(result, flagFix, flagApply, flagYes, flagPatch); err != nil {
		return err
	}

	if flagGitHubPR != "" {
		if err := postToGitHubPR(flagGitHubPR, result); err != nil {
			log.Printf("Warning: failed to post GitHub PR comment: %v", err)
		}
	}

	for _, issue := range result.Issues {
		if issue.Severity == "high" {
			os.Exit(1)
		}
	}

	return nil
}

// postToGitHubPR posts review results as a comment on a GitHub pull request.
// prRef format: "owner/repo#number"
func postToGitHubPR(prRef string, result *reviewer.Result) error {
	// Parse "owner/repo#number"
	parts := strings.SplitN(prRef, "#", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid --github-pr format %q (expected: owner/repo#number)", prRef)
	}

	repoParts := strings.SplitN(parts[0], "/", 2)
	if len(repoParts) != 2 {
		return fmt.Errorf("invalid --github-pr format %q (expected: owner/repo#number)", prRef)
	}

	prNumber, err := strconv.Atoi(parts[1])
	if err != nil {
		return fmt.Errorf("invalid PR number %q: %w", parts[1], err)
	}

	owner := repoParts[0]
	repo := repoParts[1]

	// Get GitHub token from environment
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return fmt.Errorf("GITHUB_TOKEN environment variable is required for --github-pr")
	}

	// Convert reviewer issues to github issues
	var ghIssues []gh.Issue
	for _, issue := range result.Issues {
		ghIssues = append(ghIssues, gh.Issue{
			Title:       issue.Title,
			Description: issue.Description,
			File:        issue.File,
			Line:        issue.Line,
			Severity:    issue.Severity,
			Suggestion:  issue.Suggestion,
		})
	}

	client := gh.NewClient(token)
	if err := client.PostReviewComment(owner, repo, prNumber, ghIssues, result.Summary); err != nil {
		return err
	}

	log.Printf("Posted review comment to %s PR #%d", parts[0], prNumber)
	return nil
}
