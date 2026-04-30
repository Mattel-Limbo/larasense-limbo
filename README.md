# Larasense Limbo

AI-powered code review CLI for Laravel projects. Analyzes git diffs **or scans your entire codebase** to detect performance issues, security vulnerabilities, bad practices, and convention violations — using any OpenAI-compatible AI provider.

## Features

- **Git Diff Analysis** — Reads `git diff` between any two refs (branches, commits, tags)
- **Full Codebase Scan** — Audit all Laravel files in a project, not just changed ones — perfect for initial adoption or periodic audits
- **Laravel-Aware Filtering** — Only processes `.php` and `.blade.php` files in relevant directories (`app/`, `routes/`, `resources/views/`, etc.)
- **Smart File Classification** — Automatically identifies 18 Laravel component types (Controller, Model, Middleware, Form Request, Job, Event, etc.) and provides contextual hints to the AI
- **Structured AI Review** — Sends diff + context to an AI provider with a strict Laravel code review prompt
- **Token-Aware Batching** — Automatically splits large codebases into batches (~20K tokens each) to stay within AI provider limits
- **Configurable** — YAML config file with environment variable support (`${VAR}` syntax)
- **Dual Output** — Human-readable terminal output with severity icons, or JSON for CI pipelines
- **CI-Ready** — Exits with code `1` when high-severity issues are found
- **Retry & Timeout** — 3 retries with exponential backoff, 120s request timeout
- **Multi-Format Response Parsing** — Handles OpenAI Responses API, Chat Completions API, and direct JSON

## What It Detects

| Category | Examples |
|----------|---------|
| **Security** | SQL injection, mass assignment (`$request->all()`), XSS in Blade, hardcoded secrets, missing auth middleware |
| **Logic Errors** | Undefined variables, wrong variable in loop (`foreach $x` but uses `$y`), unreachable code after return |
| **Semantic Errors** | Wrong comparison (`=` vs `===`), type mismatch, inverted logic (`&&` vs `||`) |
| **Performance** | N+1 queries, missing eager loading, unbounded `Model::all()`, inefficient loops |
| **Bad Practices** | Fat controllers, business logic in Blade views, missing Form Requests, missing error handling |
| **Conventions** | Improper Eloquent usage, missing route model binding, naming violations, dead code |

## Requirements

- Go 1.21+ (for building from source)
- Git (accessible via `PATH`)
- An OpenAI-compatible API key

## Installation

### From Source

```bash
git clone https://github.com/Mattel-Limbo/larasense-limbo.git
cd larasense-limbo

# Linux/macOS
go build -o larasense-limbo .

# Windows (the .exe extension is required)
go build -o larasense-limbo.exe .
```

The binary will be created in the current directory. Move it to a directory in your `PATH` for global access:

```bash
# Linux/macOS
sudo mv larasense-limbo /usr/local/bin/

# Windows — move to a directory in your PATH
move larasense-limbo.exe C:\your\bin\path\
```

> **Windows users:** Always build with `-o larasense-limbo.exe`. Without the `.exe` extension, Windows will not recognize the binary as an executable and may try to "open" it instead of running it.

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
# Review only changed files (git diff)
larasense-limbo analyze

# Or scan the entire codebase
larasense-limbo scan
```

## Usage

### Analyze (Diff-Based Review)

Review only changed files between two git refs:

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

#### Analyze Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--base` | string | `origin/main` | Base ref for diff comparison |
| `--head` | string | `HEAD` | Head ref for diff comparison |
| `--json` | bool | `false` | Output results as JSON instead of human-readable format |
| `--verbose` | bool | `false` | Show detailed AI request/response logs (URL, body, timing, status) |
| `--no-cache` | bool | `false` | Skip cache and re-review all files |
| `--format` | string | `human` | Output format: `human`, `json`, `github` |
| `--github-pr` | string | | Post results as PR comment (format: `owner/repo#number`) |
| `--fix` | bool | `false` | Generate code fix suggestions for issues |
| `--patch` | string | | Write fixes as unified diff to file (requires `--fix`) |
| `--apply` | bool | `false` | Apply fixes interactively with y/n prompt per fix (requires `--fix`) |
| `--yes` | bool | `false` | Apply all fixes without prompting (requires `--fix --apply`) |

