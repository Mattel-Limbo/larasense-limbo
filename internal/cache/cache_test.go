package cache

import (
	"os"
	"testing"
)

func TestCache_HasChanged_NewFile(t *testing.T) {
	c := &Cache{Files: make(map[string]Entry)}

	if !c.HasChanged("app/test.php", "some diff") {
		t.Error("new file should be marked as changed")
	}
}

func TestCache_HasChanged_SameContent(t *testing.T) {
	c := &Cache{Files: make(map[string]Entry)}

	c.Update("app/test.php", "some diff")

	if c.HasChanged("app/test.php", "some diff") {
		t.Error("same content should not be marked as changed")
	}
}

func TestCache_HasChanged_DifferentContent(t *testing.T) {
	c := &Cache{Files: make(map[string]Entry)}

	c.Update("app/test.php", "old diff")

	if !c.HasChanged("app/test.php", "new diff") {
		t.Error("different content should be marked as changed")
	}
}

func TestCache_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	// Create and save
	c := &Cache{Files: make(map[string]Entry)}
	c.Update("app/test.php", "diff content")
	err := c.Save()
	if err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Load and verify
	loaded := Load()
	if loaded.HasChanged("app/test.php", "diff content") {
		t.Error("loaded cache should recognize unchanged file")
	}
	if !loaded.HasChanged("app/test.php", "different content") {
		t.Error("loaded cache should detect changed file")
	}
}

func TestCache_LoadMissing(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	c := Load()
	if c == nil {
		t.Fatal("Load() should return empty cache, not nil")
	}
	if len(c.Files) != 0 {
		t.Errorf("empty cache should have 0 files, got %d", len(c.Files))
	}
}

func TestCache_NoDirtyNoSave(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	c := &Cache{Files: make(map[string]Entry)}
	// Don't update anything
	err := c.Save()
	if err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// File should not exist since nothing was dirty
	if _, err := os.Stat(cacheFile); err == nil {
		t.Error("cache file should not be created when nothing is dirty")
	}
}

func TestCache_UpdateWithIssues(t *testing.T) {
	c := &Cache{Files: make(map[string]Entry)}

	issues := []CachedIssue{
		{Title: "N+1 Query", Description: "Loading in loop", File: "app/test.php", Line: 10, Severity: "high", Suggestion: "Use eager loading"},
	}
	c.UpdateWithIssues("app/test.php", "some diff", issues)

	cached, hit := c.GetCachedIssues("app/test.php", "some diff")
	if !hit {
		t.Fatal("expected cache hit for same content")
	}
	if len(cached) != 1 {
		t.Fatalf("expected 1 cached issue, got %d", len(cached))
	}
	if cached[0].Title != "N+1 Query" {
		t.Errorf("unexpected cached title: %s", cached[0].Title)
	}
}

func TestCache_GetCachedIssues_Miss(t *testing.T) {
	c := &Cache{Files: make(map[string]Entry)}

	issues := []CachedIssue{
		{Title: "Test", File: "app/test.php", Severity: "low"},
	}
	c.UpdateWithIssues("app/test.php", "old diff", issues)

	// Different content should miss
	cached, hit := c.GetCachedIssues("app/test.php", "new diff")
	if hit {
		t.Error("expected cache miss for different content")
	}
	if cached != nil {
		t.Error("expected nil issues on cache miss")
	}
}

func TestCache_GetCachedIssues_EmptyIssues(t *testing.T) {
	c := &Cache{Files: make(map[string]Entry)}

	// File reviewed with no issues found
	c.UpdateWithIssues("app/clean.php", "clean diff", []CachedIssue{})

	cached, hit := c.GetCachedIssues("app/clean.php", "clean diff")
	if !hit {
		t.Fatal("expected cache hit for file with empty issues")
	}
	if len(cached) != 0 {
		t.Errorf("expected 0 cached issues, got %d", len(cached))
	}
}

func TestCache_GetCachedIssues_NotExists(t *testing.T) {
	c := &Cache{Files: make(map[string]Entry)}

	cached, hit := c.GetCachedIssues("nonexistent.php", "diff")
	if hit {
		t.Error("expected cache miss for nonexistent file")
	}
	if cached != nil {
		t.Error("expected nil issues for nonexistent file")
	}
}

func TestCache_SaveAndLoadWithIssues(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	// Create and save with issues
	c := &Cache{Files: make(map[string]Entry)}
	issues := []CachedIssue{
		{Title: "SQL Injection", File: "app/test.php", Line: 5, Severity: "high", Suggestion: "Use parameterized queries"},
	}
	c.UpdateWithIssues("app/test.php", "diff content", issues)
	err := c.Save()
	if err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Load and verify issues are persisted
	loaded := Load()
	cached, hit := loaded.GetCachedIssues("app/test.php", "diff content")
	if !hit {
		t.Fatal("expected cache hit after load")
	}
	if len(cached) != 1 {
		t.Fatalf("expected 1 cached issue after load, got %d", len(cached))
	}
	if cached[0].Title != "SQL Injection" {
		t.Errorf("unexpected cached title after load: %s", cached[0].Title)
	}
}
