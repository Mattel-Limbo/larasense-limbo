# Progress Tracker

Status tracking untuk fitur-fitur larasense-limbo.

> **Legend:** ✅ Done | 🚧 In Progress | 📋 Planned | 💡 Future Idea

---

## Core (MVP)

| # | Fitur | Status | Catatan |
|---|-------|--------|---------|
| 1 | CLI command `analyze` | ✅ Done | Cobra, flags: `--base`, `--head`, `--json` |
| 2 | Config file `.larasense-limbo.yml` | ✅ Done | Viper, YAML + env vars, `${VAR}` expansion |
| 3 | Git diff extraction | ✅ Done | `git diff base...head` via `os/exec` |
| 4 | Diff parser | ✅ Done | Parse files, hunks, added/removed/context lines |
| 5 | Laravel file filtering | ✅ Done | `.php` + `.blade.php`, glob include/exclude |
| 6 | Laravel file classification | ✅ Done | 18 component types (Controller, Model, dll.) |
| 7 | Context builder | ✅ Done | Hints per tipe file, surrounding ±20 lines |
| 8 | AI provider client | ✅ Done | HTTP POST ke `/v1/chat/completions` atau `/v1/responses`, Bearer auth |
| 9 | AI prompt (Laravel reviewer) | ✅ Done | Strict JSON output, fokus performance/security/practices/conventions |
| 10 | Response parser (multi-format) | ✅ Done | OpenAI Responses API, Chat Completions, direct JSON, markdown fence stripping |
| 11 | Human-readable output | ✅ Done | Grouped by file, severity icons, suggestions |
| 12 | JSON output | ✅ Done | `--json` flag, structured output |
| 13 | Severity filtering | ✅ Done | `severity_threshold` config (low/medium/high) |
| 14 | Max issues limit | ✅ Done | `max_issues` config |
| 15 | Exit code 1 on high severity | ✅ Done | CI gate compatible |

## Reliability

| # | Fitur | Status | Catatan |
|---|-------|--------|---------|
| 16 | Retry with exponential backoff | ✅ Done | 3 retries, 2s → 4s delay |
| 17 | Request timeout | ✅ Done | 120s default |
| 18 | Graceful error on missing context | ✅ Done | Non-fatal jika `git show` gagal |
| 19 | Config validation | ✅ Done | Required: `base_url`, `api_key` |
| 20 | Environment variable support | ✅ Done | Prefix `LARASENSE_LIMBO_`, auto env binding |

## Testing

| # | Fitur | Status | Catatan |
|---|-------|--------|---------|
| 21 | Diff parser tests | ✅ Done | Parse, empty, hunk header, changed lines |
| 22 | Context builder tests | ✅ Done | classifyFile, isLaravelFile, matchGlob, generateHint |
| 23 | AI response parser tests | ✅ Done | Direct JSON, markdown wrapped, surrounding text, Chat Completions, Responses API |
| 24 | Git module tests | ✅ Done | Real temp git repo: GetDiff, GetFileContent, GetSurroundingLines, error cases, multi-file diff |
| 25 | Reviewer integration tests | ✅ Done | filterIssues (threshold + max + combined), meetsThreshold (case-insensitive + unknown), buildSummary, New constructor |
| 26 | Output formatter tests | ✅ Done | FormatJSON, FormatHuman, groupByFile, severityIcon, empty/no-issues cases |
| 27 | Config loader tests | ✅ Done | DefaultConfig, Load (valid/missing/defaults), env var expansion, validation |
| 28 | CLI command tests | ✅ Done | version output, init create/refuse/force, analyze flags, root subcommands |

## CI/CD

| # | Fitur | Status | Catatan |
|---|-------|--------|---------|
| 29 | GitHub Actions workflow | ✅ Done | CI: test + build + lint on push/PR, matrix: 3 OS × 3 Go versions |
| 30 | Release workflow | ✅ Done | GoReleaser, multi-platform binaries (linux/darwin/windows × amd64/arm64), trigger on tag `v*` |
| 31 | Dockerfile | ✅ Done | Multi-stage build (golang:1.23-alpine → alpine:3.20), includes git |
| 32 | Makefile | ✅ Done | `make build`, `make test`, `make vet`, `make lint`, `make install`, `make clean`, `make run` |

## Planned Features

| # | Fitur | Status | Catatan |
|---|-------|--------|---------|
| 33 | Multiple AI provider support | ✅ Done | Provider presets: openai, anthropic, gemini, ollama, openrouter. `provider.name` auto-fills base_url/model/endpoint |
| 34 | Custom prompt override | ✅ Done | `review.custom_prompt` di config, di-append ke system prompt |
| 35 | GitHub PR comment integration | ✅ Done | `--github-pr owner/repo#number`, markdown comment via GitHub API |
| 36 | Inline annotation output | ✅ Done | `--format github`, high→`::error`, medium→`::warning`, low→`::notice` |
| 37 | Cache layer | ✅ Done | SHA-256 hash per file, `.larasense-limbo-cache.json`, `--no-cache` flag |
| 38 | Verbose/debug mode | ✅ Done | `--verbose` flag, log request/response body, timing, parsed issue count |
| 39 | Config init command | ✅ Done | `larasense-limbo init` + `--force` flag, generate config template |
| 40 | Version command | ✅ Done | `larasense-limbo version`, build info via `-ldflags` |

