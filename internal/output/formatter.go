package output

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Mattel-Limbo/larasense-limbo/internal/reviewer"
)

// FormatJSON returns the review result as formatted JSON.
func FormatJSON(result *reviewer.Result) (string, error) {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshaling result to JSON: %w", err)
	}
	return string(data), nil
}

// FormatHuman returns the review result as a human-readable string.
func FormatHuman(result *reviewer.Result) string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString("╔══════════════════════════════════════════════════════════════╗\n")
	sb.WriteString("║              Laravel AI Code Review Results                 ║\n")
	sb.WriteString("╚══════════════════════════════════════════════════════════════╝\n")
	sb.WriteString("\n")

	// Summary
	fmt.Fprintf(&sb, "  %s\n\n", result.Summary)

	if len(result.Issues) == 0 {
		sb.WriteString("  ✅ No issues found. Your code looks good!\n\n")
		return sb.String()
	}

	// Group issues by file
	grouped := groupByFile(result.Issues)

	fileIdx := 0
	for file, issues := range grouped {
		fileIdx++
		fmt.Fprintf(&sb, "  📄 %s\n", file)
		sb.WriteString("  " + strings.Repeat("─", 58) + "\n")

		for i, issue := range issues {
			severityIcon := severityIcon(issue.Severity)
			fmt.Fprintf(&sb, "\n  %s [%s] #%d: %s\n", severityIcon, strings.ToUpper(issue.Severity), i+1, issue.Title)

			if issue.Line > 0 {
				fmt.Fprintf(&sb, "     Line: %d\n", issue.Line)
			}

			fmt.Fprintf(&sb, "     %s\n", issue.Description)

			if issue.Suggestion != "" {
				fmt.Fprintf(&sb, "     💡 Suggestion: %s\n", issue.Suggestion)
			}

			if issue.Fix != nil {
				sb.WriteString("     ┌─ Before:\n")
				for _, line := range strings.Split(issue.Fix.Before, "\n") {
					fmt.Fprintf(&sb, "     │ \033[31m- %s\033[0m\n", line)
				}
				sb.WriteString("     ├─ After:\n")
				for _, line := range strings.Split(issue.Fix.After, "\n") {
					fmt.Fprintf(&sb, "     │ \033[32m+ %s\033[0m\n", line)
				}
				sb.WriteString("     └─\n")
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString("══════════════════════════════════════════════════════════════\n")

	return sb.String()
}

// FormatGitHubAnnotations returns issues as GitHub Actions workflow commands.
// Format: ::warning file={file},line={line}::{title}: {description}
// See: https://docs.github.com/en/actions/writing-workflows/choosing-what-your-workflow-does/workflow-commands-for-github-actions
func FormatGitHubAnnotations(result *reviewer.Result) string {
	var sb strings.Builder

	for _, issue := range result.Issues {
		level := "warning"
		switch strings.ToLower(issue.Severity) {
		case "high":
			level = "error"
		case "low":
			level = "notice"
		}

		file := issue.File
		if file == "" {
			file = "(unknown)"
		}

		if issue.Line > 0 {
			fmt.Fprintf(&sb, "::%s file=%s,line=%d::%s: %s\n", level, file, issue.Line, issue.Title, issue.Description)
		} else {
			fmt.Fprintf(&sb, "::%s file=%s::%s: %s\n", level, file, issue.Title, issue.Description)
		}
	}

	return sb.String()
}

// groupByFile groups issues by their file path.
func groupByFile(issues []reviewer.Issue) map[string][]reviewer.Issue {
	grouped := make(map[string][]reviewer.Issue)
	for _, issue := range issues {
		file := issue.File
		if file == "" {
			file = "(unknown)"
		}
		grouped[file] = append(grouped[file], issue)
	}
	return grouped
}

// severityIcon returns an emoji icon for the severity level.
func severityIcon(severity string) string {
	switch strings.ToLower(severity) {
	case "high":
		return "🔴"
	case "medium":
		return "🟡"
	case "low":
		return "🔵"
	default:
		return "⚪"
	}
}
