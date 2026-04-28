package context

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/larasense/larasense-limbo/internal/config"
	"github.com/larasense/larasense-limbo/internal/diff"
	"github.com/larasense/larasense-limbo/internal/git"
)

// FileContext holds the analysis context for a single changed file.
type FileContext struct {
	Path         string `json:"path"`
	Type         string `json:"type"`
	Hint         string `json:"hint"`
	DiffText     string `json:"diff_text"`
	Surrounding  string `json:"surrounding_context"`
	ChangedLines []int  `json:"changed_lines"`
}

// ReviewContext holds the full context sent to the AI for review.
type ReviewContext struct {
	Files []FileContext `json:"files"`
}

// Builder constructs Laravel-aware context from parsed diffs.
type Builder struct {
	cfg  *config.Config
	head string
}

// NewBuilder creates a new context builder.
func NewBuilder(cfg *config.Config, head string) *Builder {
	return &Builder{cfg: cfg, head: head}
}

// Build creates a ReviewContext from a set of parsed file diffs.
func (b *Builder) Build(files []diff.FileDiff) (*ReviewContext, error) {
	var filtered []diff.FileDiff
	for _, f := range files {
		if b.shouldInclude(f.Path) {
			filtered = append(filtered, f)
		}
	}

	if len(filtered) == 0 {
		return &ReviewContext{}, nil
	}

	ctx := &ReviewContext{}
	for _, f := range filtered {
		fc := FileContext{
			Path:         f.Path,
			Type:         classifyFile(f.Path),
			Hint:         generateHint(f.Path),
			DiffText:     f.DiffText(),
			ChangedLines: f.ChangedLines(),
		}

		// Get surrounding context (±20 lines around first changed line)
		if len(fc.ChangedLines) > 0 && !f.IsDeleted {
			surrounding, err := git.GetSurroundingLines(b.head, f.Path, fc.ChangedLines[0], 20)
			if err == nil {
				fc.Surrounding = surrounding
			}
			// Non-fatal: we proceed without surrounding context if git show fails
		}

		ctx.Files = append(ctx.Files, fc)
	}

	return ctx, nil
}

// shouldInclude checks if a file path matches the configured include/exclude filters
// and is a Laravel-relevant file.
func (b *Builder) shouldInclude(path string) bool {
	// Must be a PHP or Blade file
	if !isLaravelFile(path) {
		return false
	}

	// Check exclude patterns first
	for _, pattern := range b.cfg.Filters.Exclude {
		if matchGlob(pattern, path) {
			return false
		}
	}

	// If include patterns are defined, file must match at least one
	if len(b.cfg.Filters.Include) > 0 {
		for _, pattern := range b.cfg.Filters.Include {
			if matchGlob(pattern, path) {
				return true
			}
		}
		return false
	}

	return true
}

// isLaravelFile checks if the file is a PHP/Blade file in a Laravel-relevant directory.
func isLaravelFile(path string) bool {
	ext := filepath.Ext(path)

	// Blade templates
	if strings.HasSuffix(path, ".blade.php") {
		return true
	}

	// Regular PHP files
	if ext != ".php" {
		return false
	}

	// Must be in a Laravel-relevant directory
	laravelDirs := []string{
		"app/",
		"routes/",
		"resources/views/",
		"config/",
		"database/migrations/",
		"database/factories/",
	}

	normalizedPath := filepath.ToSlash(path)
	for _, dir := range laravelDirs {
		if strings.HasPrefix(normalizedPath, dir) {
			return true
		}
	}

	return false
}

