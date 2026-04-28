# Larasense Limbo

AI-powered code review CLI for Laravel projects. Analyzes git diffs and detects performance issues, security vulnerabilities, bad practices, and convention violations — using any OpenAI-compatible AI provider.

## Features

- **Git Diff Analysis** — Reads `git diff` between any two refs (branches, commits, tags)
- **Laravel-Aware Filtering** — Only processes `.php` and `.blade.php` files in relevant directories (`app/`, `routes/`, `resources/views/`, etc.)
- **Smart File Classification** — Automatically identifies 18 Laravel component types (Controller, Model, Middleware, Form Request, Job, Event, etc.) and provides contextual hints to the AI
- **Structured AI Review** — Sends diff + context to an AI provider with a strict Laravel code review prompt
- **Configurable** — YAML config file with environment variable support (`${VAR}` syntax)
- **Dual Output** — Human-readable terminal output with severity icons, or JSON for CI pipelines
- **CI-Ready** — Exits with code `1` when high-severity issues are found
- **Retry & Timeout** — 3 retries with exponential backoff, 120s request timeout
- **Multi-Format Response Parsing** — Handles OpenAI Responses API, Chat Completions API, and direct JSON

## What It Detects

| Category | Examples |
|----------|---------|
| **Performance** | N+1 queries, missing eager loading, unnecessary DB queries, inefficient loops |
| **Security** | Missing validation, mass assignment (`$request->all()`), SQL injection, XSS in Blade, CSRF |
| **Bad Practices** | Fat controllers, business logic in Blade views, missing Form Requests, hardcoded values |
| **Conventions** | Improper Eloquent usage, missing route model binding, naming violations, missing middleware |

## Requirements

- Go 1.21+ (for building from source)
- Git (accessible via `PATH`)
- An OpenAI-compatible API key

## Installation

### From Source

```bash
git clone https://github.com/Mattel-Limbo/larasense-limbo.git
cd larasense-limbo
go build -o larasense-limbo .
```

The binary will be created in the current directory. Move it to a directory in your `PATH` for global access:

```bash
# Linux/macOS
sudo mv larasense-limbo /usr/local/bin/

# Windows — move to a directory in your PATH
move larasense-limbo.exe C:\your\bin\path\
```

### Verify Installation

```bash
larasense-limbo --help
```

## Quick Start

1. **Generate a config file** in your Laravel project root:

```bash
cd /path/to/your/laravel-project
larasense-limbo init
```

This creates `.larasense-limbo.yml` with sensible defaults. Edit it:

```yaml
provider:
  base_url: https://api.openai.com
  api_key: ${AI_API_KEY}
  model: gpt-4.1

review:
  max_issues: 5
  severity_threshold: medium

filters:
  include:
    - "app/**"
    - "routes/**"
    - "resources/views/**"
  exclude:
    - "tests/**"
    - "database/seeders/**"
```

2. **Set your API key:**

```bash
export AI_API_KEY=sk-your-api-key-here
```

3. **Run the review:**

```bash
larasense-limbo analyze
```

## Usage

### Basic Usage

```bash
# Review changes between origin/main and HEAD (default)
larasense-limbo analyze

# Review changes between specific refs
larasense-limbo analyze --base origin/develop --head feature/user-auth

# Output as JSON (for CI/CD pipelines)
larasense-limbo analyze --json

# Show detailed request/response logs for debugging
larasense-limbo analyze --base main --verbose

# Combine flags
larasense-limbo analyze --base main --head HEAD --json
```

### CLI Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--base` | string | `origin/main` | Base ref for diff comparison |
| `--head` | string | `HEAD` | Head ref for diff comparison |
| `--json` | bool | `false` | Output results as JSON instead of human-readable format |
| `--verbose` | bool | `false` | Show detailed AI request/response logs (URL, body, timing, status) |
| `--no-cache` | bool | `false` | Skip cache and re-review all files |
| `--format` | string | `human` | Output format: `human`, `json`, `github` |
| `--github-pr` | string | | Post results as PR comment (format: `owner/repo#number`) |

### Output Examples

**Human-readable output:**