## New Features

| # | Fitur | Status | Catatan |
|---|-------|--------|---------|
| 41 | Full codebase scan command | ✅ Done | `larasense-limbo scan`, filesystem walker, dedicated AI prompt, token-aware batching |
| 42 | Scanner module | ✅ Done | `internal/scanner/scanner.go`, walks filesystem, skips vendor/node_modules/storage/.git |
| 43 | Token-aware batching | ✅ Done | `internal/reviewer/batcher.go`, ~80KB per batch (~20K tokens), oversized files get own batch |
| 44 | Scan-specific AI prompt | ✅ Done | `BuildScanPrompt()`, deterministic pattern-matching checklist, fixed severity levels |
| 45 | Shared review pipeline | ✅ Done | Refactored `reviewer.go`: `review()` shared by `Run()` and `RunScan()`, `analyzeInBatches()` |
| 46 | Deterministic AI config | ✅ Done | `temperature: 0.0`, `seed: 42`, `max_tokens: 1024` defaults for consistent results |
| 47 | Pattern-matching prompts | ✅ Done | Prompts redesigned as deterministic scanner with fixed severity checklist, reduces hallucinated issues |
| 48 | Markdown fallback parser | ✅ Done | `parseMarkdownFallback()` extracts issues from Markdown when AI ignores JSON instruction |
| 49 | Robust JSON parser | ✅ Done | 5-strategy parser: direct JSON → code fence → `{"issues"` pattern → brace matching → Markdown fallback |
| 50 | Subdirectory scan support | ✅ Done | `--path app/Http/Controllers` works correctly, paths relative to project root |
| 51 | Recommended models docs | ✅ Done | 3-tier model ranking, best practice configs (production/budget/local/security), parameter reference |
| 52 | Auto-fix suggestions | ✅ Done | `--fix` flag, before/after code blocks, `--patch` for unified diff, `--apply` for direct edit |
| 53 | Fix struct in Issue | ✅ Done | `*Fix` field (start_line, end_line, before, after), nil when --fix not used, backward compatible |
| 54 | Fixer module | ✅ Done | `internal/fixer/fixer.go`: GeneratePatch, WritePatch, ApplyFixes with before-validation safety |
| 55 | Fix-aware AI prompts | ✅ Done | `BuildDiffFixPrompt()`, `BuildScanFixPrompt()` — only generate fixes for high+medium severity |
| 56 | Fix cache integration | ✅ Done | CachedFix struct, cache invalidation when fixMode on but cached data lacks fixes |

## AI Fixer Enhancements

| # | Fitur | Status | Catatan |
|---|-------|--------|---------|
| 57 | Interactive fix mode | ✅ Done | `--apply` interactive y/n/q per fix, `--yes` apply all, colored diff preview |
| 58 | 6-strategy fuzzy matching | ✅ Done | Exact → trimmed → normalized whitespace → stripped indentation → contains → full-file search |
| 59 | Nearby search ±15 lines | ✅ Done | Handle AI line number offset, `searchNearby()` + `searchByContent()` |
| 60 | Markdown fallback parser | ✅ Done | Parse `###`/`####` headers, severity markers, code blocks, file hints inference |
| 61 | Truncated JSON repair | ✅ Done | `repairTruncatedJSON()` — find last complete issue, close brackets |
| 62 | Auto-retry on truncation | ✅ Done | Detect `finish_reason: "length"`, retry with 2x `max_tokens` (cap 65536) |
| 63 | Dry-run mode | ✅ Done | `--fix --dry-run` — show what would be applied without writing files |
| 64 | Undo/rollback | ✅ Done | Auto-backup before apply, `larasense-limbo undo` command, cleanup after restore |
| 65 | Git-aware apply | ✅ Done | `--git-branch name` or `auto`, creates branch before apply, shows revert instructions |
| 66 | Batch fix summary | ✅ Done | Summary table: total fixable, applied, skipped + breakdown by skip reason |
| 67 | Batch apply per file | ✅ Done | 3-phase: validate all fixes against original → overlap check → apply bottom-up in single pass |
| 68 | Overlapping fix detection | ✅ Done | `overlapsApplied()` — detect & skip with clear message |
| 69 | EndLine clamping | ✅ Done | AI sometimes overshoots file length, clamped to actual line count |
| 70 | `max_tokens` default 16384 | ✅ Done | Prevents truncation for most responses, user-configurable |

## Planned Ideas

