package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Provider.Model != "gpt-4.1" {
		t.Errorf("expected default model 'gpt-4.1', got '%s'", cfg.Provider.Model)
	}
	if cfg.Provider.Endpoint != "chat" {
		t.Errorf("expected default endpoint 'chat', got '%s'", cfg.Provider.Endpoint)
	}
	if cfg.Review.MaxIssues != 5 {
		t.Errorf("expected default max_issues 5, got %d", cfg.Review.MaxIssues)
	}
	if cfg.Review.SeverityThreshold != "medium" {
		t.Errorf("expected default severity_threshold 'medium', got '%s'", cfg.Review.SeverityThreshold)
	}
	if len(cfg.Filters.Include) != 3 {
		t.Errorf("expected 3 default include patterns, got %d", len(cfg.Filters.Include))
	}
	if len(cfg.Filters.Exclude) != 2 {
		t.Errorf("expected 2 default exclude patterns, got %d", len(cfg.Filters.Exclude))
	}
}

func TestExpandEnvVars(t *testing.T) {
	os.Setenv("TEST_LIMBO_KEY", "my-secret-key")
	defer os.Unsetenv("TEST_LIMBO_KEY")

	tests := []struct {
		input    string
		expected string
	}{
		{"${TEST_LIMBO_KEY}", "my-secret-key"},
		{"prefix-${TEST_LIMBO_KEY}-suffix", "prefix-my-secret-key-suffix"},
		{"no-vars-here", "no-vars-here"},
		{"${NONEXISTENT_VAR_12345}", ""},
		{"", ""},
	}

	for _, tt := range tests {
		got := expandEnvVars(tt.input)
		if got != tt.expected {
			t.Errorf("expandEnvVars(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestLoad_ValidConfig(t *testing.T) {
	dir := t.TempDir()
	configContent := `provider:
  base_url: http://localhost:1430
  api_key: test-key-123
  model: gpt-4o-mini
  endpoint: responses

review:
  max_issues: 10
  severity_threshold: low
  custom_prompt: "Focus on security"

filters:
  include:
    - "app/**"
  exclude:
    - "vendor/**"
`
	err := os.WriteFile(filepath.Join(dir, ".larasense-limbo.yml"), []byte(configContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Change to temp dir so viper finds the config
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Provider.BaseURL != "http://localhost:1430" {
		t.Errorf("base_url = %q, want 'http://localhost:1430'", cfg.Provider.BaseURL)
	}
	if cfg.Provider.APIKey != "test-key-123" {
		t.Errorf("api_key = %q, want 'test-key-123'", cfg.Provider.APIKey)
	}
	if cfg.Provider.Model != "gpt-4o-mini" {
		t.Errorf("model = %q, want 'gpt-4o-mini'", cfg.Provider.Model)
	}
	if cfg.Provider.Endpoint != "responses" {
		t.Errorf("endpoint = %q, want 'responses'", cfg.Provider.Endpoint)
	}
	if cfg.Review.MaxIssues != 10 {
		t.Errorf("max_issues = %d, want 10", cfg.Review.MaxIssues)
	}
	if cfg.Review.SeverityThreshold != "low" {
		t.Errorf("severity_threshold = %q, want 'low'", cfg.Review.SeverityThreshold)
	}
	if cfg.Review.CustomPrompt != "Focus on security" {
		t.Errorf("custom_prompt = %q, want 'Focus on security'", cfg.Review.CustomPrompt)
	}
	if len(cfg.Filters.Include) != 1 || cfg.Filters.Include[0] != "app/**" {
		t.Errorf("include = %v, want ['app/**']", cfg.Filters.Include)
	}
	if len(cfg.Filters.Exclude) != 1 || cfg.Filters.Exclude[0] != "vendor/**" {
		t.Errorf("exclude = %v, want ['vendor/**']", cfg.Filters.Exclude)
	}
}

func TestLoad_MissingBaseURL(t *testing.T) {
	dir := t.TempDir()
	configContent := `provider:
  api_key: test-key
`
	os.WriteFile(filepath.Join(dir, ".larasense-limbo.yml"), []byte(configContent), 0644)

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing base_url, got nil")
	}
	if !contains(err.Error(), "base_url") {
		t.Errorf("error should mention 'base_url', got: %s", err.Error())
	}
}

func TestLoad_MissingAPIKey(t *testing.T) {
	dir := t.TempDir()
	configContent := `provider:
  base_url: http://localhost:1430
`
	os.WriteFile(filepath.Join(dir, ".larasense-limbo.yml"), []byte(configContent), 0644)

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing api_key, got nil")
	}
	if !contains(err.Error(), "api_key") {
		t.Errorf("error should mention 'api_key', got: %s", err.Error())
	}
}

func TestLoad_EnvVarExpansion(t *testing.T) {
	dir := t.TempDir()
	configContent := `provider:
  base_url: http://localhost:1430
  api_key: ${TEST_LIMBO_API_KEY}
`
	os.WriteFile(filepath.Join(dir, ".larasense-limbo.yml"), []byte(configContent), 0644)

	os.Setenv("TEST_LIMBO_API_KEY", "expanded-key-value")
	defer os.Unsetenv("TEST_LIMBO_API_KEY")

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Provider.APIKey != "expanded-key-value" {
		t.Errorf("api_key = %q, want 'expanded-key-value'", cfg.Provider.APIKey)
	}
}

func TestLoad_DefaultsApplied(t *testing.T) {
	dir := t.TempDir()
	configContent := `provider:
  base_url: http://localhost:1430
  api_key: test-key
`
	os.WriteFile(filepath.Join(dir, ".larasense-limbo.yml"), []byte(configContent), 0644)

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// These should be defaults since not specified in config
	if cfg.Provider.Model != "gpt-4.1" {
		t.Errorf("model should default to 'gpt-4.1', got '%s'", cfg.Provider.Model)
	}
	if cfg.Provider.Endpoint != "chat" {
		t.Errorf("endpoint should default to 'chat', got '%s'", cfg.Provider.Endpoint)
	}
	if cfg.Review.MaxIssues != 5 {
		t.Errorf("max_issues should default to 5, got %d", cfg.Review.MaxIssues)
	}
	if cfg.Review.SeverityThreshold != "medium" {
		t.Errorf("severity_threshold should default to 'medium', got '%s'", cfg.Review.SeverityThreshold)
	}
}

func TestLoad_ProviderPreset_OpenAI(t *testing.T) {
	dir := t.TempDir()
	configContent := `provider:
  name: openai
  api_key: test-key
`
	os.WriteFile(filepath.Join(dir, ".larasense-limbo.yml"), []byte(configContent), 0644)

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Provider.BaseURL != "https://api.openai.com" {
		t.Errorf("base_url = %q, want openai preset", cfg.Provider.BaseURL)
	}
	if cfg.Provider.Model != "gpt-4.1" {
		t.Errorf("model = %q, want 'gpt-4.1'", cfg.Provider.Model)
	}
}

func TestLoad_ProviderPreset_Ollama(t *testing.T) {
	dir := t.TempDir()
	configContent := `provider:
  name: ollama
`
	os.WriteFile(filepath.Join(dir, ".larasense-limbo.yml"), []byte(configContent), 0644)

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v (ollama should not require api_key)", err)
	}
	if cfg.Provider.BaseURL != "http://localhost:11434" {
		t.Errorf("base_url = %q, want ollama preset", cfg.Provider.BaseURL)
	}
	if cfg.Provider.Model != "llama3.1" {
		t.Errorf("model = %q, want 'llama3.1'", cfg.Provider.Model)
	}
}

func TestLoad_ProviderPreset_OverrideModel(t *testing.T) {
	dir := t.TempDir()
	configContent := `provider:
  name: openai
  api_key: test-key
  model: gpt-4o
`
	os.WriteFile(filepath.Join(dir, ".larasense-limbo.yml"), []byte(configContent), 0644)

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Provider.Model != "gpt-4o" {
		t.Errorf("model = %q, want user override 'gpt-4o'", cfg.Provider.Model)
	}
}

func TestLoad_ProviderPreset_Unknown(t *testing.T) {
	dir := t.TempDir()
	configContent := `provider:
  name: nonexistent
  api_key: test-key
`
	os.WriteFile(filepath.Join(dir, ".larasense-limbo.yml"), []byte(configContent), 0644)

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for unknown provider name")
	}
	if !contains(err.Error(), "unknown provider") {
		t.Errorf("error should mention 'unknown provider', got: %s", err.Error())
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchStr(s, substr)
}

func searchStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
