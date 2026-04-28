package reviewer

import (
	"fmt"
	"log"
	"strings"

	"github.com/Mattel-Limbo/larasense-limbo/internal/ai"
	"github.com/Mattel-Limbo/larasense-limbo/internal/config"
	"github.com/Mattel-Limbo/larasense-limbo/internal/context"
	"github.com/Mattel-Limbo/larasense-limbo/internal/diff"
	"github.com/Mattel-Limbo/larasense-limbo/internal/git"
)

// Issue is a re-export of ai.Issue for use by consumers of this package.
type Issue = ai.Issue

// Result holds the final review output.
type Result struct {
	Issues     []Issue `json:"issues"`
	FilesCount int     `json:"files_analyzed"`
	Summary    string  `json:"summary"`
}

// Reviewer orchestrates the full code review pipeline.
type Reviewer struct {
	cfg     *config.Config
	verbose bool
}

// New creates a new Reviewer.
func New(cfg *config.Config, verbose bool) *Reviewer {
	return &Reviewer{cfg: cfg, verbose: verbose}
}

// Run executes the full review pipeline: diff → parse → context → AI → result.
func (r *Reviewer) Run(base, head string) (*Result, error) {
	// Step 1: Get git diff
	log.Printf("Getting diff: %s...%s", base, head)
	rawDiff, err := git.GetDiff(base, head)
	if err != nil {
		return nil, fmt.Errorf("getting git diff: %w", err)
	}

	if strings.TrimSpace(rawDiff) == "" {
		return &Result{
			Summary: "No changes found between the specified refs.",
		}, nil
	}

	// Step 2: Parse diff
	log.Println("Parsing diff...")
	files, err := diff.Parse(rawDiff)
	if err != nil {
		return nil, fmt.Errorf("parsing diff: %w", err)
	}

	if len(files) == 0 {
		return &Result{
			Summary: "No files found in diff.",
		}, nil
	}

	log.Printf("Found %d changed files in diff", len(files))

	// Step 3: Build context
	log.Println("Building Laravel context...")
	builder := context.NewBuilder(r.cfg, head)
	reviewCtx, err := builder.Build(files)
	if err != nil {
		return nil, fmt.Errorf("building context: %w", err)
	}

	if len(reviewCtx.Files) == 0 {
		return &Result{
			Summary: "No Laravel-relevant files found in the diff. Only .php and .blade.php files in app/, routes/, resources/views/ are analyzed.",
		}, nil
	}

	log.Printf("Analyzing %d Laravel files", len(reviewCtx.Files))

	// Step 4: Build diff text for AI
	var diffText strings.Builder
	for _, f := range reviewCtx.Files {
		fmt.Fprintf(&diffText, "--- %s ---\n%s\n", f.Path, f.DiffText)
	}

	contextText := reviewCtx.FormatForAI()

	// Step 5: Send to AI
	log.Println("Sending to AI provider for analysis...")
	client := ai.NewClient(&r.cfg.Provider, r.verbose)
	aiResp, err := client.Analyze(diffText.String(), contextText)
	if err != nil {
		return nil, fmt.Errorf("AI analysis: %w", err)
	}

	// Step 6: Apply filters
	issues := filterIssues(aiResp.Issues, r.cfg)

	// Step 7: Build result
	result := &Result{
		Issues:     issues,
		FilesCount: len(reviewCtx.Files),
		Summary:    buildSummary(issues, len(reviewCtx.Files)),
	}

	return result, nil
}

// filterIssues applies max_issues and severity_threshold from config.
func filterIssues(issues []ai.Issue, cfg *config.Config) []ai.Issue {
	threshold := cfg.Review.SeverityThreshold
	maxIssues := cfg.Review.MaxIssues

	// Filter by severity threshold
	var filtered []ai.Issue
	for _, issue := range issues {
		if meetsThreshold(issue.Severity, threshold) {
			filtered = append(filtered, issue)
		}
	}

	// Limit count
	if maxIssues > 0 && len(filtered) > maxIssues {
		filtered = filtered[:maxIssues]
	}

	return filtered
}

// meetsThreshold checks if an issue's severity meets the minimum threshold.
func meetsThreshold(severity, threshold string) bool {
	levels := map[string]int{
		"low":    1,
		"medium": 2,
		"high":   3,
	}

	issueLvl, ok := levels[strings.ToLower(severity)]
	if !ok {
		issueLvl = 1
	}

	threshLvl, ok := levels[strings.ToLower(threshold)]
	if !ok {
		threshLvl = 1
	}

	return issueLvl >= threshLvl
}

// buildSummary creates a human-readable summary of the review.
func buildSummary(issues []ai.Issue, filesCount int) string {
	if len(issues) == 0 {
		return fmt.Sprintf("Reviewed %d file(s). No issues found. Great job!", filesCount)
	}

	var high, medium, low int
	for _, issue := range issues {
		switch strings.ToLower(issue.Severity) {
		case "high":
			high++
		case "medium":
			medium++
		default:
			low++
		}
	}

	return fmt.Sprintf(
		"Reviewed %d file(s). Found %d issue(s): %d high, %d medium, %d low.",
		filesCount, len(issues), high, medium, low,
	)
}
