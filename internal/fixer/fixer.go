package fixer

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Mattel-Limbo/larasense-limbo/internal/ai"
)

// ApplyMode controls how fixes are applied.
type ApplyMode int

const (
	// ApplyNone only shows fix suggestions, does not apply.
	ApplyNone ApplyMode = iota
	// ApplyInteractive prompts the user for each fix (y/n).
	ApplyInteractive
	// ApplyAll applies all fixes without prompting.
	ApplyAll
	// ApplyDryRun shows what would be applied without writing to files.
	ApplyDryRun
)

// AppliedFix records a successfully applied fix.
type AppliedFix struct {
	File  string
	Line  int
	Title string
}

// SkippedFix records a fix that was skipped and why.
type SkippedFix struct {
	File   string
	Line   int
	Title  string
	Reason string
}

// ApplyResult holds the outcome of an apply operation.
type ApplyResult struct {
	Applied []AppliedFix
	Skipped []SkippedFix
}

// GeneratePatch creates a unified diff string from fixable issues.
func GeneratePatch(issues []ai.Issue) string {
	byFile := groupFixesByFile(issues)
	var sb strings.Builder

	for file, fixes := range byFile {
		sortFixesAsc(fixes)

		sb.WriteString(fmt.Sprintf("--- a/%s\n", file))
		sb.WriteString(fmt.Sprintf("+++ b/%s\n", file))

		for _, issue := range fixes {
			f := issue.Fix
			beforeLines := strings.Split(f.Before, "\n")
			afterLines := strings.Split(f.After, "\n")

			sb.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n",
				f.StartLine, len(beforeLines),
				f.StartLine, len(afterLines)))

			for _, line := range beforeLines {
				sb.WriteString("-" + line + "\n")
			}
			for _, line := range afterLines {
				sb.WriteString("+" + line + "\n")
			}
		}
	}

	return sb.String()
}

// WritePatch writes a unified diff patch file.
func WritePatch(issues []ai.Issue, outputPath string) error {
	patch := GeneratePatch(issues)
	if patch == "" {
		return fmt.Errorf("no fixable issues found")
	}
	return os.WriteFile(outputPath, []byte(patch), 0644)
}

