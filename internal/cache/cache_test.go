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