### Scan (Full Codebase Audit)

Scan all Laravel files in the project — not just changed ones. Useful for:
- **Initial adoption** — audit an existing codebase when first adopting larasense-limbo
- **Periodic audits** — run scheduled full scans to catch accumulated technical debt
- **Pre-release checks** — full audit before major deployments

```bash
# Scan current directory
larasense-limbo scan

# Scan a specific Laravel project
larasense-limbo scan --path /path/to/laravel-project

# Scan specific file(s) — saves tokens, targeted review
larasense-limbo scan --file app/Http/Controllers/UserController.php
larasense-limbo scan --file app/Models/User.php --file routes/web.php

# Scan only modified/staged files from git status (pre-commit workflow)
larasense-limbo scan --modified
larasense-limbo scan --modified --fix --apply

# Combine with fix
larasense-limbo scan --file app/Traits/Auditable.php --fix --apply

# Output as JSON
larasense-limbo scan --json

# Verbose mode for debugging
larasense-limbo scan --verbose

# Force re-scan (skip cache)
larasense-limbo scan --no-cache
```

#### Scan Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--path` | string | `.` | Root directory to scan |
| `--file` | strings | | Scan specific file(s) instead of directory (repeatable, skips include/exclude filters) |
| `--modified` | bool | `false` | Scan only modified/staged files from `git status` (pre-commit workflow) |
| `--json` | bool | `false` | Output results as JSON |
| `--format` | string | `human` | Output format: `human`, `json`, `github` |
| `--verbose` | bool | `false` | Show detailed AI request/response logs |
| `--no-cache` | bool | `false` | Skip cache and re-scan all files |
| `--fix` | bool | `false` | Generate code fix suggestions for issues |
| `--patch` | string | | Write fixes as unified diff to file (requires `--fix`) |
| `--apply` | bool | `false` | Apply fixes interactively with y/n prompt per fix (requires `--fix`) |
| `--yes` | bool | `false` | Apply all fixes without prompting (requires `--fix --apply`) |

The scan command automatically skips `vendor/`, `node_modules/`, `.git/`, and `storage/` directories. Files are filtered using the same `include`/`exclude` glob patterns from your config.

For large projects, files are automatically split into token-aware batches (~20K tokens each) to stay within AI provider limits.

### Auto-fix Suggestions (`--fix`)

Generate code fix suggestions alongside issue detection. Works with both `analyze` and `scan`:

```bash
# Show issues with before/after fix code in terminal
larasense-limbo analyze --fix
larasense-limbo scan --fix
larasense-limbo scan --path app/Http/Controllers --fix

# Preview what would be applied (no file changes)
larasense-limbo scan --fix --dry-run

# Generate a patch file (review before applying)
larasense-limbo scan --fix --patch fixes.patch
git apply fixes.patch

# Apply fixes interactively (y/n per fix with colored diff preview)
larasense-limbo scan --fix --apply
larasense-limbo analyze --base main --fix --apply

# Apply ALL fixes without prompting (CI-friendly)
larasense-limbo scan --fix --apply --yes

# Apply on a separate git branch (easy revert)
larasense-limbo scan --fix --apply --yes --git-branch fix/ai-review
larasense-limbo scan --fix --apply --yes --git-branch auto  # auto-generates branch name

# Undo all applied fixes (restore from backup)
larasense-limbo undo
```