```
╔══════════════════════════════════════════════════════════════╗
║              Laravel AI Code Review Results                 ║
╚══════════════════════════════════════════════════════════════╝

  Reviewed 3 file(s). Found 2 issue(s): 1 high, 1 medium, 0 low.

  📄 app/Http/Controllers/UserController.php
  ──────────────────────────────────────────────────────────────

  🔴 [HIGH] #1: Mass Assignment Vulnerability
     Line: 15
     Using $request->all() passes unvalidated data directly to create()
     💡 Suggestion: Use a Form Request with validated() method

  🟡 [MEDIUM] #2: Missing Eager Loading
     Line: 8
     Querying users without eager loading related posts may cause N+1
     💡 Suggestion: Use User::with('posts')->paginate(15)

══════════════════════════════════════════════════════════════
```

**JSON output (`--json`):**

```json
{
  "issues": [
    {
      "title": "Mass Assignment Vulnerability",
      "description": "Using $request->all() passes unvalidated data directly to create()",
      "file": "app/Http/Controllers/UserController.php",
      "line": 15,
      "severity": "high",
      "suggestion": "Use a Form Request with validated() method"
    }
  ],
  "files_analyzed": 3,
  "summary": "Reviewed 3 file(s). Found 1 issue(s): 1 high, 0 medium, 0 low."
}
```

## Other Commands

### Init

Generate a default `.larasense-limbo.yml` config file in the current directory:

```bash
# Generate config (fails if file already exists)
larasense-limbo init

# Overwrite existing config
larasense-limbo init --force
```

### Version

Print version and build information:

```bash
larasense-limbo version
```

Output:

```
larasense-limbo 1.0.0
  commit:  abc1234
  built:   2026-04-28
  go:      go1.21.0
  os/arch: linux/amd64
```

Version info is injected at build time via `-ldflags`:

```bash
go build -ldflags "\
  -X github.com/Mattel-Limbo/larasense-limbo/cmd.Version=1.0.0 \
  -X github.com/Mattel-Limbo/larasense-limbo/cmd.Commit=$(git rev-parse --short HEAD) \
  -X github.com/Mattel-Limbo/larasense-limbo/cmd.BuildDate=$(date -u +%Y-%m-%d)" \
  -o larasense-limbo .
```

## Configuration

### Config File

Larasense Limbo reads `.larasense-limbo.yml` from the current working directory.

```yaml
# AI Provider settings
provider:
  base_url: https://api.openai.com    # Required — API base URL
  api_key: ${AI_API_KEY}              # Required — supports ${VAR} env expansion
  model: gpt-4.1                      # Model to use (default: gpt-4.1)

# Review behavior
review:
  max_issues: 5                       # Max issues to return (default: 5)
  severity_threshold: medium          # Minimum severity: low, medium, high (default: medium)
  # custom_prompt: "Focus on security"  # Additional instructions appended to AI prompt (optional)

# File filtering (glob patterns)
filters:
  include:                            # Only analyze files matching these patterns
    - "app/**"
    - "routes/**"
    - "resources/views/**"
  exclude:                            # Skip files matching these patterns
    - "tests/**"
    - "database/seeders/**"
```

### Environment Variables

All config values can be overridden via environment variables with the `LARASENSE_LIMBO_` prefix:

| Environment Variable | Config Key | Description |
|---------------------|------------|-------------|
| `LARASENSE_LIMBO_PROVIDER_BASE_URL` | `provider.base_url` | AI provider API base URL |
| `LARASENSE_LIMBO_PROVIDER_API_KEY` | `provider.api_key` | API key |
| `LARASENSE_LIMBO_PROVIDER_MODEL` | `provider.model` | Model name |
| `LARASENSE_LIMBO_REVIEW_MAX_ISSUES` | `review.max_issues` | Maximum issues to return |
| `LARASENSE_LIMBO_REVIEW_SEVERITY_THRESHOLD` | `review.severity_threshold` | Minimum severity level |

The `api_key` field also supports `${VAR}` syntax for inline env var expansion (e.g., `api_key: ${AI_API_KEY}`).

### Supported AI Providers

