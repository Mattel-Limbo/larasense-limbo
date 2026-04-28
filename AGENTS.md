# PROJECT KNOWLEDGE BASE

**Generated:** 2025-04-28
**Module:** `github.com/larasense/larasense-limbo`

## OVERVIEW

Go CLI tool that analyzes git diffs in Laravel projects via AI provider API, reporting performance, security, and convention issues. Stack: Cobra CLI + Viper config + `os/exec` git + `net/http` AI client.

## STRUCTURE

```
larasense-limbo/
├── main.go                          # Entry → cmd.Execute()
├── cmd/
│   ├── root.go                      # Root cobra command, no logic
│   └── analyze.go                   # `analyze` subcommand, flags, orchestration call
├── internal/
│   ├── config/config.go             # Viper YAML + env loader, ${VAR} expansion
│   ├── git/git.go                   # Shell-out to `git diff`, `git show`
│   ├── diff/parser.go               # Raw diff → []FileDiff with hunks/lines
│   ├── context/builder.go           # Laravel-aware file classifier (18 types), glob filter, AI hint generator
│   ├── ai/client.go                 # HTTP POST to /v1/responses, retry, multi-format response parser
│   ├── reviewer/reviewer.go         # Pipeline orchestrator: git→parse→context→AI→filter→result
│   └── output/formatter.go          # Human-readable + JSON output
└── .larasense-limbo.yml              # Example config (tracked)
```

## WHERE TO LOOK

| Task | Location | Notes |
|------|----------|-------|
| Add new CLI command | `cmd/` | Create file, register via `rootCmd.AddCommand()` in `init()` |
| Change AI prompt | `internal/ai/client.go:buildPrompt()` | System prompt for Laravel review |
| Add new Laravel file type | `internal/context/builder.go:classifyFile()` + `generateHint()` | Both must be updated together |
| Change file filtering | `internal/context/builder.go:shouldInclude()` + `isLaravelFile()` | `isLaravelFile` is hardcoded allowlist, `shouldInclude` uses config globs |
| Modify AI request/response format | `internal/ai/client.go:doRequest()` + `extractContent()` | Handles OpenAI Responses API, Chat Completions, and direct JSON |
| Change config schema | `internal/config/config.go` | Struct tags = YAML keys; env prefix `LARASENSE_LIMBO_` |
| Adjust severity filtering | `internal/reviewer/reviewer.go:filterIssues()` + `meetsThreshold()` | low=1, medium=2, high=3 |

## CODE MAP

| Symbol | Type | Location | Refs | Role |
|--------|------|----------|------|------|
| `ai.Issue` | struct | `ai/client.go:56` | 7 | Core data type — re-exported as `reviewer.Issue` via type alias |
| `config.Config` | struct | `config/config.go:12` | 10 | Central config, consumed by context.Builder + reviewer.Reviewer |
| `reviewer.Result` | struct | `reviewer/reviewer.go:19` | 8 | Final output consumed by output.FormatJSON/FormatHuman |
| `diff.FileDiff` | struct | `diff/parser.go:10` | 9 | Parsed diff per file, consumed by context.Builder |
| `context.ReviewContext` | struct | `context/builder.go:24` | 5 | AI-ready context with classified files |
| `ai.Client` | struct | `ai/client.go:22` | 5 | HTTP client with retry, created in reviewer.Run() |
| `context.Builder` | struct | `context/builder.go:29` | 5 | Filters + classifies files, created in reviewer.Run() |
| `reviewer.Reviewer` | struct | `reviewer/reviewer.go:26` | 4 | Pipeline orchestrator, created in cmd/analyze.go |

## DATA FLOW

```
cmd/analyze.go:runAnalyze()
  → config.Load()                          # Viper: YAML + env
  → reviewer.New(cfg).Run(base, head)
      → git.GetDiff(base, head)            # exec: git diff base...head
      → diff.Parse(rawDiff)                # → []FileDiff
      → context.NewBuilder(cfg, head).Build(files)
          → shouldInclude() per file       # glob + isLaravelFile filter
          → classifyFile() + generateHint()# 18 Laravel types
          → git.GetSurroundingLines()      # ±20 lines context
      → ai.NewClient(&cfg.Provider).Analyze(diff, context)
          → HTTP POST /v1/responses        # 3 retries, 120s timeout
          → extractContent()               # Responses API / Chat / direct JSON
          → parseIssuesFromText()          # strips markdown fences
      → filterIssues()                     # severity threshold + max count
  → output.FormatHuman() or FormatJSON()
  → os.Exit(1) if any high-severity issue
```

## CONVENTIONS

- **Error wrapping**: Always `fmt.Errorf("context: %w", err)` — never bare returns
- **Constructors**: `New*()` pattern returning pointer (`NewClient`, `NewBuilder`, `New`)
- **Type re-export**: `reviewer.Issue = ai.Issue` (type alias) so consumers don't import `ai` directly
- **Non-fatal errors**: Surrounding context fetch failure is silently skipped (intentional — see `builder.go:68`)
- **Config env vars**: Prefix `LARASENSE_LIMBO_`, dot→underscore (`provider.api_key` → `LARASENSE_LIMBO_PROVIDER_API_KEY`)
- **Config `${VAR}` expansion**: `api_key: ${AI_API_KEY}` expanded via `os.Expand()` in `config.go`

## ANTI-PATTERNS (THIS PROJECT)

- **DO NOT** add heavy SDK dependencies — use `net/http` for AI provider
- **DO NOT** scan entire repo — only process changed files from git diff
- **DO NOT** implement GitHub App or auto-fix — MVP scope only
- **AI prompt constraints**: Do NOT report style-only issues; do NOT report test file issues; only report issues in CHANGED lines

## COMMANDS

```bash
# Build
go build -o larasense-limbo.exe .

# Test (all packages)
go test ./... -v

# Vet
go vet ./...

# Run
larasense-limbo analyze                            # default: origin/main...HEAD
larasense-limbo analyze --base origin/develop --head feature/x
larasense-limbo analyze --json                     # CI-friendly JSON output
AI_API_KEY=sk-xxx larasense-limbo analyze          # env var for API key
```

## NOTES

- **No CI/CD yet** — no GitHub Actions, Makefile, or Dockerfile
- **No git repo initialized** — `git rev-parse` fails (no commits yet)
- **Exit code 1** on any high-severity issue — designed for CI gate usage
- **AI response parsing** handles 3 formats + markdown fence stripping — fragile if provider changes format
- **`go 1.25.3`** in go.mod — verify this is intentional (unusually high version)
- **Test coverage**: Only `diff/`, `context/`, `ai/` have tests — `cmd/`, `git/`, `output/`, `reviewer/` untested
- **`classifyFile()` and `generateHint()` must stay in sync** — adding a type to one without the other silently falls through to defaults
