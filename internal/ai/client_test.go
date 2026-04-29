package ai

import (
	"testing"
)

func TestParseIssuesFromText_DirectJSON(t *testing.T) {
	input := `{
  "issues": [
    {
      "title": "Mass Assignment Vulnerability",
      "description": "Using $request->all() allows mass assignment",
      "file": "app/Http/Controllers/UserController.php",
      "line": 15,
      "severity": "high",
      "suggestion": "Use $request->validated() with a Form Request"
    }
  ]
}`

	resp, err := parseIssuesFromText(input)
	if err != nil {
		t.Fatalf("parseIssuesFromText() error: %v", err)
	}
	if len(resp.Issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(resp.Issues))
	}
	if resp.Issues[0].Title != "Mass Assignment Vulnerability" {
		t.Errorf("unexpected title: %s", resp.Issues[0].Title)
	}
	if resp.Issues[0].Severity != "high" {
		t.Errorf("unexpected severity: %s", resp.Issues[0].Severity)
	}
}

func TestParseIssuesFromText_MarkdownWrapped(t *testing.T) {
	input := "```json\n" + `{
  "issues": [
    {
      "title": "N+1 Query",
      "description": "Loading posts in a loop",
      "file": "app/Http/Controllers/PostController.php",
      "line": 22,
      "severity": "medium",
      "suggestion": "Use eager loading with ->with('comments')"
    }
  ]
}` + "\n```"

	resp, err := parseIssuesFromText(input)
	if err != nil {
		t.Fatalf("parseIssuesFromText() error: %v", err)
	}
	if len(resp.Issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(resp.Issues))
	}
	if resp.Issues[0].Title != "N+1 Query" {
		t.Errorf("unexpected title: %s", resp.Issues[0].Title)
	}
}

func TestParseIssuesFromText_EmptyIssues(t *testing.T) {
	input := `{"issues": []}`

	resp, err := parseIssuesFromText(input)
	if err != nil {
		t.Fatalf("parseIssuesFromText() error: %v", err)
	}
	if len(resp.Issues) != 0 {
		t.Errorf("expected 0 issues, got %d", len(resp.Issues))
	}
}

func TestParseIssuesFromText_WithSurroundingText(t *testing.T) {
	input := `Here is my analysis:
{
  "issues": [
    {
      "title": "Missing Validation",
      "description": "No validation on store method",
      "file": "app/Http/Controllers/UserController.php",
      "line": 20,
      "severity": "high",
      "suggestion": "Add a Form Request class"
    }
  ]
}
That's all I found.`

	resp, err := parseIssuesFromText(input)
	if err != nil {
		t.Fatalf("parseIssuesFromText() error: %v", err)
	}
	if len(resp.Issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(resp.Issues))
	}
}

func TestExtractContent_ChatCompletions(t *testing.T) {
	body := []byte(`{
  "choices": [
    {
      "message": {
        "content": "{\"issues\": [{\"title\": \"Test\", \"description\": \"desc\", \"file\": \"app/test.php\", \"line\": 1, \"severity\": \"low\", \"suggestion\": \"fix it\"}]}"
      }
    }
  ]
}`)

	resp, err := extractContent(body)
	if err != nil {
		t.Fatalf("extractContent() error: %v", err)
	}
	if len(resp.Issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(resp.Issues))
	}
}

func TestExtractContent_ResponsesAPI(t *testing.T) {
	body := []byte(`{
  "output": [
    {
      "content": [
        {
          "text": "{\"issues\": [{\"title\": \"Test\", \"description\": \"desc\", \"file\": \"app/test.php\", \"line\": 1, \"severity\": \"low\", \"suggestion\": \"fix it\"}]}"
        }
      ]
    }
  ]
}`)

	resp, err := extractContent(body)
	if err != nil {
		t.Fatalf("extractContent() error: %v", err)
	}
	if len(resp.Issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(resp.Issues))
	}
}

func TestBuildDiffPrompt(t *testing.T) {
	prompt := BuildDiffPrompt()
	if prompt == "" {
		t.Error("expected non-empty prompt")
	}
	keywords := []string{"N+1", "mass assignment", "severity", "JSON"}
	for _, kw := range keywords {
		if !containsStr(prompt, kw) {
			t.Errorf("diff prompt should contain %q", kw)
		}
	}
}

func TestBuildScanPrompt(t *testing.T) {
	prompt := BuildScanPrompt()
	if prompt == "" {
		t.Error("expected non-empty prompt")
	}
	keywords := []string{"audits Laravel code", "entire file", "severity", "JSON"}
	for _, kw := range keywords {
		if !containsStr(prompt, kw) {
			t.Errorf("scan prompt should contain %q", kw)
		}
	}
}

func TestParseMarkdownFallback(t *testing.T) {
	markdown := `# Code Review

## File: ` + "`app/Console/Commands/SyncIdCardWithNpwpDocument.php`" + `

### Bug: Undefined Variable

**Severity: 🔴 Critical**

**Lines affected:** 60, 65, 70

The variable $businessAccount is used but was never defined.

---

## File: ` + "`app/Console/Commands/GeneratePOSAccountCommand.php`" + `

### Unbounded Query

**Severity: 🟡 Medium**

This fetches every business account with no filter.`

	issues := parseMarkdownFallback(markdown)

	if len(issues) != 2 {
		t.Fatalf("expected 2 issues from markdown, got %d", len(issues))
	}

	if issues[0].File != "app/Console/Commands/SyncIdCardWithNpwpDocument.php" {
		t.Errorf("issue 0 file = %q", issues[0].File)
	}
	if issues[0].Severity != "high" {
		t.Errorf("issue 0 severity = %q, want 'high'", issues[0].Severity)
	}
	if issues[0].Line != 60 {
		t.Errorf("issue 0 line = %d, want 60", issues[0].Line)
	}

	if issues[1].File != "app/Console/Commands/GeneratePOSAccountCommand.php" {
		t.Errorf("issue 1 file = %q", issues[1].File)
	}
	if issues[1].Severity != "medium" {
		t.Errorf("issue 1 severity = %q, want 'medium'", issues[1].Severity)
	}
}

func containsStr(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && contains(s, substr)
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