Use `provider.name` for quick setup with built-in presets, or set `base_url` manually for any OpenAI-compatible API:

```yaml
# Quick setup with preset (auto-fills base_url, model, endpoint)
provider:
  name: openai        # or: anthropic, gemini, ollama, openrouter
  api_key: ${AI_API_KEY}

# Or manual setup for any provider
provider:
  base_url: http://localhost:1430
  api_key: ${AI_API_KEY}
  model: claude-sonnet-4.5
  endpoint: chat
```

**Available presets:**

| Preset | Base URL | Default Model | API Key Required |
|--------|----------|---------------|---|
| `openai` | `https://api.openai.com` | `gpt-4.1` | Yes |
| `anthropic` | `https://api.anthropic.com` | `claude-sonnet-4-20250514` | Yes |
| `gemini` | `https://generativelanguage.googleapis.com/v1beta/openai` | `gemini-2.5-flash` | Yes |
| `ollama` | `http://localhost:11434` | `llama3.1` | No |
| `openrouter` | `https://openrouter.ai/api` | `openai/gpt-4.1` | Yes |

You can override any preset field — e.g., `name: openai` with `model: gpt-4o` uses OpenAI's URL but a different model.

### Custom Prompt

You can add custom instructions that get appended to the built-in AI review prompt:

```yaml
review:
  custom_prompt: "Focus only on security vulnerabilities and SQL injection risks. Ignore performance issues."
```

More examples:

```yaml
# Only check for N+1 queries
custom_prompt: "Only report N+1 query problems and missing eager loading."

# Review for a specific Laravel version
custom_prompt: "This project uses Laravel 11. Check for deprecated features from Laravel 10."

# Stricter review
custom_prompt: "Be extra strict. Report any method longer than 20 lines as a bad practice."
```

### Cache

Larasense-limbo caches review results per file using SHA-256 hashes. On subsequent runs, unchanged files are skipped — saving API calls and time.

```bash
# Normal run (uses cache)
larasense-limbo analyze --base main

# Force re-review all files
larasense-limbo analyze --base main --no-cache
```

Cache is stored in `.larasense-limbo-cache.json` (auto-added to `.gitignore`).

### GitHub PR Comments

Post review results directly as a comment on a GitHub pull request:

```bash
export GITHUB_TOKEN=ghp_your_token
larasense-limbo analyze --base main --github-pr Mattel-Limbo/larasense-limbo#42
```

This posts a formatted markdown comment on PR #42 with all issues grouped by file.

## Laravel File Classification

The tool automatically classifies changed files into 18 Laravel component types and provides contextual hints to the AI:

| Type | Directory Pattern | Review Focus |
|------|------------------|--------------|
| Controller | `app/Http/Controllers/` | Fat controller, validation, resource usage |
| Model | `app/Models/` | Mass assignment, relationships, N+1 risks |
| Middleware | `app/Http/Middleware/` | Request/response handling, security |
| Form Request | `app/Http/Requests/` | Validation rules, authorization |
| Service | `app/Services/` | Single responsibility, dependency injection |
| Repository | `app/Repositories/` | Query building, Eloquent usage |
| Event | `app/Events/` | Event data structure |
| Listener | `app/Listeners/` | Queue handling, error management |
| Job | `app/Jobs/` | Retry logic, timeout, idempotency |
| Mailable | `app/Mail/` | View binding, queue usage |
| Notification | `app/Notifications/` | Channel configuration |
| Policy | `app/Policies/` | Gate/policy logic |
| Service Provider | `app/Providers/` | Binding registration, boot logic |
| Blade View | `resources/views/**/*.blade.php` | XSS, logic in views, directives |
| Route | `routes/` | Middleware, naming, RESTful conventions |
| Config | `config/` | Hardcoded secrets, env() usage |
| Migration | `database/migrations/` | Column types, indexes, rollback |
| Factory | `database/factories/` | Realistic fake data |

## CI/CD Integration

### GitHub Actions (with inline annotations)

Use `--format github` to get inline annotations directly on PR files:

