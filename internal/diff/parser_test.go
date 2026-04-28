package diff

import (
	"testing"
)

const sampleDiff = `diff --git a/app/Http/Controllers/UserController.php b/app/Http/Controllers/UserController.php
index abc1234..def5678 100644
--- a/app/Http/Controllers/UserController.php
+++ b/app/Http/Controllers/UserController.php
@@ -10,6 +10,12 @@ class UserController extends Controller
     public function index()
     {
-        $users = User::all();
+        $users = User::with('posts')->paginate(15);
+        return view('users.index', compact('users'));
+    }
+
+    public function store(Request $request)
+    {
+        $user = User::create($request->all());
         return view('users.index', compact('users'));
     }
 }
diff --git a/app/Models/User.php b/app/Models/User.php
new file mode 100644
index 0000000..abc1234
--- /dev/null
+++ b/app/Models/User.php
@@ -0,0 +1,10 @@
+<?php
+
+namespace App\Models;
+
+use Illuminate\Database\Eloquent\Model;
+
+class User extends Model
+{
+    protected $guarded = [];
+}
`

func TestParse(t *testing.T) {
	files, err := Parse(sampleDiff)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}

	// First file: UserController.php
	ctrl := files[0]
	if ctrl.Path != "app/Http/Controllers/UserController.php" {
		t.Errorf("expected path 'app/Http/Controllers/UserController.php', got '%s'", ctrl.Path)
	}
	if ctrl.IsNew {
		t.Error("UserController should not be marked as new")
	}
	if len(ctrl.Hunks) == 0 {
		t.Error("expected at least one hunk in UserController")
	}

	changedLines := ctrl.ChangedLines()
	if len(changedLines) == 0 {
		t.Error("expected changed lines in UserController")
	}

	diffText := ctrl.DiffText()
	if diffText == "" {
		t.Error("expected non-empty diff text for UserController")
	}

	// Second file: User.php (new file)
	model := files[1]
	if model.Path != "app/Models/User.php" {
		t.Errorf("expected path 'app/Models/User.php', got '%s'", model.Path)
	}
	if !model.IsNew {
		t.Error("User model should be marked as new")
	}
}

func TestParseEmpty(t *testing.T) {
	files, err := Parse("")
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if files != nil {
		t.Errorf("expected nil for empty diff, got %v", files)
	}
}

func TestParseHunkHeader(t *testing.T) {
	tests := []struct {
		header   string
		expected int
	}{
		{"@@ -10,6 +10,12 @@ class UserController", 10},
		{"@@ -0,0 +1,10 @@", 1},
		{"@@ -5,3 +5,8 @@ namespace App", 5},
		{"@@ -1 +1 @@", 1},
	}

	for _, tt := range tests {
		got := parseHunkHeader(tt.header)
		if got != tt.expected {
			t.Errorf("parseHunkHeader(%q) = %d, want %d", tt.header, got, tt.expected)
		}
	}
}

func TestChangedLines(t *testing.T) {
	files, err := Parse(sampleDiff)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	// The new User.php file should have lines 1-10 as changed
	model := files[1]
	lines := model.ChangedLines()
	if len(lines) == 0 {
		t.Fatal("expected changed lines in new file")
	}
	// First changed line should be 1 (start of new file)
	if lines[0] != 1 {
		t.Errorf("expected first changed line to be 1, got %d", lines[0])
	}
}
