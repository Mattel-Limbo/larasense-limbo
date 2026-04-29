package scanner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Mattel-Limbo/larasense-limbo/internal/config"
)

func setupTestDir(t *testing.T) (string, func()) {
	t.Helper()
	dir := t.TempDir()

	dirs := []string{
		"app/Http/Controllers",
		"app/Models",
		"routes",
		"resources/views/users",
		"tests/Feature",
		"vendor/laravel/framework",
		"node_modules/some-package",
		"config",
		"storage/logs",
	}
	for _, d := range dirs {
		os.MkdirAll(filepath.Join(dir, d), 0755)
	}

	files := map[string]string{
		"app/Http/Controllers/UserController.php": "<?php\nclass UserController {}\n",
		"app/Models/User.php":                     "<?php\nclass User {}\n",
		"routes/web.php":                          "<?php\nRoute::get('/', fn() => 'hello');\n",
		"resources/views/users/index.blade.php":   "<div>{{ $user->name }}</div>\n",
		"tests/Feature/UserTest.php":              "<?php\nclass UserTest {}\n",
		"vendor/laravel/framework/Auth.php":       "<?php\nclass Auth {}\n",
		"config/app.php":                          "<?php\nreturn ['name' => env('APP_NAME')];\n",
		"README.md":                               "# Project\n",
		"package.json":                            "{}\n",
	}
	for path, content := range files {
		os.WriteFile(filepath.Join(dir, path), []byte(content), 0644)
	}

	origDir, _ := os.Getwd()
	os.Chdir(dir)

	return dir, func() { os.Chdir(origDir) }
}

func TestScan_FindsLaravelFiles(t *testing.T) {
	dir, cleanup := setupTestDir(t)
	defer cleanup()
	cfg := config.DefaultConfig()

	s := New(cfg, dir)
	files, err := s.Scan()
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	if len(files) == 0 {
		t.Fatal("expected to find Laravel files")
	}

	paths := make(map[string]bool)
	for _, f := range files {
		paths[f.Path] = true
	}

	if !paths["app/Http/Controllers/UserController.php"] {
		t.Error("should find UserController.php")
	}
	if !paths["app/Models/User.php"] {
		t.Error("should find User.php")
	}
	if !paths["routes/web.php"] {
		t.Error("should find web.php")
	}
	if !paths["resources/views/users/index.blade.php"] {
		t.Error("should find blade template")
	}
}

func TestScan_ExcludesNonLaravelFiles(t *testing.T) {
	dir, cleanup := setupTestDir(t)
	defer cleanup()
	cfg := config.DefaultConfig()

	s := New(cfg, dir)
	files, err := s.Scan()
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	for _, f := range files {
		if f.Path == "README.md" || f.Path == "package.json" {
			t.Errorf("should not include non-PHP file: %s", f.Path)
		}
		if f.Path == "tests/Feature/UserTest.php" {
			t.Errorf("should not include test file (excluded by default config): %s", f.Path)
		}
	}
}

func TestScan_SkipsVendorAndNodeModules(t *testing.T) {
	dir, cleanup := setupTestDir(t)
	defer cleanup()
	cfg := config.DefaultConfig()

	s := New(cfg, dir)
	files, err := s.Scan()
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	for _, f := range files {
		if filepath.HasPrefix(f.Path, "vendor/") {
			t.Errorf("should not include vendor file: %s", f.Path)
		}
		if filepath.HasPrefix(f.Path, "node_modules/") {
			t.Errorf("should not include node_modules file: %s", f.Path)
		}
	}
}

func TestScan_ClassifiesFiles(t *testing.T) {
	dir, cleanup := setupTestDir(t)
	defer cleanup()
	cfg := config.DefaultConfig()

	s := New(cfg, dir)
	files, err := s.Scan()
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	typeMap := make(map[string]string)
	for _, f := range files {
		typeMap[f.Path] = f.Type
	}

	if typeMap["app/Http/Controllers/UserController.php"] != "controller" {
		t.Errorf("UserController should be classified as 'controller', got %q", typeMap["app/Http/Controllers/UserController.php"])
	}
	if typeMap["app/Models/User.php"] != "model" {
		t.Errorf("User should be classified as 'model', got %q", typeMap["app/Models/User.php"])
	}
	if typeMap["routes/web.php"] != "route" {
		t.Errorf("web.php should be classified as 'route', got %q", typeMap["routes/web.php"])
	}
}

func TestScan_ReadsFileContent(t *testing.T) {
	dir, cleanup := setupTestDir(t)
	defer cleanup()
	cfg := config.DefaultConfig()

	s := New(cfg, dir)
	files, err := s.Scan()
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	for _, f := range files {
		if f.DiffText == "" {
			t.Errorf("file %s should have content in DiffText", f.Path)
		}
	}
}

func TestScan_SubdirectoryPath(t *testing.T) {
	dir, cleanup := setupTestDir(t)
	defer cleanup()
	cfg := config.DefaultConfig()

	subDir := filepath.Join(dir, "app", "Http", "Controllers")
	s := New(cfg, subDir)
	files, err := s.Scan()
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	if len(files) == 0 {
		t.Fatal("expected to find files when scanning subdirectory")
	}

	for _, f := range files {
		if f.Path != "app/Http/Controllers/UserController.php" {
			t.Errorf("expected path relative to project root, got %q", f.Path)
		}
		if f.Type != "controller" {
			t.Errorf("expected type 'controller', got %q", f.Type)
		}
	}
}

func TestScan_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()
	cfg := config.DefaultConfig()

	s := New(cfg, dir)
	files, err := s.Scan()
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	if len(files) != 0 {
		t.Errorf("expected 0 files in empty dir, got %d", len(files))
	}
}
