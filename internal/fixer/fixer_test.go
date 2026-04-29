package fixer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Mattel-Limbo/larasense-limbo/internal/ai"
)

func TestGeneratePatch_SingleFix(t *testing.T) {
	issues := []ai.Issue{
		{
			Title:    "Mass Assignment",
			File:     "app/Http/Controllers/UserController.php",
			Line:     15,
			Severity: "high",
			Fix: &ai.Fix{
				StartLine: 15,
				EndLine:   15,
				Before:    "        $user = User::create($request->all());",
				After:     "        $user = User::create($request->validated());",
			},
		},
	}

	patch := GeneratePatch(issues)

	if !strings.Contains(patch, "--- a/app/Http/Controllers/UserController.php") {
		t.Error("patch should contain file header")
	}
	if !strings.Contains(patch, "-        $user = User::create($request->all());") {
		t.Error("patch should contain removed line")
	}
	if !strings.Contains(patch, "+        $user = User::create($request->validated());") {
		t.Error("patch should contain added line")
	}
}

func TestGeneratePatch_NoFixes(t *testing.T) {
	issues := []ai.Issue{
		{Title: "No fix", Severity: "low"},
	}

	patch := GeneratePatch(issues)

	if patch != "" {
		t.Errorf("expected empty patch for issues without fixes, got %q", patch)
	}
}

func TestCountFixable(t *testing.T) {
	issues := []ai.Issue{
		{Title: "Has fix", Fix: &ai.Fix{Before: "a", After: "b"}},
		{Title: "No fix"},
		{Title: "Has fix 2", Fix: &ai.Fix{Before: "c", After: "d"}},
	}

	if got := CountFixable(issues); got != 2 {
		t.Errorf("CountFixable() = %d, want 2", got)
	}
}

func TestCountFixable_None(t *testing.T) {
	issues := []ai.Issue{
		{Title: "No fix 1"},
		{Title: "No fix 2"},
	}

	if got := CountFixable(issues); got != 0 {
		t.Errorf("CountFixable() = %d, want 0", got)
	}
}

func TestApplyFixes(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.php")

	content := "line1\nline2\nline3\nline4\nline5\n"
	os.WriteFile(filePath, []byte(content), 0644)

	issues := []ai.Issue{
		{
			Title: "Fix line 3",
			File:  filePath,
			Fix: &ai.Fix{
				StartLine: 3,
				EndLine:   3,
				Before:    "line3",
				After:     "fixed_line3",
			},
		},
	}

	applied, err := ApplyFixes(issues)
	if err != nil {
		t.Fatalf("ApplyFixes() error: %v", err)
	}

	if len(applied) != 1 {
		t.Fatalf("expected 1 applied fix, got %d", len(applied))
	}

	result, _ := os.ReadFile(filePath)
	if !strings.Contains(string(result), "fixed_line3") {
		t.Error("file should contain fixed line")
	}
	if strings.Contains(string(result), "\nline3\n") {
		t.Error("file should not contain original line3")
	}
}

func TestApplyFixes_BeforeMismatch(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.php")

	content := "line1\nline2\nline3\n"
	os.WriteFile(filePath, []byte(content), 0644)

	issues := []ai.Issue{
		{
			Title: "Fix with wrong before",
			File:  filePath,
			Fix: &ai.Fix{
				StartLine: 2,
				EndLine:   2,
				Before:    "wrong_content",
				After:     "fixed",
			},
		},
	}

	applied, err := ApplyFixes(issues)
	if err != nil {
		t.Fatalf("ApplyFixes() error: %v", err)
	}

	if len(applied) != 0 {
		t.Error("should skip fix when before content doesn't match")
	}

	result, _ := os.ReadFile(filePath)
	if string(result) != content {
		t.Error("file should be unchanged when before doesn't match")
	}
}

func TestWritePatch(t *testing.T) {
	dir := t.TempDir()
	patchPath := filepath.Join(dir, "fixes.patch")

	issues := []ai.Issue{
		{
			Title: "Test fix",
			File:  "app/test.php",
			Fix: &ai.Fix{
				StartLine: 1,
				EndLine:   1,
				Before:    "old",
				After:     "new",
			},
		},
	}

	err := WritePatch(issues, patchPath)
	if err != nil {
		t.Fatalf("WritePatch() error: %v", err)
	}

	content, err := os.ReadFile(patchPath)
	if err != nil {
		t.Fatalf("reading patch file: %v", err)
	}

	if !strings.Contains(string(content), "--- a/app/test.php") {
		t.Error("patch file should contain diff header")
	}
}

func TestWritePatch_NoFixes(t *testing.T) {
	dir := t.TempDir()
	patchPath := filepath.Join(dir, "fixes.patch")

	issues := []ai.Issue{{Title: "No fix"}}

	err := WritePatch(issues, patchPath)
	if err == nil {
		t.Error("expected error when no fixable issues")
	}
}
