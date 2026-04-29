package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestRepo(t *testing.T) (repoDir string, cleanup func()) {
	t.Helper()

	dir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	run(t, "git", "init")
	run(t, "git", "config", "user.email", "test@test.com")
	run(t, "git", "config", "user.name", "Test")

	return dir, func() { os.Chdir(origDir) }
}

func run(t *testing.T, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v failed: %v\n%s", name, args, err, out)
	}
	return string(out)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestGetDiff_BasicDiff(t *testing.T) {
	_, cleanup := setupTestRepo(t)
	defer cleanup()

	writeFile(t, "app/Http/Controllers/UserController.php", `<?php
class UserController extends Controller
{
    public function index()
    {
        $users = User::all();
        return view('users.index', compact('users'));
    }
}
`)
	run(t, "git", "add", ".")
	run(t, "git", "commit", "-m", "initial commit")

	run(t, "git", "checkout", "-b", "feature/test")
	writeFile(t, "app/Http/Controllers/UserController.php", `<?php
class UserController extends Controller
{
    public function index()
    {
        $users = User::with('posts')->paginate(15);
        return view('users.index', compact('users'));
    }
}
`)
	run(t, "git", "add", ".")
	run(t, "git", "commit", "-m", "add eager loading")

	diff, err := GetDiff("master", "feature/test")
	if err != nil {
		t.Fatalf("GetDiff() error: %v", err)
	}

	if diff == "" {
		t.Fatal("expected non-empty diff")
	}
	if !strings.Contains(diff, "UserController.php") {
		t.Error("diff should contain UserController.php")
	}
	if !strings.Contains(diff, "paginate") {
		t.Error("diff should contain the changed line with 'paginate'")
	}
}

func TestGetDiff_NoDifference(t *testing.T) {
	_, cleanup := setupTestRepo(t)
	defer cleanup()

	writeFile(t, "file.txt", "hello")
	run(t, "git", "add", ".")
	run(t, "git", "commit", "-m", "initial")

	diff, err := GetDiff("HEAD", "HEAD")
	if err != nil {
		t.Fatalf("GetDiff() error: %v", err)
	}
	if strings.TrimSpace(diff) != "" {
		t.Errorf("expected empty diff for same ref, got: %q", diff)
	}
}

func TestGetDiff_InvalidRef(t *testing.T) {
	_, cleanup := setupTestRepo(t)
	defer cleanup()

	writeFile(t, "file.txt", "hello")
	run(t, "git", "add", ".")
	run(t, "git", "commit", "-m", "initial")

	_, err := GetDiff("nonexistent-branch", "HEAD")
	if err == nil {
		t.Fatal("expected error for invalid ref, got nil")
	}
}

func TestGetFileContent_ExistingFile(t *testing.T) {
	_, cleanup := setupTestRepo(t)
	defer cleanup()

	expected := "<?php\necho 'hello';\n"
	writeFile(t, "app/test.php", expected)
	run(t, "git", "add", ".")
	run(t, "git", "commit", "-m", "add test file")

	content, err := GetFileContent("HEAD", "app/test.php")
	if err != nil {
		t.Fatalf("GetFileContent() error: %v", err)
	}
	if content != expected {
		t.Errorf("GetFileContent() = %q, want %q", content, expected)
	}
}

func TestGetFileContent_EmptyRefDefaultsToHEAD(t *testing.T) {
	_, cleanup := setupTestRepo(t)
	defer cleanup()

	expected := "content at HEAD\n"
	writeFile(t, "file.txt", expected)
	run(t, "git", "add", ".")
	run(t, "git", "commit", "-m", "initial")

	content, err := GetFileContent("", "file.txt")
	if err != nil {
		t.Fatalf("GetFileContent() error: %v", err)
	}
	if content != expected {
		t.Errorf("GetFileContent() = %q, want %q", content, expected)
	}
}

func TestGetFileContent_NonexistentFile(t *testing.T) {
	_, cleanup := setupTestRepo(t)
	defer cleanup()

	writeFile(t, "file.txt", "hello")
	run(t, "git", "add", ".")
	run(t, "git", "commit", "-m", "initial")

	_, err := GetFileContent("HEAD", "nonexistent.php")
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
}