| # | Fitur | Status | Catatan |
|---|-------|--------|---------|
| 71 | Scan spesifik file (`--file`) | ✅ Done | `scan --file path/to/file.php`, repeatable, skips filters, hemat token |
| 72 | Scan modified files (`--modified`) | ✅ Done | `scan --modified`, reads `git status --porcelain`, skips deleted, handles renames |
| 73 | Logic/Semantic error detection | ✅ Done | HIGH: undefined variable, wrong variable in loop. MEDIUM: unreachable code, wrong comparison, type mismatch. LOW: dead code |
| 74 | Flag aliases/simplification | ✅ Done | `--auto` (fix+apply+yes), `--preview` (fix+dry-run), `--fresh` (no-cache), positional args auto-detect file/dir |
| 75 | Terminal UI improvement | ✅ Done | Progress bar in-place, header box (provider/target/mode), duration tracking, better skip messages, `internal/ui` package |

## Detection Coverage Expansion

| # | Pattern | Severity | Status | Catatan |
|---|---------|----------|--------|---------|
| 76 | Insecure file upload | HIGH | ✅ Done | No MIME/extension validation on uploaded files |
| 77 | CSRF missing | HIGH | ✅ Done | POST/PUT/DELETE form without @csrf or token verification |
| 78 | Open redirect | HIGH | ✅ Done | `redirect($userInput)` without URL whitelist |
| 79 | Debug left in code | HIGH | ✅ Done | `dd()`, `dump()`, `ray()` calls left in production code |
| 80 | Exposed env() in code | HIGH | ✅ Done | Direct `env()` calls outside config files (breaks config cache) |
| 81 | Missing authorization | MEDIUM | ✅ Done | Controller action without `authorize()`, Gate, or Policy check |
| 82 | Missing DB transaction | MEDIUM | ✅ Done | Multiple related DB writes without `DB::transaction()` |
| 83 | Queue without retry config | MEDIUM | ✅ Done | Job class without `$tries`, `$timeout`, or `$backoff` |
| 84 | Raw DB with user input | MEDIUM | ✅ Done | `DB::raw()` or `whereRaw()` concatenating user input |
| 85 | Hardcoded env() in code | MEDIUM | ✅ Done | `env()` used outside `config/` files (breaks `config:cache`) |
| 86 | Magic numbers/strings | LOW | ✅ Done | Hardcoded numeric/string values instead of constants or enums |
| 87 | God model | LOW | ✅ Done | Model with >20 relationships or >500 lines |
| 88 | Missing return type | LOW | ✅ Done | Public method without return type declaration (PHP 8+) |
| 89 | Deprecated API usage | LOW | ✅ Done | Using deprecated Laravel/PHP functions |

## Distribution & Packaging

| # | Fitur | Status | Catatan |
|---|-------|--------|---------|
| 90 | NPM package distribution | ✅ Done | `npm install -g larasense-limbo`, auto-download binary per OS/arch, postinstall script, bin wrapper |
| 91 | Install script (curl) | ✅ Done | `curl -fsSL .../install.sh \| sh`, auto-detect OS/arch, VERSION/INSTALL_DIR env vars, sudo fallback, PATH check |
| 92 | GitHub Action | ✅ Done | `uses: Mattel-Limbo/larasense-limbo/action@main`, composite action, cache via actions/cache, auto-detect OS/arch, version input |

## Future Ideas

| # | Fitur | Status | Catatan |
|---|-------|--------|---------|
| 93 | Fix confidence score | 💡 Future | AI rate confidence per fix, skip low-confidence |
| 94 | Multi-pass fix | 💡 Future | Re-run AI setelah fix applied untuk generate fix yang akurat terhadap state baru |
| 95 | Fix dependency graph | 💡 Future | Detect fix A depends on fix B, apply in correct order |
| 96 | Semantic diff display | 💡 Future | Syntax-highlighted diff di terminal |
| 97 | Fix templates | 💡 Future | Pre-defined fix patterns tanpa AI (e.g., `Model::all()` → `Model::cursor()`) |
| 98 | Multi-language support | 💡 Future | Vue/JS files di Laravel project |
| 99 | Rule customization | 💡 Future | Enable/disable specific rule categories |
| 100 | Baseline support | 💡 Future | Ignore existing issues, hanya report baru |
| 101 | SARIF output | 💡 Future | Standard format untuk security tools |
| 102 | VS Code extension | 💡 Future | Real-time review di editor |
| 103 | Pre-commit hook | 💡 Future | Review otomatis sebelum commit |
| 104 | Team config sharing | 💡 Future | Shared config via package registry |
| 105 | Review history/analytics | 💡 Future | Track issue trends over time |
| 106 | Plugin system | 💡 Future | Custom analyzers via Go plugins |

---

## Summary

| Kategori | Done | Planned | Future |
|----------|------|---------|--------|
| Core (MVP) | 15 | 0 | 0 |
| Reliability | 5 | 0 | 0 |
| Testing | 8 | 0 | 0 |
| CI/CD | 4 | 0 | 0 |
| Planned Features | 8 | 0 | 0 |
| New Features | 16 | 0 | 0 |
| AI Fixer Enhancements | 14 | 0 | 0 |
| Planned Ideas | 5 | 0 | 0 |
| Detection Coverage | 14 | 0 | 0 |
| Distribution & Packaging | 3 | 0 | 0 |
| Future Ideas | 0 | 0 | 14 |
| **Total** | **92** | **0** | **14** |
