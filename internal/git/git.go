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
