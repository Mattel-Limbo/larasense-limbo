package fixer

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/Mattel-Limbo/larasense-limbo/internal/ai"
)

type AppliedFix struct {
	File  string
	Line  int
	Title string
}

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

func WritePatch(issues []ai.Issue, outputPath string) error {
	patch := GeneratePatch(issues)
	if patch == "" {
		return fmt.Errorf("no fixable issues found")
	}
	return os.WriteFile(outputPath, []byte(patch), 0644)
}

func ApplyFixes(issues []ai.Issue) ([]AppliedFix, error) {
	byFile := groupFixesByFile(issues)
	var applied []AppliedFix

	for file, fixes := range byFile {
		content, err := os.ReadFile(file)
		if err != nil {
			return applied, fmt.Errorf("reading %s: %w", file, err)
		}

		lines := strings.Split(string(content), "\n")

		// Apply bottom-up to preserve line numbers
		sortFixesDesc(fixes)

		for _, issue := range fixes {
			f := issue.Fix
			if f.StartLine < 1 || f.EndLine > len(lines) || f.StartLine > f.EndLine {
				continue
			}

			actualBefore := strings.Join(lines[f.StartLine-1:f.EndLine], "\n")
			if strings.TrimSpace(actualBefore) != strings.TrimSpace(f.Before) {
				continue
			}

			afterLines := strings.Split(f.After, "\n")
			newLines := make([]string, 0, len(lines)-(f.EndLine-f.StartLine+1)+len(afterLines))
			newLines = append(newLines, lines[:f.StartLine-1]...)
			newLines = append(newLines, afterLines...)
			newLines = append(newLines, lines[f.EndLine:]...)
			lines = newLines

			applied = append(applied, AppliedFix{
				File:  file,
				Line:  f.StartLine,
				Title: issue.Title,
			})
		}

		if err := os.WriteFile(file, []byte(strings.Join(lines, "\n")), 0644); err != nil {
			return applied, fmt.Errorf("writing %s: %w", file, err)
		}
	}

	return applied, nil
}

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
