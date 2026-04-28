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
git clone https://github.com/larasense/larasense-limbo.git
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

1. **Create a config file** in your Laravel project root:

```bash
cd /path/to/your/laravel-project
```

Create `.larasense-limbo.yml`:

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

# Combine flags
larasense-limbo analyze --base main --head HEAD --json
```

### CLI Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--base` | string | `origin/main` | Base ref for diff comparison |
| `--head` | string | `HEAD` | Head ref for diff comparison |
| `--json` | bool | `false` | Output results as JSON instead of human-readable format |

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

Any OpenAI-compatible API works. The tool sends requests to `{base_url}/v1/responses` and parses responses in these formats:

| Provider | `base_url` | Notes |
|----------|-----------|-------|
| OpenAI | `https://api.openai.com` | Responses API and Chat Completions |
| Azure OpenAI | `https://your-resource.openai.azure.com` | With compatible endpoint |
| OpenRouter | `https://openrouter.ai/api` | Multi-model gateway |
| Local (Ollama, LM Studio) | `http://localhost:11434` | Self-hosted models |

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

### GitHub Actions

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
          fetch-depth: 0  # Full history for git diff

      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'

      - name: Install larasense-limbo
        run: |
          go install github.com/larasense/larasense-limbo@latest

      - name: Run AI Code Review
        env:
          AI_API_KEY: ${{ secrets.AI_API_KEY }}
        run: |
          larasense-limbo analyze \
            --base origin/${{ github.base_ref }} \
            --head ${{ github.sha }} \
            --json
```

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
│   └── analyze.go                   # analyze subcommand + flags
├── internal/
│   ├── config/config.go             # Viper YAML + env config loader
│   ├── git/git.go                   # Git operations (exec)
│   ├── diff/parser.go               # Git diff parser
│   ├── context/builder.go           # Laravel-aware context builder
│   ├── ai/client.go                 # AI provider HTTP client
│   ├── reviewer/reviewer.go         # Pipeline orchestrator
│   └── output/formatter.go          # Human + JSON output
└── .larasense-limbo.yml             # Example config
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

### Build

```bash
go build -o larasense-limbo .
```

### Test

```bash
go test ./... -v
```

### Vet

```bash
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