// classifyFile determines the Laravel component type of a file.
func classifyFile(path string) string {
	normalizedPath := filepath.ToSlash(path)

	switch {
	case strings.Contains(normalizedPath, "app/Http/Controllers/"):
		return "controller"
	case strings.Contains(normalizedPath, "app/Models/"):
		return "model"
	case strings.Contains(normalizedPath, "app/Http/Middleware/"):
		return "middleware"
	case strings.Contains(normalizedPath, "app/Http/Requests/"):
		return "form_request"
	case strings.Contains(normalizedPath, "app/Services/"):
		return "service"
	case strings.Contains(normalizedPath, "app/Repositories/"):
		return "repository"
	case strings.Contains(normalizedPath, "app/Events/"):
		return "event"
	case strings.Contains(normalizedPath, "app/Listeners/"):
		return "listener"
	case strings.Contains(normalizedPath, "app/Jobs/"):
		return "job"
	case strings.Contains(normalizedPath, "app/Mail/"):
		return "mailable"
	case strings.Contains(normalizedPath, "app/Notifications/"):
		return "notification"
	case strings.Contains(normalizedPath, "app/Policies/"):
		return "policy"
	case strings.Contains(normalizedPath, "app/Providers/"):
		return "service_provider"
	case strings.HasSuffix(normalizedPath, ".blade.php"):
		return "blade_view"
	case strings.HasPrefix(normalizedPath, "routes/"):
		return "route"
	case strings.HasPrefix(normalizedPath, "config/"):
		return "config"
	case strings.HasPrefix(normalizedPath, "database/migrations/"):
		return "migration"
	case strings.HasPrefix(normalizedPath, "database/factories/"):
		return "factory"
	default:
		return "php"
	}
}

// generateHint produces a contextual hint for the AI about what this file is.
func generateHint(path string) string {
	fileType := classifyFile(path)

	hints := map[string]string{
		"controller":       "This is a Laravel Controller. Check for fat controller anti-pattern, proper validation, and resource usage.",
		"model":            "This is an Eloquent Model. Check for mass assignment protection ($fillable/$guarded), proper relationships, and N+1 query risks.",
		"middleware":       "This is HTTP Middleware. Check for proper request/response handling and security concerns.",
		"form_request":     "This is a Form Request. Check validation rules completeness and authorization logic.",
		"service":          "This is a Service class. Check for single responsibility and proper dependency injection.",
		"repository":       "This is a Repository class. Check for proper query building and Eloquent usage.",
		"event":            "This is an Event class. Check for proper event data structure.",
		"listener":         "This is an Event Listener. Check for proper queue handling and error management.",
		"job":              "This is a Queue Job. Check for proper retry logic, timeout settings, and idempotency.",
		"mailable":         "This is a Mailable class. Check for proper view binding and queue usage.",
		"notification":     "This is a Notification class. Check for proper channel configuration.",
		"policy":           "This is an Authorization Policy. Check for proper gate/policy logic.",
		"service_provider": "This is a Service Provider. Check for proper binding registration and boot logic.",
		"blade_view":       "This is a Blade template. Check for XSS vulnerabilities (unescaped output), logic in views, and proper directive usage.",
		"route":            "This is a route definition file. Check for proper middleware assignment, route naming, and RESTful conventions.",
		"config":           "This is a configuration file. Check for hardcoded secrets and proper env() usage.",
		"migration":        "This is a database migration. Check for proper column types, indexes, and rollback support.",
		"factory":          "This is a Model Factory. Check for realistic fake data generation.",
	}

	if hint, ok := hints[fileType]; ok {
		return hint
	}
	return "This is a PHP file in the Laravel project."
}

// matchGlob performs simple glob matching supporting ** and * patterns.
func matchGlob(pattern, path string) bool {
	pattern = filepath.ToSlash(pattern)
	path = filepath.ToSlash(path)

	// Handle ** (match any number of directories)
	if strings.Contains(pattern, "**") {
		// Convert "app/**" to match anything under "app/"
		prefix := strings.Split(pattern, "**")[0]
		if strings.HasPrefix(path, prefix) {
			return true
		}
		return false
	}

	// Use filepath.Match for simple glob patterns
	matched, err := filepath.Match(pattern, path)
	if err != nil {
		return false
	}
	return matched
}

// FormatForAI converts the ReviewContext into a string suitable for AI prompt input.
func (rc *ReviewContext) FormatForAI() string {
	var sb strings.Builder

	for i, f := range rc.Files {
		if i > 0 {
			sb.WriteString("\n---\n\n")
		}

		fmt.Fprintf(&sb, "File: %s\n", f.Path)
		fmt.Fprintf(&sb, "Type: %s\n", f.Type)
		fmt.Fprintf(&sb, "Hint: %s\n\n", f.Hint)

		if f.Surrounding != "" {
			sb.WriteString("Surrounding Context:\n")
			sb.WriteString(f.Surrounding)
			sb.WriteString("\n")
		}

		sb.WriteString("Changes (diff):\n")
		sb.WriteString(f.DiffText)
	}

	return sb.String()
}
