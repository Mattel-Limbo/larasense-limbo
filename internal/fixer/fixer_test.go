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

func TestApplyFixes_ApplyAll(t *testing.T) {
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

	result, err := ApplyFixes(issues, ApplyAll, strings.NewReader(""))
	if err != nil {
		t.Fatalf("ApplyFixes() error: %v", err)
	}

	if len(result.Applied) != 1 {
		t.Fatalf("expected 1 applied fix, got %d", len(result.Applied))
	}

	fileContent, _ := os.ReadFile(filePath)
	if !strings.Contains(string(fileContent), "fixed_line3") {
		t.Error("file should contain fixed line")
	}
	if strings.Contains(string(fileContent), "\nline3\n") {
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
				Before:    "completely_different_content_xyz",
				After:     "fixed",
			},
		},
	}

	result, err := ApplyFixes(issues, ApplyAll, strings.NewReader(""))
	if err != nil {
		t.Fatalf("ApplyFixes() error: %v", err)
	}

	if len(result.Applied) != 0 {
		t.Error("should skip fix when before content doesn't match")
	}
	if len(result.Skipped) != 1 {
		t.Errorf("expected 1 skipped fix, got %d", len(result.Skipped))
	}

	fileContent, _ := os.ReadFile(filePath)
	if string(fileContent) != content {
		t.Error("file should be unchanged when before doesn't match")
	}
}

func TestApplyFixes_FuzzyWhitespace(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.php")

	// File has tabs, AI gives spaces
	content := "line1\n\t\tline2\nline3\n"
	os.WriteFile(filePath, []byte(content), 0644)

	issues := []ai.Issue{
		{
			Title: "Fix with whitespace diff",
			File:  filePath,
			Fix: &ai.Fix{
				StartLine: 2,
				EndLine:   2,
				Before:    "        line2",
				After:     "        fixed_line2",
			},
		},
	}

	result, err := ApplyFixes(issues, ApplyAll, strings.NewReader(""))
	if err != nil {
		t.Fatalf("ApplyFixes() error: %v", err)
	}

	// Should match via normalized whitespace
	if len(result.Applied) != 1 {
		t.Errorf("expected 1 applied fix (fuzzy whitespace match), got %d applied, %d skipped",
			len(result.Applied), len(result.Skipped))
		for _, s := range result.Skipped {
			t.Logf("  skipped: %s (%s)", s.Title, s.Reason)
		}
	}
}

func TestApplyFixes_Interactive_Accept(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.php")

	content := "line1\nline2\nline3\n"
	os.WriteFile(filePath, []byte(content), 0644)

	issues := []ai.Issue{
		{
			Title:    "Fix line 2",
			File:     filePath,
			Severity: "high",
			Fix: &ai.Fix{
				StartLine: 2,
				EndLine:   2,
				Before:    "line2",
				After:     "fixed_line2",
			},
		},
	}

	// Simulate user typing "y"
	input := strings.NewReader("y\n")
	result, err := ApplyFixes(issues, ApplyInteractive, input)
	if err != nil {
		t.Fatalf("ApplyFixes() error: %v", err)
	}

	if len(result.Applied) != 1 {
		t.Errorf("expected 1 applied fix, got %d", len(result.Applied))
	}
}

func TestApplyFixes_Interactive_Reject(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.php")

	content := "line1\nline2\nline3\n"
	os.WriteFile(filePath, []byte(content), 0644)

	issues := []ai.Issue{
		{
			Title:    "Fix line 2",
			File:     filePath,
			Severity: "high",
			Fix: &ai.Fix{
				StartLine: 2,
				EndLine:   2,
				Before:    "line2",
				After:     "fixed_line2",
			},
		},
	}

	// Simulate user typing "n"
	input := strings.NewReader("n\n")
	result, err := ApplyFixes(issues, ApplyInteractive, input)
	if err != nil {
		t.Fatalf("ApplyFixes() error: %v", err)
	}

	if len(result.Applied) != 0 {
		t.Error("should not apply when user says no")
	}
	if len(result.Skipped) != 1 {
		t.Errorf("expected 1 skipped, got %d", len(result.Skipped))
	}

	fileContent, _ := os.ReadFile(filePath)
	if string(fileContent) != content {
		t.Error("file should be unchanged when user rejects")
	}
}