func TestGetFileContent_InvalidRef(t *testing.T) {
	_, cleanup := setupTestRepo(t)
	defer cleanup()

	writeFile(t, "file.txt", "hello")
	run(t, "git", "add", ".")
	run(t, "git", "commit", "-m", "initial")

	_, err := GetFileContent("nonexistent-ref", "file.txt")
	if err == nil {
		t.Fatal("expected error for invalid ref, got nil")
	}
}

func TestGetSurroundingLines_MiddleOfFile(t *testing.T) {
	_, cleanup := setupTestRepo(t)
	defer cleanup()

	var lines []string
	for i := 1; i <= 20; i++ {
		lines = append(lines, "line "+string(rune('0'+i/10))+string(rune('0'+i%10)))
	}
	writeFile(t, "file.php", strings.Join(lines, "\n")+"\n")
	run(t, "git", "add", ".")
	run(t, "git", "commit", "-m", "add file")

	result, err := GetSurroundingLines("HEAD", "file.php", 10, 3)
	if err != nil {
		t.Fatalf("GetSurroundingLines() error: %v", err)
	}

	if result == "" {
		t.Fatal("expected non-empty result")
	}

	if !strings.Contains(result, "7:") {
		t.Error("result should contain line 7")
	}
	if !strings.Contains(result, "10:") {
		t.Error("result should contain target line 10")
	}
	if !strings.Contains(result, "13:") {
		t.Error("result should contain line 13")
	}
}

func TestGetSurroundingLines_StartOfFile(t *testing.T) {
	_, cleanup := setupTestRepo(t)
	defer cleanup()

	var lines []string
	for i := 1; i <= 10; i++ {
		lines = append(lines, "line "+string(rune('0'+i/10))+string(rune('0'+i%10)))
	}
	writeFile(t, "file.php", strings.Join(lines, "\n")+"\n")
	run(t, "git", "add", ".")
	run(t, "git", "commit", "-m", "add file")

	result, err := GetSurroundingLines("HEAD", "file.php", 1, 5)
	if err != nil {
		t.Fatalf("GetSurroundingLines() error: %v", err)
	}

	if !strings.Contains(result, "1:") {
		t.Error("result should contain line 1")
	}
}

func TestGetSurroundingLines_EndOfFile(t *testing.T) {
	_, cleanup := setupTestRepo(t)
	defer cleanup()

	var lines []string
	for i := 1; i <= 10; i++ {
		lines = append(lines, "line "+string(rune('0'+i/10))+string(rune('0'+i%10)))
	}
	writeFile(t, "file.php", strings.Join(lines, "\n")+"\n")
	run(t, "git", "add", ".")
	run(t, "git", "commit", "-m", "add file")

	result, err := GetSurroundingLines("HEAD", "file.php", 10, 5)
	if err != nil {
		t.Fatalf("GetSurroundingLines() error: %v", err)
	}

	if !strings.Contains(result, "10:") {
		t.Error("result should contain line 10")
	}
}

func TestGetSurroundingLines_NonexistentFile(t *testing.T) {
	_, cleanup := setupTestRepo(t)
	defer cleanup()

	writeFile(t, "file.txt", "hello")
	run(t, "git", "add", ".")
	run(t, "git", "commit", "-m", "initial")

	_, err := GetSurroundingLines("HEAD", "nonexistent.php", 5, 3)
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
}

func TestGetDiff_MultipleFiles(t *testing.T) {
	_, cleanup := setupTestRepo(t)
	defer cleanup()

	writeFile(t, "app/Models/User.php", "<?php\nclass User {}\n")
	writeFile(t, "routes/web.php", "<?php\nRoute::get('/', fn() => 'hello');\n")
	run(t, "git", "add", ".")
	run(t, "git", "commit", "-m", "initial")

	run(t, "git", "checkout", "-b", "feature/multi")
	writeFile(t, "app/Models/User.php", "<?php\nclass User {\n    protected $guarded = [];\n}\n")
	writeFile(t, "routes/web.php", "<?php\nRoute::get('/', fn() => 'hello');\nRoute::get('/users', fn() => 'users');\n")
	run(t, "git", "add", ".")
	run(t, "git", "commit", "-m", "update both files")

	diff, err := GetDiff("master", "feature/multi")
	if err != nil {
		t.Fatalf("GetDiff() error: %v", err)
	}

	if !strings.Contains(diff, "User.php") {
		t.Error("diff should contain User.php")
	}
	if !strings.Contains(diff, "web.php") {
		t.Error("diff should contain web.php")
	}
}
