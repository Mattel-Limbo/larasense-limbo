package diff

import (
	"fmt"
	"strconv"
	"strings"
)

// FileDiff represents a single file's changes from a git diff.
type FileDiff struct {
	Path    string
	Hunks   []Hunk
	IsNew   bool
	IsDeleted bool
}

// Hunk represents a contiguous block of changes within a file.
type Hunk struct {
	StartLine int
	EndLine   int
	Lines     []DiffLine
}

// DiffLine represents a single line in a diff hunk.
type DiffLine struct {
	Number  int
	Content string
	Type    LineType
}

// LineType indicates whether a line was added, removed, or is context.
type LineType int

const (
	LineContext LineType = iota
	LineAdded
	LineRemoved
)

// Parse takes raw git diff output and returns a slice of FileDiff.
func Parse(raw string) ([]FileDiff, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	var files []FileDiff
	// Split by "diff --git" markers
	parts := splitByDiffHeaders(raw)

	for _, part := range parts {
		fd, err := parseFileDiff(part)
		if err != nil {
			return nil, fmt.Errorf("parsing file diff: %w", err)
		}
		if fd != nil {
			files = append(files, *fd)
		}
	}

	return files, nil
}

// splitByDiffHeaders splits raw diff output into per-file sections.
func splitByDiffHeaders(raw string) []string {
	const marker = "diff --git "
	var parts []string
	lines := strings.Split(raw, "\n")

	var current []string
	for _, line := range lines {
		if strings.HasPrefix(line, marker) {
			if len(current) > 0 {
				parts = append(parts, strings.Join(current, "\n"))
			}
			current = []string{line}
		} else {
			current = append(current, line)
		}
	}
	if len(current) > 0 {
		parts = append(parts, strings.Join(current, "\n"))
	}
	return parts
}

// parseFileDiff parses a single file's diff section.
func parseFileDiff(section string) (*FileDiff, error) {
	lines := strings.Split(section, "\n")
	if len(lines) == 0 {
		return nil, nil
	}

	fd := &FileDiff{}

	// Extract file path from the first line: "diff --git a/path b/path"
	firstLine := lines[0]
	if strings.HasPrefix(firstLine, "diff --git ") {
		parts := strings.Fields(firstLine)
		if len(parts) >= 4 {
			fd.Path = strings.TrimPrefix(parts[3], "b/")
		}
	}

	if fd.Path == "" {
		return nil, nil
	}

	// Check for new/deleted file
	for _, line := range lines {
		if strings.HasPrefix(line, "new file mode") {
			fd.IsNew = true
		}
		if strings.HasPrefix(line, "deleted file mode") {
			fd.IsDeleted = true
		}
	}

	// Parse hunks
	fd.Hunks = parseHunks(lines)

	return fd, nil
}

// parseHunks extracts all hunks from a file diff section.
func parseHunks(lines []string) []Hunk {
	var hunks []Hunk
	var currentHunk *Hunk
	lineNum := 0

	for _, line := range lines {
		if strings.HasPrefix(line, "@@ ") {
			// Parse hunk header: @@ -old,count +new,count @@
			newStart := parseHunkHeader(line)
			if currentHunk != nil {
				currentHunk.EndLine = lineNum
				hunks = append(hunks, *currentHunk)
			}
			lineNum = newStart
			currentHunk = &Hunk{StartLine: newStart}
			continue
		}

		if currentHunk == nil {
			continue
		}

		dl := DiffLine{Number: lineNum}

		switch {
		case strings.HasPrefix(line, "+"):
			dl.Type = LineAdded
			dl.Content = strings.TrimPrefix(line, "+")
			currentHunk.Lines = append(currentHunk.Lines, dl)
			lineNum++
		case strings.HasPrefix(line, "-"):
			dl.Type = LineRemoved
			dl.Content = strings.TrimPrefix(line, "-")
			currentHunk.Lines = append(currentHunk.Lines, dl)
			// Removed lines don't advance the new-file line counter
		case strings.HasPrefix(line, " "):
			dl.Type = LineContext
			dl.Content = strings.TrimPrefix(line, " ")
			currentHunk.Lines = append(currentHunk.Lines, dl)
			lineNum++
		default:
			// Skip binary/no-newline markers
			if strings.HasPrefix(line, "\\") {
				continue
			}
		}
	}

	if currentHunk != nil {
		currentHunk.EndLine = lineNum
		hunks = append(hunks, *currentHunk)
	}

	return hunks
}

// parseHunkHeader extracts the new-file start line from a hunk header.
// Format: @@ -oldStart,oldCount +newStart,newCount @@ optional context
func parseHunkHeader(header string) int {
	// Find the +N,M or +N part
	parts := strings.Split(header, "+")
	if len(parts) < 2 {
		return 1
	}

	numPart := parts[1]
	// Take everything before the next space or comma
	numPart = strings.Split(numPart, ",")[0]
	numPart = strings.Split(numPart, " ")[0]

	n, err := strconv.Atoi(strings.TrimSpace(numPart))
	if err != nil {
		return 1
	}
	return n
}

// ChangedLines returns only the added/modified line numbers for a FileDiff.
func (fd *FileDiff) ChangedLines() []int {
	var lines []int
	for _, hunk := range fd.Hunks {
		for _, dl := range hunk.Lines {
			if dl.Type == LineAdded {
				lines = append(lines, dl.Number)
			}
		}
	}
	return lines
}

// DiffText returns the raw diff text for this file (added/removed lines only).
func (fd *FileDiff) DiffText() string {
	var sb strings.Builder
	for _, hunk := range fd.Hunks {
		for _, dl := range hunk.Lines {
			switch dl.Type {
			case LineAdded:
				fmt.Fprintf(&sb, "+%s\n", dl.Content)
			case LineRemoved:
				fmt.Fprintf(&sb, "-%s\n", dl.Content)
			}
		}
	}
	return sb.String()
}