#### Fix Flags (available on both `analyze` and `scan`)

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--fix` | bool | `false` | Generate code fix suggestions for high and medium severity issues |
| `--dry-run` | bool | `false` | Show what fixes would be applied without writing to files (requires `--fix`) |
| `--patch` | string | | Write fixes as unified diff to file (requires `--fix`) |
| `--apply` | bool | `false` | Apply fixes interactively — prompts y/n per fix with colored diff preview (requires `--fix`) |
| `--yes` | bool | `false` | Apply all fixes without prompting (requires `--fix --apply`) |
| `--git-branch` | string | | Create a git branch before applying fixes for easy revert (requires `--fix --apply`) |

#### How It Works

1. When `--fix` is used, the AI prompt is enhanced to request before/after code blocks
2. Fixes are only generated for **high** and **medium** severity issues (saves tokens)
3. Without `--fix`, the pipeline is unchanged — zero extra token consumption
4. Multiple fixes in the same file are validated against the **original** file content, then applied **bottom-up** in a single pass — no line-shift conflicts
5. `--apply` shows each fix with colored before/after diff and prompts `y/n/q` (quit)
6. `--apply --yes` skips prompts and applies all fixes (for CI or batch operations)
7. Before-matching uses 6 strategies (exact → trimmed → normalized whitespace → stripped indentation → contains → full-file search) + nearby search ±15 lines
8. If AI response is truncated (`finish_reason: "length"`), auto-retries with 2x `max_tokens`
9. Skipped fixes are logged with reason (mismatch, user rejected, invalid range, overlapping)
10. A summary table is shown at the end with total fixable, applied, skipped + breakdown by skip reason

#### Output with `--fix`

```
  🔴 [HIGH] #1: Mass Assignment Vulnerability
     Line: 15
     Using $request->all() passes unvalidated data directly to create()
     💡 Suggestion: Use $request->validated() with a Form Request
     ┌─ Before:
     │ - $user = User::create($request->all());
     ├─ After:
     │ + $user = User::create($request->validated());
     └─
```

#### Safety

- **Interactive by default**: `--apply` prompts y/n per fix with colored diff preview. Press `q` to abort remaining fixes.
- **Dry-run preview**: `--dry-run` shows all fixes that would be applied without modifying any files.
- **6-strategy fuzzy matching**: Handles AI line number offsets (±15 lines), whitespace differences, indentation mismatches, and full-file content search.
- **Batch apply per file**: Multiple fixes in the same file are validated against the original content, then applied bottom-up in a single pass — eliminates line-shift conflicts between fixes.
- **Overlapping detection**: If two fixes target the same lines, the second is skipped to prevent file corruption.
- **Auto-retry on truncation**: If AI response is cut off (`finish_reason: "length"`), automatically retries with 2x `max_tokens` (cap at 65536).
- **Undo support**: `larasense-limbo undo` restores all files to their pre-fix state from automatic backups.
- **Git-aware apply**: `--git-branch` creates a separate branch before applying, making revert as simple as `git checkout <original-branch>`.
- **`--yes` requires `--apply`**: You can't skip prompting without explicitly opting in.
- **Patch review**: Use `--patch` to generate a diff file you can review before applying with `git apply`.

#### Undo Command

If applied fixes cause issues, revert all changes:

```bash
larasense-limbo undo
```

This restores files from the backup created during `--apply` and removes the backup directory. If you used `--git-branch`, you can also simply `git checkout <original-branch>`.

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

### Scan

Full codebase audit — see [Scan (Full Codebase Audit)](#scan-full-codebase-audit) above for details.

### Undo

Revert all applied fixes to their original state:

```bash
larasense-limbo undo
```

Restores files from the backup created during `--fix --apply`. The backup is removed after a successful undo.

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
# Linux/macOS
go build -ldflags "\
  -X github.com/Mattel-Limbo/larasense-limbo/cmd.Version=1.0.0 \
  -X github.com/Mattel-Limbo/larasense-limbo/cmd.Commit=$(git rev-parse --short HEAD) \
  -X github.com/Mattel-Limbo/larasense-limbo/cmd.BuildDate=$(date -u +%Y-%m-%d)" \
  -o larasense-limbo .

# Windows (PowerShell)
go build -ldflags "-X github.com/Mattel-Limbo/larasense-limbo/cmd.Version=1.0.0 -X github.com/Mattel-Limbo/larasense-limbo/cmd.Commit=$(git rev-parse --short HEAD) -X github.com/Mattel-Limbo/larasense-limbo/cmd.BuildDate=$(Get-Date -Format 'yyyy-MM-dd')" -o larasense-limbo.exe .
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

### Recommended Models

Not all models perform equally for code review. Here are tested recommendations ranked by **consistency and accuracy**:

#### Tier 1 — Best (Recommended for Production)

| Provider | Model | Why | Cost |
|----------|-------|-----|------|
| OpenAI | `gpt-4.1` | Best JSON compliance, follows structured prompts precisely, very consistent results across runs | Medium |
| Anthropic | `claude-sonnet-4-20250514` | Excellent code understanding, strong at detecting security issues, reliable JSON output | Medium |
| Google | `gemini-2.5-flash` | Fast, cheap, good JSON compliance, great for high-volume CI/CD pipelines | Low |

#### Tier 2 — Good (Suitable for Development)

| Provider | Model | Why | Cost |
|----------|-------|-----|------|
| OpenAI | `gpt-4o` | Good balance of speed and quality, slightly less consistent than gpt-4.1 | Medium |
| OpenAI | `gpt-4o-mini` | Budget-friendly, acceptable quality for non-critical reviews | Low |
| Ollama | `llama3.1` / `codellama` | Free, runs locally, but less consistent JSON output — may need `--no-cache` more often | Free |

#### Tier 3 — Use with Caution

| Provider | Model | Why | Cost |
|----------|-------|-----|------|
| Any | `claude-opus-*` | Very thorough but tends to ignore JSON-only instructions and return Markdown instead, causing higher token usage | High |
| Any | Small models (<7B) | Inconsistent severity ratings, often miss issues or hallucinate false positives | Free/Low |

> **Tip:** When using OpenRouter, prefix the model name with the provider — e.g., `openai/gpt-4.1`, `anthropic/claude-sonnet-4-20250514`, `google/gemini-2.5-flash`.

### Best Practice Configuration

The default configuration works well for most projects. Below are optimized configurations for specific use cases.

#### Production CI/CD (Maximum Consistency)

Use this when you need **identical results across runs** — critical for CI gates and automated PR reviews:

```yaml
provider:
  base_url: http://localhost:1430
  api_key: ${AI_API_KEY}
  model: claude-sonnet-4.5
  max_tokens: 1024        # Cap output to prevent verbose responses
  temperature: 0.0         # Fully deterministic — zero randomness
  seed: 42                 # Fixed seed for reproducible results