// ApplyFixes applies fixes to source files.
// Fixes are collected per file, validated against the original content, then applied
// in descending line order (bottom-up) in a single pass to avoid line-shift conflicts.
func ApplyFixes(issues []ai.Issue, mode ApplyMode, reader io.Reader) (*ApplyResult, error) {
	byFile := groupFixesByFile(issues)
	result := &ApplyResult{}
	inputScanner := bufio.NewScanner(reader)

	for file, fixes := range byFile {
		content, err := os.ReadFile(file)
		if err != nil {
			for _, issue := range fixes {
				result.Skipped = append(result.Skipped, SkippedFix{
					File: file, Line: issue.Fix.StartLine, Title: issue.Title,
					Reason: fmt.Sprintf("cannot read file: %v", err),
				})
			}
			continue
		}
		lines := strings.Split(string(content), "\n")

		// Phase 1: Validate and resolve all fixes against the ORIGINAL file content
		type resolvedFix struct {
			issue    ai.Issue
			start    int
			end      int
			accepted bool
		}
		var resolved []resolvedFix

		for _, issue := range fixes {
			f := issue.Fix
			severity := strings.ToUpper(issue.Severity)

			if f.StartLine < 1 || f.StartLine > f.EndLine {
				result.Skipped = append(result.Skipped, SkippedFix{
					File: file, Line: f.StartLine, Title: issue.Title,
					Reason: fmt.Sprintf("invalid line range %d-%d", f.StartLine, f.EndLine),
				})
				continue
			}

			endLine := f.EndLine
			if endLine > len(lines) {
				endLine = len(lines)
			}

			actualBefore := strings.Join(lines[f.StartLine-1:endLine], "\n")
			matched := matchBefore(actualBefore, f.Before)

			if !matched {
				found, newStart, newEnd := searchNearby(lines, f.Before, f.StartLine, 15)
				if found {
					matched = true
					log.Printf("[fix] %s:%d — matched at nearby lines %d-%d", file, f.StartLine, newStart, newEnd)
					f.StartLine = newStart
					endLine = newEnd
				}
			}

			if !matched {
				found, newStart, newEnd := searchByContent(lines, f.Before)
				if found {
					matched = true
					log.Printf("[fix] %s:%d — found by content search at lines %d-%d", file, f.StartLine, newStart, newEnd)
					f.StartLine = newStart
					endLine = newEnd
				}
			}

			if !matched {
				result.Skipped = append(result.Skipped, SkippedFix{
					File: file, Line: f.StartLine, Title: issue.Title,
					Reason: "code at target lines doesn't match AI's 'before' — file may have changed since analysis",
				})
				continue
			}

			if mode == ApplyDryRun {
				printDryRunFix(file, f.StartLine, severity, issue.Title, f.Before, f.After)
				result.Applied = append(result.Applied, AppliedFix{
					File: file, Line: f.StartLine, Title: issue.Title,
				})
				continue
			}

			accepted := true
			if mode == ApplyInteractive {
				accepted = promptFix(inputScanner, file, f.StartLine, severity, issue.Title, f.Before, f.After)
				if !accepted {
					result.Skipped = append(result.Skipped, SkippedFix{
						File: file, Line: f.StartLine, Title: issue.Title,
						Reason: "skipped by user",
					})
				}
			}

			if accepted {
				resolved = append(resolved, resolvedFix{
					issue: issue, start: f.StartLine, end: endLine, accepted: true,
				})
			}
		}

		if mode == ApplyDryRun || len(resolved) == 0 {
			continue
		}

		// Phase 2: Check for overlapping ranges among accepted fixes
		var toApply []resolvedFix
		sort.Slice(resolved, func(i, j int) bool {
			return resolved[i].start < resolved[j].start
		})
		for i, rf := range resolved {
			overlaps := false
			for j := 0; j < i; j++ {
				if toApply[j].accepted && rf.start <= toApply[j].end && rf.end >= toApply[j].start {
					overlaps = true
					break
				}
			}
			if overlaps {
				result.Skipped = append(result.Skipped, SkippedFix{
					File: file, Line: rf.start, Title: rf.issue.Title,
					Reason: "overlaps with another fix in the same file",
				})
			} else {
				toApply = append(toApply, rf)
			}
		}

		if len(toApply) == 0 {
			continue
		}

		// Phase 3: Apply all fixes bottom-up in a single pass (no line-shift conflicts)
		if err := backupFile(file); err != nil {
			log.Printf("[warning] could not backup %s: %v", file, err)
		}

		sort.Slice(toApply, func(i, j int) bool {
			return toApply[i].start > toApply[j].start
		})

		for _, rf := range toApply {
			afterLines := strings.Split(rf.issue.Fix.After, "\n")
			newLines := make([]string, 0, len(lines)-(rf.end-rf.start+1)+len(afterLines))
			newLines = append(newLines, lines[:rf.start-1]...)
			newLines = append(newLines, afterLines...)
			newLines = append(newLines, lines[rf.end:]...)
			lines = newLines

			result.Applied = append(result.Applied, AppliedFix{
				File: file, Line: rf.start, Title: rf.issue.Title,
			})
		}

		if err := os.WriteFile(file, []byte(strings.Join(lines, "\n")), 0644); err != nil {
			return result, fmt.Errorf("writing %s: %w", file, err)
		}
	}

	return result, nil
}