func TestMatchBefore(t *testing.T) {
	tests := []struct {
		name     string
		actual   string
		expected string
		want     bool
	}{
		{"exact", "hello world", "hello world", true},
		{"trimmed", "  hello  ", "hello", true},
		{"normalized", "\t\thello  ", "        hello", true},
		{"contains", "prefix $businessAccounts = BusinessAccount::all(); suffix", "$businessAccounts = BusinessAccount::all();", true},
		{"mismatch", "hello", "goodbye", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchBefore(tt.actual, tt.expected)
			if got != tt.want {
				t.Errorf("matchBefore(%q, %q) = %v, want %v", tt.actual, tt.expected, got, tt.want)
			}
		})
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

// TestApplyFixes_AILineOffset simulates the real-world case where AI reports
// line 28 but the actual code starts at line 30 (GeneratePOSAccountCommand.php scenario).
func TestApplyFixes_AILineOffset(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "GeneratePOSAccountCommand.php")

	// Real file content — code starts at line 30, not 28
	content := `<?php

namespace App\Console\Commands;

use App\Models\BusinessAccount;
use App\Jobs\SyncPOSAccountJob;
use Illuminate\Console\Command;

class GeneratePOSAccountCommand extends Command
{
    protected $signature = 'pos:generate-account';

    protected $description = 'Command description';

    /**
     * Execute the console command.
     */
    public function handle()
    {
        $businessAccounts = BusinessAccount::all();
        
        foreach ($businessAccounts as $businessAccount) {
            SyncPOSAccountJob::dispatch($businessAccount);
        }
    }
}
`
	os.WriteFile(filePath, []byte(content), 0644)

	// AI says line 28 but actual code is at line 20-24
	issues := []ai.Issue{
		{
			Title:    "Unbounded query without pagination",
			File:     filePath,
			Line:     28,
			Severity: "medium",
			Fix: &ai.Fix{
				StartLine: 28,
				EndLine:   33,
				Before: `        $businessAccounts = BusinessAccount::all();
        
        foreach ($businessAccounts as $businessAccount) {
            SyncPOSAccountJob::dispatch($businessAccount);
        }`,
				After: `        BusinessAccount::chunk(100, function ($businessAccounts) {
            foreach ($businessAccounts as $businessAccount) {
                SyncPOSAccountJob::dispatch($businessAccount);
            }
        });`,
			},
		},
	}

	result, err := ApplyFixes(issues, ApplyAll, strings.NewReader(""))
	if err != nil {
		t.Fatalf("ApplyFixes() error: %v", err)
	}

	if len(result.Applied) != 1 {
		t.Errorf("expected 1 applied fix (AI line offset), got %d applied, %d skipped",
			len(result.Applied), len(result.Skipped))
		for _, s := range result.Skipped {
			t.Logf("  skipped: %s (%s)", s.Title, s.Reason)
		}
	}

	if len(result.Applied) > 0 {
		fileContent, _ := os.ReadFile(filePath)
		if !strings.Contains(string(fileContent), "chunk(100") {
			t.Error("file should contain chunk() after fix")
		}
		if strings.Contains(string(fileContent), "BusinessAccount::all()") {
			t.Error("file should not contain BusinessAccount::all() after fix")
		}
	}
}

// TestApplyFixes_IndentationMismatch simulates AI giving different indentation than file.
func TestApplyFixes_IndentationMismatch(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.php")

	// File uses 4-space indent
	content := "<?php\nclass Test {\n    public function handle() {\n        $items = Item::all();\n        foreach ($items as $item) {\n            process($item);\n        }\n    }\n}\n"
	os.WriteFile(filePath, []byte(content), 0644)

	// AI gives 8-space indent in before
	issues := []ai.Issue{
		{
			Title:    "Unbounded query",
			File:     filePath,
			Severity: "medium",
			Fix: &ai.Fix{
				StartLine: 4,
				EndLine:   7,
				Before:    "        $items = Item::all();\n        foreach ($items as $item) {\n            process($item);\n        }",
				After:     "        Item::chunk(100, function ($items) {\n            foreach ($items as $item) {\n                process($item);\n            }\n        });",
			},
		},
	}

	result, err := ApplyFixes(issues, ApplyAll, strings.NewReader(""))
	if err != nil {
		t.Fatalf("ApplyFixes() error: %v", err)
	}

	if len(result.Applied) != 1 {
		t.Errorf("expected 1 applied fix (indentation mismatch), got %d applied, %d skipped",
			len(result.Applied), len(result.Skipped))
		for _, s := range result.Skipped {
			t.Logf("  skipped: %s (%s)", s.Title, s.Reason)
		}
	}
}
