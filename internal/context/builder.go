package context

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Mattel-Limbo/larasense-limbo/internal/config"
	"github.com/Mattel-Limbo/larasense-limbo/internal/diff"
	"github.com/Mattel-Limbo/larasense-limbo/internal/git"
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

		// Get surrounding context with adaptive window size
		contextLines := b.cfg.Review.ContextLines
		if contextLines == 0 {
			contextLines = 10 // default fallback
		}
		// Adaptive: use smaller window for small diffs
		if len(fc.ChangedLines) > 0 && len(fc.ChangedLines) < 10 && contextLines > 5 {
			contextLines = 5
		}

		if len(fc.ChangedLines) > 0 && !f.IsDeleted && contextLines > 0 {
			surrounding, err := git.GetSurroundingLines(b.head, f.Path, fc.ChangedLines[0], contextLines)
			if err == nil {
				fc.Surrounding = surrounding
			}
			// Non-fatal: we proceed without surrounding context if git show fails
		}

		ctx.Files = append(ctx.Files, fc)
	}

	return ctx, nil
}

func (b *Builder) shouldInclude(path string) bool {
	return ShouldIncludeFile(path, b.cfg)
}

// ShouldIncludeFile checks if a file path matches the configured include/exclude filters
// and is a Laravel-relevant file.
func ShouldIncludeFile(path string, cfg *config.Config) bool {
	if !isLaravelFile(path) {
		return false
	}

	for _, pattern := range cfg.Filters.Exclude {
		if matchGlob(pattern, path) {
			return false
		}
	}

	if len(cfg.Filters.Include) > 0 {
		for _, pattern := range cfg.Filters.Include {
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

// ClassifyFile determines the Laravel component type of a file.
func ClassifyFile(path string) string {
	return classifyFile(path)
}

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

// GenerateHint produces a contextual hint for the AI about what this file type is.
func GenerateHint(path string) string {
	return generateHint(path)
}

// generateHint produces a concise keyword-based hint for the AI about what to check.
// Optimized for minimal token usage — uses short tags instead of full sentences.
func generateHint(path string) string {
	fileType := classifyFile(path)

	hints := map[string]string{
		"controller":       "Check: fat-controller, validation, resource-usage",
		"model":            "Check: N+1, mass-assignment, $fillable/$guarded, relationships",
		"middleware":       "Check: request/response-handling, security",
		"form_request":     "Check: validation-rules, authorization",
		"service":          "Check: SRP, dependency-injection",
		"repository":       "Check: query-building, Eloquent-usage",
		"event":            "Check: event-data-structure",
		"listener":         "Check: queue-handling, error-management",
		"job":              "Check: retry-logic, timeout, idempotency",
		"mailable":         "Check: view-binding, queue-usage",
		"notification":     "Check: channel-config",
		"policy":           "Check: gate/policy-logic",
		"service_provider": "Check: binding-registration, boot-logic",
		"blade_view":       "Check: XSS, unescaped-output, logic-in-views",
		"route":            "Check: middleware, route-naming, RESTful",
		"config":           "Check: hardcoded-secrets, env()-usage",
		"migration":        "Check: column-types, indexes, rollback",
		"factory":          "Check: realistic-fake-data",
	}

	if hint, ok := hints[fileType]; ok {
		return hint
	}
	return "Check: general-php"
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

// FormatForAI converts the ReviewContext into a string suitable for diff-based AI prompt input.
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

// FormatForScan converts the ReviewContext into a string for full-file scan AI prompt input.
func (rc *ReviewContext) FormatForScan() string {
	var sb strings.Builder

	for i, f := range rc.Files {
		if i > 0 {
			sb.WriteString("\n---\n\n")
		}

		fmt.Fprintf(&sb, "File: %s [%s]\n", f.Path, f.Type)
		sb.WriteString(f.DiffText)
	}

	return sb.String()
}