// promptFix displays a fix and asks the user to accept or skip.
func promptFix(scanner *bufio.Scanner, file string, line int, severity, title, before, after string) bool {
	icon := severityIcon(severity)

	fmt.Println()
	fmt.Printf("  ╔══════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("  ║  %s [%s] %s\n", icon, severity, title)
	fmt.Printf("  ║  📄 %s (line %d)\n", file, line)
	fmt.Printf("  ╠══════════════════════════════════════════════════════════════╣\n")

	// Before (red)
	fmt.Printf("  ║  \033[31m┌─ Before:\033[0m\n")
	for _, l := range strings.Split(before, "\n") {
		fmt.Printf("  ║  \033[31m│ - %s\033[0m\n", l)
	}

	// After (green)
	fmt.Printf("  ║  \033[32m├─ After:\033[0m\n")
	for _, l := range strings.Split(after, "\n") {
		fmt.Printf("  ║  \033[32m│ + %s\033[0m\n", l)
	}
	fmt.Printf("  ║  └─\n")

	fmt.Printf("  ╚══════════════════════════════════════════════════════════════╝\n")
	fmt.Printf("  Apply this fix? [\033[32my\033[0m/\033[31mn\033[0m/\033[33mq\033[0m(uit)] > ")

	if !scanner.Scan() {
		return false
	}

	answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
	switch answer {
	case "y", "yes":
		return true
	case "q", "quit":
		fmt.Println("  Aborting remaining fixes.")
		os.Exit(0)
		return false
	default:
		return false
	}
}

func printDryRunFix(file string, line int, severity, title, before, after string) {
	icon := severityIcon(severity)
	fmt.Println()
	fmt.Printf("  [DRY-RUN] %s [%s] %s\n", icon, severity, title)
	fmt.Printf("  📄 %s (line %d)\n", file, line)
	fmt.Printf("  \033[31m┌─ Would remove:\033[0m\n")
	for _, l := range strings.Split(before, "\n") {
		fmt.Printf("  \033[31m│ - %s\033[0m\n", l)
	}
	fmt.Printf("  \033[32m├─ Would add:\033[0m\n")
	for _, l := range strings.Split(after, "\n") {
		fmt.Printf("  \033[32m│ + %s\033[0m\n", l)
	}
	fmt.Printf("  └─\n")
}

// matchBefore checks if the actual file content matches the AI's "before" string.
// Uses progressively looser matching strategies.
func matchBefore(actual, expected string) bool {
	// Strategy 1: exact match
	if actual == expected {
		return true
	}

	// Strategy 2: trimmed match (leading/trailing whitespace)
	if strings.TrimSpace(actual) == strings.TrimSpace(expected) {
		return true
	}

	// Strategy 3: normalized whitespace per line (trim trailing spaces)
	actualNorm := normalizeLines(actual)
	expectedNorm := normalizeLines(expected)
	if actualNorm == expectedNorm {
		return true
	}

	// Strategy 4: stripped comparison — ignore ALL leading whitespace per line
	// This handles tabs vs spaces, different indentation levels
	if stripIndentation(actual) == stripIndentation(expected) {
		return true
	}

	// Strategy 5: actual contains expected (AI sometimes gives partial before)
	if len(expected) > 10 && strings.Contains(stripIndentation(actual), stripIndentation(expected)) {
		return true
	}

	return false
}

// stripIndentation removes all leading whitespace from each line and joins.
func stripIndentation(s string) string {
	lines := strings.Split(s, "\n")
	var stripped []string
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if trimmed != "" {
			stripped = append(stripped, trimmed)
		}
	}
	return strings.Join(stripped, "\n")
}

// searchNearby looks for the "before" content within ±radius lines of the target.
func searchNearby(lines []string, before string, targetLine, radius int) (bool, int, int) {
	beforeLines := strings.Split(before, "\n")
	// Remove empty trailing lines from before (AI often adds trailing newline)
	for len(beforeLines) > 0 && strings.TrimSpace(beforeLines[len(beforeLines)-1]) == "" {
		beforeLines = beforeLines[:len(beforeLines)-1]
	}
	span := len(beforeLines)
	if span == 0 {
		return false, 0, 0
	}

	start := targetLine - radius
	if start < 1 {
		start = 1
	}
	end := targetLine + radius
	if end > len(lines) {
		end = len(lines)
	}

	for i := start; i <= end-span+1; i++ {
		if i-1+span > len(lines) {
			break
		}
		candidate := strings.Join(lines[i-1:i-1+span], "\n")
		if matchBefore(candidate, before) {
			return true, i, i + span - 1
		}
	}

	return false, 0, 0
}

