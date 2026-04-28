package output

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Mattel-Limbo/larasense-limbo/internal/reviewer"
)

func TestFormatJSON_WithIssues(t *testing.T) {
	result := &reviewer.Result{
		Issues: []reviewer.Issue{
			{
				Title:       "Mass Assignment",
				Description: "Using $request->all()",
				File:        "app/Http/Controllers/UserController.php",
				Line:        15,
				Severity:    "high",
				Suggestion:  "Use validated()",
			},
		},
		FilesCount: 1,
		Summary:    "Reviewed 1 file(s). Found 1 issue(s): 1 high, 0 medium, 0 low.",
	}

	out, err := FormatJSON(result)
	if err != nil {
		t.Fatalf("FormatJSON() error: %v", err)
	}

	// Should be valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	// Should contain key fields
	if !strings.Contains(out, "Mass Assignment") {
		t.Error("JSON output should contain issue title")
	}
	if !strings.Contains(out, `"severity": "high"`) {
		t.Error("JSON output should contain severity")
	}
	if !strings.Contains(out, `"files_analyzed": 1`) {
		t.Error("JSON output should contain files_analyzed")
	}
}

func TestFormatJSON_NoIssues(t *testing.T) {
	result := &reviewer.Result{
		Issues:     []reviewer.Issue{},
		FilesCount: 3,
		Summary:    "Reviewed 3 file(s). No issues found. Great job!",
	}

	out, err := FormatJSON(result)
	if err != nil {
		t.Fatalf("FormatJSON() error: %v", err)
	}

	if !strings.Contains(out, "Great job!") {
		t.Error("JSON output should contain summary")
	}
}

func TestFormatHuman_WithIssues(t *testing.T) {
	result := &reviewer.Result{
		Issues: []reviewer.Issue{
			{
				Title:       "N+1 Query",
				Description: "Loading posts in loop",
				File:        "app/Http/Controllers/PostController.php",
				Line:        22,
				Severity:    "high",
				Suggestion:  "Use eager loading",
			},
			{
				Title:       "Missing Validation",
				Description: "No validation on input",
				File:        "app/Http/Controllers/PostController.php",
				Line:        30,
				Severity:    "medium",
				Suggestion:  "Add Form Request",
			},
		},
		FilesCount: 1,
		Summary:    "Reviewed 1 file(s). Found 2 issue(s): 1 high, 1 medium, 0 low.",
	}

	out := FormatHuman(result)

	// Should contain header
	if !strings.Contains(out, "Laravel AI Code Review Results") {
		t.Error("should contain header")
	}

	// Should contain summary
	if !strings.Contains(out, "Found 2 issue(s)") {
		t.Error("should contain summary")
	}

	// Should contain file name
	if !strings.Contains(out, "PostController.php") {
		t.Error("should contain file name")
	}

	// Should contain issue titles
	if !strings.Contains(out, "N+1 Query") {
		t.Error("should contain issue title 'N+1 Query'")
	}
	if !strings.Contains(out, "Missing Validation") {
		t.Error("should contain issue title 'Missing Validation'")
	}

	// Should contain severity markers
	if !strings.Contains(out, "[HIGH]") {
		t.Error("should contain [HIGH] severity")
	}
	if !strings.Contains(out, "[MEDIUM]") {
		t.Error("should contain [MEDIUM] severity")
	}

	// Should contain line numbers
	if !strings.Contains(out, "Line: 22") {
		t.Error("should contain line number 22")
	}

	// Should contain suggestions
	if !strings.Contains(out, "Use eager loading") {
		t.Error("should contain suggestion")
	}
}

func TestFormatHuman_NoIssues(t *testing.T) {
	result := &reviewer.Result{
		Issues:     []reviewer.Issue{},
		FilesCount: 2,
		Summary:    "Reviewed 2 file(s). No issues found. Great job!",
	}

	out := FormatHuman(result)

	if !strings.Contains(out, "No issues found") {
		t.Error("should contain 'No issues found' message")
	}
	if !strings.Contains(out, "Great job!") {
		t.Error("should contain summary")
	}
}

func TestFormatHuman_EmptyFile(t *testing.T) {
	result := &reviewer.Result{
		Issues: []reviewer.Issue{
			{
				Title:       "Some Issue",
				Description: "desc",
				File:        "",
				Severity:    "low",
			},
		},
		FilesCount: 1,
		Summary:    "Found 1 issue(s)",
	}

	out := FormatHuman(result)

	// Empty file should be grouped as "(unknown)"
	if !strings.Contains(out, "(unknown)") {
		t.Error("empty file should be grouped as '(unknown)'")
	}
}

func TestFormatGitHubAnnotations_WithIssues(t *testing.T) {
	result := &reviewer.Result{
		Issues: []reviewer.Issue{
			{
				Title:       "SQL Injection",
				Description: "Raw query without binding",
				File:        "app/Http/Controllers/UserController.php",
				Line:        53,
				Severity:    "high",
			},
			{
				Title:       "Missing Validation",
				Description: "No input validation",
				File:        "app/Http/Controllers/UserController.php",
				Line:        21,
				Severity:    "medium",
			},
			{
				Title:       "Naming Convention",
				Description: "Method name not camelCase",
				File:        "app/Models/User.php",
				Line:        10,
				Severity:    "low",
			},
		},
	}

	out := FormatGitHubAnnotations(result)

	// High → ::error
	if !strings.Contains(out, "::error file=app/Http/Controllers/UserController.php,line=53::SQL Injection:") {
		t.Errorf("high severity should produce ::error, got:\n%s", out)
	}
	// Medium → ::warning
	if !strings.Contains(out, "::warning file=app/Http/Controllers/UserController.php,line=21::Missing Validation:") {
		t.Errorf("medium severity should produce ::warning, got:\n%s", out)
	}
	// Low → ::notice
	if !strings.Contains(out, "::notice file=app/Models/User.php,line=10::Naming Convention:") {
		t.Errorf("low severity should produce ::notice, got:\n%s", out)
	}
}

func TestFormatGitHubAnnotations_NoIssues(t *testing.T) {
	result := &reviewer.Result{Issues: []reviewer.Issue{}}
	out := FormatGitHubAnnotations(result)
	if out != "" {
		t.Errorf("no issues should produce empty output, got: %q", out)
	}
}

func TestGroupByFile(t *testing.T) {
	issues := []reviewer.Issue{
		{File: "a.php", Title: "issue1"},
		{File: "b.php", Title: "issue2"},
		{File: "a.php", Title: "issue3"},
		{File: "", Title: "issue4"},
	}

	grouped := groupByFile(issues)

	if len(grouped["a.php"]) != 2 {
		t.Errorf("a.php should have 2 issues, got %d", len(grouped["a.php"]))
	}
	if len(grouped["b.php"]) != 1 {
		t.Errorf("b.php should have 1 issue, got %d", len(grouped["b.php"]))
	}
	if len(grouped["(unknown)"]) != 1 {
		t.Errorf("(unknown) should have 1 issue, got %d", len(grouped["(unknown)"]))
	}
}

func TestSeverityIcon(t *testing.T) {
	tests := []struct {
		severity string
		expected string
	}{
		{"high", "🔴"},
		{"HIGH", "🔴"},
		{"medium", "🟡"},
		{"Medium", "🟡"},
		{"low", "🔵"},
		{"LOW", "🔵"},
		{"unknown", "⚪"},
		{"", "⚪"},
	}

	for _, tt := range tests {
		got := severityIcon(tt.severity)
		if got != tt.expected {
			t.Errorf("severityIcon(%q) = %q, want %q", tt.severity, got, tt.expected)
		}
	}
}
