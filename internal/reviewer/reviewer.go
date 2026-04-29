package reviewer

import (
	"fmt"
	"log"
	"strings"

	"github.com/Mattel-Limbo/larasense-limbo/internal/ai"
	"github.com/Mattel-Limbo/larasense-limbo/internal/cache"
	"github.com/Mattel-Limbo/larasense-limbo/internal/config"
	"github.com/Mattel-Limbo/larasense-limbo/internal/context"
	"github.com/Mattel-Limbo/larasense-limbo/internal/diff"
	"github.com/Mattel-Limbo/larasense-limbo/internal/git"
)

type Issue = ai.Issue

type Result struct {
	Issues     []Issue `json:"issues"`
	FilesCount int     `json:"files_analyzed"`
	Summary    string  `json:"summary"`
	Mode       string  `json:"mode"`
}

type Reviewer struct {
	cfg     *config.Config
	verbose bool
	noCache bool
	fixMode bool
}

func New(cfg *config.Config, verbose, noCache, fixMode bool) *Reviewer {
	return &Reviewer{cfg: cfg, verbose: verbose, noCache: noCache, fixMode: fixMode}
}

// Run executes the diff-based review pipeline.
func (r *Reviewer) Run(base, head string) (*Result, error) {
	log.Printf("Getting diff: %s...%s", base, head)
	rawDiff, err := git.GetDiff(base, head)
	if err != nil {
		return nil, fmt.Errorf("getting git diff: %w", err)
	}

	if strings.TrimSpace(rawDiff) == "" {
		return &Result{
			Summary: "No changes found between the specified refs.",
			Mode:    "diff",
		}, nil
	}

	log.Println("Parsing diff...")
	files, err := diff.Parse(rawDiff)
	if err != nil {
		return nil, fmt.Errorf("parsing diff: %w", err)
	}

	if len(files) == 0 {
		return &Result{
			Summary: "No files found in diff.",
			Mode:    "diff",
		}, nil
	}

	log.Printf("Found %d changed files in diff", len(files))

	log.Println("Building Laravel context...")
	builder := context.NewBuilder(r.cfg, head)
	reviewCtx, err := builder.Build(files)
	if err != nil {
		return nil, fmt.Errorf("building context: %w", err)
	}

	if len(reviewCtx.Files) == 0 {
		return &Result{
			Summary: "No Laravel-relevant files found in the diff. Only .php and .blade.php files in app/, routes/, resources/views/ are analyzed.",
			Mode:    "diff",
		}, nil
	}

	log.Printf("Analyzing %d Laravel files", len(reviewCtx.Files))

	return r.review(reviewCtx.Files, "diff")
}

// RunScan executes the full-scan review pipeline.
func (r *Reviewer) RunScan(files []context.FileContext) (*Result, error) {
	if len(files) == 0 {
		return &Result{
			Summary: "No Laravel files found to scan.",
			Mode:    "scan",
		}, nil
	}

	log.Printf("Scanning %d Laravel files", len(files))

	return r.review(files, "scan")
}

func (r *Reviewer) review(allFiles []context.FileContext, mode string) (*Result, error) {
	filesToReview, cachedIssues, reviewCache := r.filterCached(allFiles)

	if len(filesToReview) == 0 && len(cachedIssues) == 0 {
		return &Result{
			FilesCount: len(allFiles),
			Summary:    fmt.Sprintf("All %d file(s) unchanged since last review. Nothing to analyze.", len(allFiles)),
			Mode:       mode,
		}, nil
	}

	// Collect all issues (cached + new from AI)
	var allIssues []Issue
	allIssues = append(allIssues, cachedIssues...)

	if len(filesToReview) > 0 {
		if len(filesToReview) < len(allFiles) {
			log.Printf("Reviewing %d of %d files (%d cached)", len(filesToReview), len(allFiles), len(allFiles)-len(filesToReview))
		}

		newIssues, err := r.analyzeInBatches(filesToReview, mode)
		if err != nil {
			return nil, err
		}

		allIssues = append(allIssues, newIssues...)

		// Update cache with new AI results per file
		if reviewCache != nil {
			for _, f := range filesToReview {
				// Collect issues belonging to this file
				var fileIssues []cache.CachedIssue
			for _, issue := range newIssues {
				if issue.File == f.Path {
					ci := cache.CachedIssue{
						Title:       issue.Title,
						Description: issue.Description,
						File:        issue.File,
						Line:        issue.Line,
						Severity:    issue.Severity,
						Suggestion:  issue.Suggestion,
					}
					if issue.Fix != nil {
						ci.Fix = &cache.CachedFix{
							StartLine: issue.Fix.StartLine,
							EndLine:   issue.Fix.EndLine,
							Before:    issue.Fix.Before,
							After:     issue.Fix.After,
						}
					}
					fileIssues = append(fileIssues, ci)
				}
				}
				reviewCache.UpdateWithIssues(f.Path, f.DiffText, fileIssues)
			}
		}
	}

	// Save cache
	if reviewCache != nil {
		if err := reviewCache.Save(); err != nil {
			log.Printf("Warning: failed to save cache: %v", err)
		}
	}

	issues := filterIssues(allIssues, r.cfg)

	return &Result{
		Issues:     issues,
		FilesCount: len(allFiles),
		Summary:    buildSummary(issues, len(allFiles), mode),
		Mode:       mode,
	}, nil
}