```yaml
name: Laravel Code Review

on:
  pull_request:
    branches: [main, develop]

jobs:
  review:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'

      - name: Install larasense-limbo
        run: go install github.com/Mattel-Limbo/larasense-limbo@latest

      - name: Run AI Code Review
        env:
          AI_API_KEY: ${{ secrets.AI_API_KEY }}
        run: |
          larasense-limbo analyze \
            --base origin/${{ github.base_ref }} \
            --head ${{ github.sha }} \
            --format github
```

This produces inline annotations on the PR:
- `high` severity → `::error` (red)
- `medium` severity → `::warning` (yellow)
- `low` severity → `::notice` (blue)

### Docker

```bash
# Build
docker build -t larasense-limbo .

# Run (mount your Laravel project)
docker run --rm -v /path/to/laravel:/repo \
  -e AI_API_KEY=your-key \
  larasense-limbo analyze --base main
```

### Release

Releases are automated via GoReleaser on tag push:

```bash
git tag v1.0.0
git push origin v1.0.0
# → GitHub Actions builds binaries for linux/darwin/windows × amd64/arm64
```

Download pre-built binaries from [Releases](https://github.com/Mattel-Limbo/larasense-limbo/releases).

### Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Review completed — no high-severity issues |
| `1` | Review completed — high-severity issues found (or execution error) |

## Architecture

```
larasense-limbo/
├── main.go                          # Entry point
├── cmd/
│   ├── root.go                      # Root CLI command
│   ├── analyze.go                   # analyze subcommand + flags
│   ├── init.go                      # init subcommand (config generator)
│   └── version.go                   # version subcommand (build info)
├── internal/
│   ├── config/config.go             # Viper YAML + env config loader
│   ├── git/git.go                   # Git operations (exec)
│   ├── diff/parser.go               # Git diff parser
│   ├── context/builder.go           # Laravel-aware context builder
│   ├── ai/client.go                 # AI provider HTTP client
│   ├── reviewer/reviewer.go         # Pipeline orchestrator
│   └── output/formatter.go          # Human + JSON + GitHub annotations output
├── .larasense-limbo.yml             # Example config
├── Dockerfile                       # Multi-stage container build
├── Makefile                         # Build/test/install targets
├── .goreleaser.yml                  # Release configuration
└── .github/workflows/
    ├── ci.yml                       # CI: test + build + lint
    └── release.yml                  # Release on tag push
```

### Data Flow

```
larasense-limbo analyze
  │
  ├─ config.Load()                    Load .larasense-limbo.yml + env vars
  │
  ├─ git.GetDiff(base, head)          Execute: git diff base...head
  │
  ├─ diff.Parse(rawDiff)              Parse into []FileDiff with hunks
  │
  ├─ context.Builder.Build(files)     Filter Laravel files, classify, add hints
  │   ├─ shouldInclude()              Glob include/exclude + isLaravelFile
  │   ├─ classifyFile()               Identify component type (18 types)
  │   ├─ generateHint()               AI context hint per type
  │   └─ git.GetSurroundingLines()    Fetch ±20 lines around changes
  │
  ├─ ai.Client.Analyze(diff, ctx)     HTTP POST to AI provider
  │   ├─ Retry (3x, exponential)      2s → 4s backoff
  │   └─ extractContent()             Parse multi-format response
  │
  ├─ filterIssues()                   Apply severity threshold + max count
  │
  └─ output.Format*()                 Human-readable or JSON
```

## Development

A `Makefile` is provided for common tasks:

```bash
make build      # Build binary with version info from git
make test       # Run all tests
make vet        # Run go vet
make lint       # Run vet + test
make clean      # Remove build artifacts
make install    # Install to $GOPATH/bin
make run        # Build and run analyze (usage: make run ARGS="--base main")
make version    # Build and show version
make help       # Show all targets
```

Or use Go directly:

```bash
go build -o larasense-limbo .
go test ./... -v
go vet ./...
```

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Language | Go |
| CLI Framework | [Cobra](https://github.com/spf13/cobra) |
| Configuration | [Viper](https://github.com/spf13/viper) (YAML + ENV) |
| HTTP Client | `net/http` (stdlib) |
| Git Integration | `os/exec` (native git) |

## License

MIT
