package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// GetDiff runs `git diff base...head` and returns the raw diff output.
func GetDiff(base, head string) (string, error) {
	diffRange := fmt.Sprintf("%s...%s", base, head)
	out, err := runGit("diff", diffRange)
	if err != nil {
		return "", fmt.Errorf("git diff %s failed: %w", diffRange, err)
	}
	return out, nil
}

// GetFileContent retrieves the content of a file at a given git ref.
// If ref is empty, it reads from the working tree.
func GetFileContent(ref, filePath string) (string, error) {
	if ref == "" {
		ref = "HEAD"
	}
	target := fmt.Sprintf("%s:%s", ref, filePath)
	out, err := runGit("show", target)
	if err != nil {
		return "", fmt.Errorf("git show %s failed: %w", target, err)
	}
	return out, nil
}

// GetSurroundingLines retrieves lines around a specific line number from a file at a ref.
// It returns ±contextLines around the target line.
func GetSurroundingLines(ref, filePath string, targetLine, contextLines int) (string, error) {
	content, err := GetFileContent(ref, filePath)
	if err != nil {
		return "", err
	}

	lines := strings.Split(content, "\n")
	start := targetLine - contextLines - 1
	end := targetLine + contextLines

	if start < 0 {
		start = 0
	}
	if end > len(lines) {
		end = len(lines)
	}

	var result strings.Builder
	for i := start; i < end; i++ {
		fmt.Fprintf(&result, "%d: %s\n", i+1, lines[i])
	}
	return result.String(), nil
}

func GetCurrentBranch() (string, error) {
	out, err := runGit("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", fmt.Errorf("getting current branch: %w", err)
	}
	return strings.TrimSpace(out), nil
}

func IsCleanWorkingTree() bool {
	out, err := runGit("status", "--porcelain")
	if err != nil {
		return false
	}
	return strings.TrimSpace(out) == ""
}

func CreateAndCheckoutBranch(name string) error {
	_, err := runGit("checkout", "-b", name)
	if err != nil {
		return fmt.Errorf("creating branch %s: %w", name, err)
	}
	return nil
}

func CheckoutBranch(name string) error {
	_, err := runGit("checkout", name)
	if err != nil {
		return fmt.Errorf("checking out branch %s: %w", name, err)
	}
	return nil
}

// GetModifiedFiles returns file paths that are modified, staged, or untracked in the working tree.
func GetModifiedFiles() ([]string, error) {
	out, err := runGit("status", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("git status failed: %w", err)
	}

	if strings.TrimSpace(out) == "" {
		return nil, nil
	}

	var files []string
	seen := make(map[string]bool)

	for _, line := range strings.Split(out, "\n") {
		if len(line) < 4 {
			continue
		}
		// git status --porcelain format: XY filename
		// X = staging area, Y = working tree
		// Skip deleted files (D in either position)
		x, y := line[0], line[1]
		if x == 'D' || y == 'D' {
			continue
		}

		path := strings.TrimSpace(line[3:])

		// Handle renamed files: "R  old -> new"
		if strings.Contains(path, " -> ") {
			parts := strings.Split(path, " -> ")
			path = parts[len(parts)-1]
		}

		if !seen[path] {
			seen[path] = true
			files = append(files, path)
		}
	}

	return files, nil
}

func runGit(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = err.Error()
		}
		return "", fmt.Errorf("%s", errMsg)
	}
	return stdout.String(), nil
}