// filterCached separates files into those needing review and those with cached results.
// Returns: filesToReview, cachedIssues, cache instance (nil if caching disabled).
func (r *Reviewer) filterCached(files []context.FileContext) ([]context.FileContext, []Issue, *cache.Cache) {
	if r.noCache {
		return files, nil, nil
	}

	reviewCache := cache.Load()
	var filesToReview []context.FileContext
	var cachedIssues []Issue

	for _, f := range files {
		if issues, hit := reviewCache.GetCachedIssues(f.Path, f.DiffText); hit {
			// If fix mode is on but cached issues lack fix data, re-analyze
			if r.fixMode && len(issues) > 0 && !cachedHasFixes(issues) {
				filesToReview = append(filesToReview, f)
				continue
			}
			log.Printf("Using cached results for %s (%d issue(s))", f.Path, len(issues))
			for _, ci := range issues {
				issue := Issue{
					Title:       ci.Title,
					Description: ci.Description,
					File:        ci.File,
					Line:        ci.Line,
					Severity:    ci.Severity,
					Suggestion:  ci.Suggestion,
				}
				if ci.Fix != nil {
					issue.Fix = &ai.Fix{
						StartLine: ci.Fix.StartLine,
						EndLine:   ci.Fix.EndLine,
						Before:    ci.Fix.Before,
						After:     ci.Fix.After,
					}
				}
				cachedIssues = append(cachedIssues, issue)
			}
		} else if reviewCache.HasChanged(f.Path, f.DiffText) {
			filesToReview = append(filesToReview, f)
		} else {
			log.Printf("Skipping %s (unchanged since last review)", f.Path)
		}
	}

	return filesToReview, cachedIssues, reviewCache
}

func (r *Reviewer) analyzeInBatches(files []context.FileContext, mode string) ([]ai.Issue, error) {
	batches := batchFiles(files)
	log.Printf("Split %d files into %d batch(es)", len(files), len(batches))

	client := ai.NewClient(&r.cfg.Provider, r.verbose)

	var systemPrompt string
	if r.fixMode {
		systemPrompt = ai.BuildDiffFixPrompt()
		if mode == "scan" {
			systemPrompt = ai.BuildScanFixPrompt()
		}
	} else {
		systemPrompt = ai.BuildDiffPrompt()
		if mode == "scan" {
			systemPrompt = ai.BuildScanPrompt()
		}
	}
	if r.cfg.Review.CustomPrompt != "" {
		systemPrompt += "\n\nAdditional instructions from the user:\n" + r.cfg.Review.CustomPrompt
	}

	var allIssues []ai.Issue

	for i, batch := range batches {
		log.Printf("Analyzing batch %d/%d (%d files)...", i+1, len(batches), len(batch))

		ctx := &context.ReviewContext{Files: batch}

		var userContent string
		if mode == "scan" {
			userContent = ctx.FormatForScan()
		} else {
			var diffText strings.Builder
			for _, f := range batch {
				fmt.Fprintf(&diffText, "--- %s ---\n%s\n", f.Path, f.DiffText)
			}
			userContent = fmt.Sprintf("## File Context\n\n%s\n\n## Git Diff\n\n%s", ctx.FormatForAI(), diffText.String())
		}

		resp, err := client.AnalyzeWithPrompt(systemPrompt, userContent)
		if err != nil {
			return nil, fmt.Errorf("batch %d/%d failed: %w", i+1, len(batches), err)
		}

		allIssues = append(allIssues, resp.Issues...)
	}

	return allIssues, nil
}

func filterIssues(issues []ai.Issue, cfg *config.Config) []ai.Issue {
	threshold := cfg.Review.SeverityThreshold
	maxIssues := cfg.Review.MaxIssues

	var filtered []ai.Issue
	for _, issue := range issues {
		if meetsThreshold(issue.Severity, threshold) {
			filtered = append(filtered, issue)
		}
	}

	if maxIssues > 0 && len(filtered) > maxIssues {
		filtered = filtered[:maxIssues]
	}

	return filtered
}

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

func buildSummary(issues []ai.Issue, filesCount int, mode string) string {
	prefix := "Reviewed"
	if mode == "scan" {
		prefix = "Scanned"
	}

	if len(issues) == 0 {
		return fmt.Sprintf("%s %d file(s). No issues found. Great job!", prefix, filesCount)
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
		"%s %d file(s). Found %d issue(s): %d high, %d medium, %d low.",
		prefix, filesCount, len(issues), high, medium, low,
	)
}

func cachedHasFixes(issues []cache.CachedIssue) bool {
	for _, ci := range issues {
		if ci.Fix != nil {
			return true
		}
	}
	return false
}
