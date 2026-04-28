package context

import (
	"testing"
)

func TestClassifyFile(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"app/Http/Controllers/UserController.php", "controller"},
		{"app/Models/User.php", "model"},
		{"app/Http/Middleware/Authenticate.php", "middleware"},
		{"app/Http/Requests/StoreUserRequest.php", "form_request"},
		{"app/Services/PaymentService.php", "service"},
		{"app/Jobs/SendEmail.php", "job"},
		{"app/Events/UserRegistered.php", "event"},
		{"app/Listeners/SendWelcomeEmail.php", "listener"},
		{"app/Mail/WelcomeMail.php", "mailable"},
		{"app/Policies/PostPolicy.php", "policy"},
		{"app/Providers/AppServiceProvider.php", "service_provider"},
		{"resources/views/users/index.blade.php", "blade_view"},
		{"routes/web.php", "route"},
		{"config/app.php", "config"},
		{"database/migrations/2024_01_01_create_users_table.php", "migration"},
		{"database/factories/UserFactory.php", "factory"},
		{"app/Helpers/StringHelper.php", "php"},
	}

	for _, tt := range tests {
		got := classifyFile(tt.path)
		if got != tt.expected {
			t.Errorf("classifyFile(%q) = %q, want %q", tt.path, got, tt.expected)
		}
	}
}

func TestIsLaravelFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"app/Http/Controllers/UserController.php", true},
		{"app/Models/User.php", true},
		{"routes/web.php", true},
		{"resources/views/home.blade.php", true},
		{"config/app.php", true},
		{"database/migrations/2024_create_users.php", true},
		{"tests/Feature/UserTest.php", false},
		{"vendor/laravel/framework/src/Auth.php", false},
		{"public/index.php", false},
		{"README.md", false},
		{"package.json", false},
		{"app/Models/User.js", false},
	}

	for _, tt := range tests {
		got := isLaravelFile(tt.path)
		if got != tt.expected {
			t.Errorf("isLaravelFile(%q) = %v, want %v", tt.path, got, tt.expected)
		}
	}
}

func TestGenerateHint(t *testing.T) {
	hint := generateHint("app/Http/Controllers/UserController.php")
	if hint == "" {
		t.Error("expected non-empty hint for controller")
	}

	hint = generateHint("app/Models/User.php")
	if hint == "" {
		t.Error("expected non-empty hint for model")
	}

	hint = generateHint("resources/views/home.blade.php")
	if hint == "" {
		t.Error("expected non-empty hint for blade view")
	}
}

func TestMatchGlob(t *testing.T) {
	tests := []struct {
		pattern  string
		path     string
		expected bool
	}{
		{"app/**", "app/Http/Controllers/UserController.php", true},
		{"app/**", "app/Models/User.php", true},
		{"routes/**", "routes/web.php", true},
		{"resources/views/**", "resources/views/home.blade.php", true},
		{"tests/**", "tests/Feature/UserTest.php", true},
		{"tests/**", "app/Models/User.php", false},
		{"app/**", "routes/web.php", false},
	}

	for _, tt := range tests {
		got := matchGlob(tt.pattern, tt.path)
		if got != tt.expected {
			t.Errorf("matchGlob(%q, %q) = %v, want %v", tt.pattern, tt.path, got, tt.expected)
		}
	}
}
