package output

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/larasense/larasense-limbo/internal/reviewer"
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
		}
		sb.WriteString("\n")
	}

	sb.WriteString("══════════════════════════════════════════════════════════════\n")

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
