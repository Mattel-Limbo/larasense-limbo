package reviewer

import (
	"testing"

	"github.com/Mattel-Limbo/larasense-limbo/internal/ai"
	"github.com/Mattel-Limbo/larasense-limbo/internal/config"
)

func TestMeetsThreshold(t *testing.T) {
	tests := []struct {
		severity  string
		threshold string
		expected  bool
	}{
		{"high", "high", true},
		{"high", "medium", true},
		{"high", "low", true},
		{"medium", "high", false},
		{"medium", "medium", true},
		{"medium", "low", true},
		{"low", "high", false},
		{"low", "medium", false},
		{"low", "low", true},
		{"HIGH", "medium", true},
		{"Medium", "LOW", true},
		{"unknown", "medium", false},
		{"high", "unknown", true},
		{"unknown", "unknown", true},
	}

	for _, tt := range tests {
		got := meetsThreshold(tt.severity, tt.threshold)
		if got != tt.expected {
			t.Errorf("meetsThreshold(%q, %q) = %v, want %v", tt.severity, tt.threshold, got, tt.expected)
		}
	}
}

func TestFilterIssues_SeverityThreshold(t *testing.T) {
	issues := []ai.Issue{
		{Title: "Critical SQL Injection", Severity: "high"},
		{Title: "Missing Eager Loading", Severity: "medium"},
		{Title: "Naming Convention", Severity: "low"},
		{Title: "Mass Assignment", Severity: "high"},
	}

	cfg := &config.Config{
		Review: config.ReviewConfig{
			SeverityThreshold: "medium",
			MaxIssues:         0,
		},
	}

	filtered := filterIssues(issues, cfg)

	if len(filtered) != 3 {
		t.Fatalf("expected 3 issues (high+medium), got %d", len(filtered))
	}

	for _, issue := range filtered {
		if issue.Severity == "low" {
			t.Errorf("low severity issue %q should have been filtered out", issue.Title)
		}
	}
}

func TestFilterIssues_HighOnly(t *testing.T) {
	issues := []ai.Issue{
		{Title: "SQL Injection", Severity: "high"},
		{Title: "Missing Eager Loading", Severity: "medium"},
		{Title: "Naming Convention", Severity: "low"},
	}

	cfg := &config.Config{
		Review: config.ReviewConfig{
			SeverityThreshold: "high",
			MaxIssues:         0,
		},
	}

	filtered := filterIssues(issues, cfg)

	if len(filtered) != 1 {
		t.Fatalf("expected 1 high issue, got %d", len(filtered))
	}
	if filtered[0].Title != "SQL Injection" {
		t.Errorf("expected 'SQL Injection', got %q", filtered[0].Title)
	}
}

func TestFilterIssues_MaxIssuesLimit(t *testing.T) {
	issues := []ai.Issue{
		{Title: "Issue 1", Severity: "high"},
		{Title: "Issue 2", Severity: "high"},
		{Title: "Issue 3", Severity: "high"},
		{Title: "Issue 4", Severity: "medium"},
		{Title: "Issue 5", Severity: "medium"},
	}

	cfg := &config.Config{
		Review: config.ReviewConfig{
			SeverityThreshold: "low",
			MaxIssues:         3,
		},
	}

	filtered := filterIssues(issues, cfg)

	if len(filtered) != 3 {
		t.Fatalf("expected 3 issues (max_issues limit), got %d", len(filtered))
	}
}

func TestFilterIssues_MaxIssuesZeroMeansUnlimited(t *testing.T) {
	issues := []ai.Issue{
		{Title: "Issue 1", Severity: "high"},
		{Title: "Issue 2", Severity: "high"},
		{Title: "Issue 3", Severity: "medium"},
	}

	cfg := &config.Config{
		Review: config.ReviewConfig{
			SeverityThreshold: "low",
			MaxIssues:         0,
		},
	}

	filtered := filterIssues(issues, cfg)

	if len(filtered) != 3 {
		t.Fatalf("expected all 3 issues when max_issues=0, got %d", len(filtered))
	}
}