review:
  max_issues: 10
  severity_threshold: medium
  context_lines: 10

filters:
  include:
    - "app/**"
    - "routes/**"
    - "resources/views/**"
    - "config/**"
    - "database/migrations/**"
  exclude:
    - "tests/**"
    - "database/seeders/**"
    - "database/factories/**"
```

**Why this works:**
- `temperature: 0.0` eliminates sampling randomness — the model always picks the most probable token
- `seed: 42` ensures the same random state across requests (supported by OpenAI, some other providers)
- `max_tokens: 1024` prevents the AI from writing overly verbose descriptions that waste tokens
- `gpt-4.1` has the best JSON compliance among tested models

#### Budget-Friendly (High Volume)

For teams running reviews on every commit or large monorepos:

```yaml
provider:
  name: gemini
  api_key: ${GEMINI_API_KEY}
  model: gemini-2.5-flash
  max_tokens: 768
  temperature: 0.0
  seed: 42

review:
  max_issues: 5
  severity_threshold: high    # Only report critical issues
  context_lines: 5            # Less context = fewer prompt tokens
```

#### Local Development (Free, No API Key)

For offline development or when you don't want to use API credits:

```yaml
provider:
  name: ollama
  model: llama3.1             # or codellama, deepseek-coder
  max_tokens: 1024
  temperature: 0.0
  seed: 42

review:
  max_issues: 10
  severity_threshold: low     # Show everything since it's free
  context_lines: 10
```

> **Note:** Run `ollama pull llama3.1` first to download the model.

#### Security-Focused Audit

For pre-release security audits or compliance checks:

```yaml
provider:
  name: openai
  api_key: ${AI_API_KEY}
  model: gpt-4.1
  max_tokens: 1500
  temperature: 0.0
  seed: 42

review:
  max_issues: 20
  severity_threshold: low
  custom_prompt: "Focus exclusively on security vulnerabilities: SQL injection, XSS, CSRF, mass assignment, hardcoded secrets, missing authentication, and insecure file uploads. Ignore performance and convention issues."
  context_lines: 15

