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
| 24 | Git module tests | 📋 Planned | Mock git commands |
| 25 | Reviewer integration tests | 📋 Planned | End-to-end pipeline test |
| 26 | Output formatter tests | ✅ Done | FormatJSON, FormatHuman, groupByFile, severityIcon, empty/no-issues cases |
| 27 | Config loader tests | ✅ Done | DefaultConfig, Load (valid/missing/defaults), env var expansion, validation |
| 28 | CLI command tests | ✅ Done | version output, init create/refuse/force, analyze flags, root subcommands |

## CI/CD

| # | Fitur | Status | Catatan |
|---|-------|--------|---------|
| 29 | GitHub Actions workflow | ✅ Done | CI: test + build + lint on push/PR, matrix: 3 OS × 3 Go versions |
| 30 | Release workflow | 📋 Planned | GoReleaser, multi-platform binaries |
| 31 | Dockerfile | 📋 Planned | Container image untuk CI |
| 32 | Makefile | ✅ Done | `make build`, `make test`, `make vet`, `make lint`, `make install`, `make clean`, `make run` |

## Planned Features

| # | Fitur | Status | Catatan |
|---|-------|--------|---------|
| 33 | Multiple AI provider support | 📋 Planned | Anthropic, Gemini, Ollama endpoint configs |
| 34 | Custom prompt override | ✅ Done | `review.custom_prompt` di config, di-append ke system prompt |
| 35 | GitHub PR comment integration | 📋 Planned | Post review sebagai PR comment via GitHub API |
| 36 | Inline annotation output | 📋 Planned | Format output untuk GitHub Actions annotations |
| 37 | Cache layer | 📋 Planned | Skip re-review file yang tidak berubah |
| 38 | Verbose/debug mode | ✅ Done | `--verbose` flag, log request/response body, timing, parsed issue count |
| 39 | Config init command | ✅ Done | `larasense-limbo init` + `--force` flag, generate config template |
| 40 | Version command | ✅ Done | `larasense-limbo version`, build info via `-ldflags` |

## Future Ideas

| # | Fitur | Status | Catatan |
|---|-------|--------|---------|
| 41 | Auto-fix suggestions | 💡 Future | Generate patch files dari AI suggestions |
| 42 | Multi-language support | 💡 Future | Vue/JS files di Laravel project |
| 43 | Rule customization | 💡 Future | Enable/disable specific rule categories |
| 44 | Baseline support | 💡 Future | Ignore existing issues, hanya report baru |
| 45 | SARIF output | 💡 Future | Standard format untuk security tools |
| 46 | VS Code extension | 💡 Future | Real-time review di editor |
| 47 | Pre-commit hook | 💡 Future | Review otomatis sebelum commit |
| 48 | Team config sharing | 💡 Future | Shared config via package registry |
| 49 | Review history/analytics | 💡 Future | Track issue trends over time |
| 50 | Plugin system | 💡 Future | Custom analyzers via Go plugins |

---

## Summary

| Kategori | Done | Planned | Future |
|----------|------|---------|--------|
| Core (MVP) | 15 | 0 | 0 |
| Reliability | 5 | 0 | 0 |
| Testing | 6 | 2 | 0 |
| CI/CD | 2 | 2 | 0 |
| Planned Features | 4 | 4 | 0 |
| Future Ideas | 0 | 0 | 10 |
| **Total** | **32** | **8** | **10** |