func TestFilterIssues_EmptyInput(t *testing.T) {
	cfg := &config.Config{
		Review: config.ReviewConfig{
			SeverityThreshold: "low",
			MaxIssues:         5,
		},
	}

	filtered := filterIssues(nil, cfg)

	if len(filtered) != 0 {
		t.Errorf("expected 0 issues for nil input, got %d", len(filtered))
	}
}

func TestFilterIssues_ThresholdAndMaxCombined(t *testing.T) {
	issues := []ai.Issue{
		{Title: "High 1", Severity: "high"},
		{Title: "High 2", Severity: "high"},
		{Title: "High 3", Severity: "high"},
		{Title: "Medium 1", Severity: "medium"},
		{Title: "Low 1", Severity: "low"},
	}

	cfg := &config.Config{
		Review: config.ReviewConfig{
			SeverityThreshold: "medium",
			MaxIssues:         2,
		},
	}

	filtered := filterIssues(issues, cfg)

	if len(filtered) != 2 {
		t.Fatalf("expected 2 issues (threshold filters low, max caps at 2), got %d", len(filtered))
	}
}

func TestBuildSummary_WithIssues(t *testing.T) {
	issues := []Issue{
		{Severity: "high"},
		{Severity: "medium"},
		{Severity: "medium"},
		{Severity: "low"},
	}

	summary := buildSummary(issues, 5, "diff")

	expected := "Reviewed 5 file(s). Found 4 issue(s): 1 high, 2 medium, 1 low."
	if summary != expected {
		t.Errorf("buildSummary() = %q, want %q", summary, expected)
	}
}

func TestBuildSummary_NoIssues(t *testing.T) {
	summary := buildSummary(nil, 3, "diff")

	expected := "Reviewed 3 file(s). No issues found. Great job!"
	if summary != expected {
		t.Errorf("buildSummary() = %q, want %q", summary, expected)
	}
}

func TestBuildSummary_AllHigh(t *testing.T) {
	issues := []Issue{
		{Severity: "high"},
		{Severity: "high"},
	}

	summary := buildSummary(issues, 1, "diff")

	expected := "Reviewed 1 file(s). Found 2 issue(s): 2 high, 0 medium, 0 low."
	if summary != expected {
		t.Errorf("buildSummary() = %q, want %q", summary, expected)
	}
}

func TestBuildSummary_UnknownSeverityCountsAsLow(t *testing.T) {
	issues := []Issue{
		{Severity: "critical"},
	}

	summary := buildSummary(issues, 1, "diff")

	expected := "Reviewed 1 file(s). Found 1 issue(s): 0 high, 0 medium, 1 low."
	if summary != expected {
		t.Errorf("buildSummary() = %q, want %q", summary, expected)
	}
}

func TestBuildSummary_ScanMode(t *testing.T) {
	issues := []Issue{
		{Severity: "high"},
	}

	summary := buildSummary(issues, 10, "scan")

	expected := "Scanned 10 file(s). Found 1 issue(s): 1 high, 0 medium, 0 low."
	if summary != expected {
		t.Errorf("buildSummary() = %q, want %q", summary, expected)
	}
}

func TestBuildSummary_ScanNoIssues(t *testing.T) {
	summary := buildSummary(nil, 5, "scan")

	expected := "Scanned 5 file(s). No issues found. Great job!"
	if summary != expected {
		t.Errorf("buildSummary() = %q, want %q", summary, expected)
	}
}

func TestNew(t *testing.T) {
	cfg := config.DefaultConfig()

	r := New(cfg, true, false, false)

	if r.cfg != cfg {
		t.Error("expected cfg to be stored")
	}
	if !r.verbose {
		t.Error("expected verbose=true")
	}
	if r.noCache {
		t.Error("expected noCache=false")
	}
	if r.fixMode {
		t.Error("expected fixMode=false")
	}
}

func TestNew_AllFlags(t *testing.T) {
	cfg := config.DefaultConfig()

	r := New(cfg, false, true, true)

	if r.verbose {
		t.Error("expected verbose=false")
	}
	if !r.noCache {
		t.Error("expected noCache=true")
	}
	if !r.fixMode {
		t.Error("expected fixMode=true")
	}
}