// searchByContent searches the entire file for the "before" block by matching
// the stripped content of each line. Handles cases where AI line number is completely wrong.
func searchByContent(lines []string, before string) (bool, int, int) {
	beforeLines := strings.Split(before, "\n")
	// Remove empty trailing lines
	for len(beforeLines) > 0 && strings.TrimSpace(beforeLines[len(beforeLines)-1]) == "" {
		beforeLines = beforeLines[:len(beforeLines)-1]
	}
	span := len(beforeLines)
	if span == 0 {
		return false, 0, 0
	}

	// Strip the first non-empty before line for anchor search
	var anchorContent string
	for _, bl := range beforeLines {
		stripped := strings.TrimSpace(bl)
		if stripped != "" && stripped != "{" && stripped != "}" {
			anchorContent = stripped
			break
		}
	}
	if anchorContent == "" {
		return false, 0, 0
	}

	// Scan file for anchor line, then verify full block
	for i := 0; i <= len(lines)-span; i++ {
		if strings.TrimSpace(lines[i]) == anchorContent || strings.Contains(strings.TrimSpace(lines[i]), anchorContent) {
			// Found anchor — check if full block matches from here
			candidate := strings.Join(lines[i:i+span], "\n")
			if matchBefore(candidate, before) {
				return true, i + 1, i + span // 1-indexed
			}

			// Try starting a few lines before the anchor (AI's before may start before the anchor)
			for offset := 1; offset <= 3; offset++ {
				startIdx := i - offset
				if startIdx < 0 {
					continue
				}
				endIdx := startIdx + span
				if endIdx > len(lines) {
					continue
				}
				candidate = strings.Join(lines[startIdx:endIdx], "\n")
				if matchBefore(candidate, before) {
					return true, startIdx + 1, endIdx
				}
			}
		}
	}

	return false, 0, 0
}

// normalizeLines trims each line and joins with single newline.
func normalizeLines(s string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " \t\r")
	}
	return strings.Join(lines, "\n")
}

func severityIcon(severity string) string {
	switch strings.ToUpper(severity) {
	case "HIGH":
		return "🔴"
	case "MEDIUM":
		return "🟡"
	case "LOW":
		return "🔵"
	default:
		return "⚪"
	}
}

// overlapsApplied checks if a fix range overlaps with any previously applied fix range.
func overlapsApplied(start, end int, applied [][2]int) bool {
	for _, r := range applied {
		aStart, aEnd := r[0], r[1]
		// Two ranges overlap if one starts before the other ends
		if start <= aEnd && end >= aStart {
			return true
		}
	}
	return false
}

// CountFixable returns the number of issues that have fix data.
func CountFixable(issues []ai.Issue) int {
	count := 0
	for _, issue := range issues {
		if issue.Fix != nil {
			count++
		}
	}
	return count
}

func groupFixesByFile(issues []ai.Issue) map[string][]ai.Issue {
	grouped := make(map[string][]ai.Issue)
	for _, issue := range issues {
		if issue.Fix != nil && issue.File != "" {
			grouped[issue.File] = append(grouped[issue.File], issue)
		}
	}
	return grouped
}

func sortFixesAsc(issues []ai.Issue) {
	sort.Slice(issues, func(i, j int) bool {
		return issues[i].Fix.StartLine < issues[j].Fix.StartLine
	})
}

func sortFixesDesc(issues []ai.Issue) {
	sort.Slice(issues, func(i, j int) bool {
		return issues[i].Fix.StartLine > issues[j].Fix.StartLine
	})
}

const backupDir = ".larasense-limbo-backup"

// backupFile saves a copy of the file before modification.
// Only backs up once per file — skips if backup already exists.
func backupFile(filePath string) error {
	backupPath := filepath.Join(backupDir, filePath)

	if _, err := os.Stat(backupPath); err == nil {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(backupPath), 0755); err != nil {
		return err
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	return os.WriteFile(backupPath, content, 0644)
}

func UndoFixes() ([]string, error) {
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("no backup found — nothing to undo")
	}

	var restored []string

	err := filepath.WalkDir(backupDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		relPath, err := filepath.Rel(backupDir, path)
		if err != nil {
			return err
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading backup %s: %w", relPath, err)
		}

		if err := os.WriteFile(relPath, content, 0644); err != nil {
			return fmt.Errorf("restoring %s: %w", relPath, err)
		}

		restored = append(restored, relPath)
		return nil
	})

	if err != nil {
		return restored, err
	}

	os.RemoveAll(backupDir)
	return restored, nil
}

func HasBackup() bool {
	info, err := os.Stat(backupDir)
	return err == nil && info.IsDir()
}