filters:
  include:
    - "app/**"
    - "routes/**"
    - "resources/views/**"
    - "config/**"
  exclude:
    - "tests/**"
```

#### Configuration Parameters Reference

| Parameter | Default | Recommended | Description |
|-----------|---------|-------------|-------------|
| `temperature` | `0.0` | `0.0` | Set to `0.0` for consistent results. Values above `0.3` introduce noticeable randomness in issue detection and severity. |
| `seed` | `42` | `42` | Any fixed integer ensures reproducibility. Set to `0` for random behavior. Not all providers support this. |
| `max_tokens` | `1024` | `768-1500` | Controls maximum AI response length. Too low may truncate results; too high wastes tokens on verbose output. Auto-scales down for large prompts. |
| `context_lines` | `10` | `5-15` | Lines of surrounding code sent with each diff. More context = better analysis but higher token cost. Use `5` for budget, `15` for thorough reviews. |
| `max_issues` | `5` | `5-10` | Limits output count. Set higher for full audits, lower for CI gates. |
| `severity_threshold` | `medium` | `medium` | Use `high` in CI to only fail on critical issues. Use `low` for thorough local reviews. |

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

Larasense-limbo caches review results per file using SHA-256 hashes. On subsequent runs, unchanged files are skipped — saving API calls and time. Cache works for both `analyze` and `scan` commands.

```bash
# Normal run (uses cache)
larasense-limbo analyze --base main
larasense-limbo scan

# Force re-review/re-scan all files
larasense-limbo analyze --base main --no-cache
larasense-limbo scan --no-cache
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
│   ├── scan.go                      # scan subcommand (full codebase audit)
│   ├── fix.go                       # Shared fix output handler (--fix, --apply, --dry-run)
│   ├── undo.go                      # undo subcommand (revert applied fixes)
│   ├── init.go                      # init subcommand (config generator)
│   └── version.go                   # version subcommand (build info)
├── internal/
│   ├── config/config.go             # Viper YAML + env config loader
│   ├── git/git.go                   # Git operations (diff, show, branch)
│   ├── diff/parser.go               # Git diff parser
│   ├── context/builder.go           # Laravel-aware context builder
│   ├── scanner/scanner.go           # Filesystem walker for full codebase scan
│   ├── ai/client.go                 # AI provider HTTP client + response parsers
│   ├── fixer/fixer.go               # Auto-fix engine (apply, patch, undo, backup)
│   ├── reviewer/
│   │   ├── reviewer.go              # Pipeline orchestrator (diff + scan + fix)
│   │   └── batcher.go               # Token-aware file batching
│   ├── cache/cache.go               # SHA-256 file cache with issue + fix storage
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

**Analyze (diff-based):**

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
  │                                         │
  └───────────────────────────────────────── ▼ ── shared pipeline ──
                                      reviewer.review()
                                        ├─ filterCached()          Skip unchanged files
                                        ├─ batchFiles()            Split into token-aware batches
                                        ├─ analyzeInBatches()      Send each batch to AI
                                        ├─ filterIssues()          Severity threshold + max count
                                        └─ output.Format*()        Human / JSON / GitHub
```

**Scan (full codebase):**

```
larasense-limbo scan
  │
  ├─ config.Load()                    Load .larasense-limbo.yml + env vars
  │
  ├─ scanner.Scan(rootDir)            Walk filesystem, read all matching files
  │   ├─ ShouldIncludeFile()          Same glob filters as analyze
  │   ├─ ClassifyFile()               Same 18 Laravel types
  │   └─ skip vendor/node_modules     Auto-skip irrelevant directories
  │                                         │
  └───────────────────────────────────────── ▼ ── shared pipeline ──
                                      reviewer.review()
                                        ├─ filterCached()
                                        ├─ batchFiles()
                                        ├─ analyzeInBatches()
                                        ├─ filterIssues()
                                        └─ output.Format*()
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
# Linux/macOS
go build -o larasense-limbo .

# Windows
go build -o larasense-limbo.exe .

# Tests (all platforms)
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
