# Setup Larasense Limbo — GitHub Action

Download and cache the [larasense-limbo](https://github.com/Mattel-Limbo/larasense-limbo) CLI binary for AI-powered Laravel code review in GitHub Actions.

## Usage

### Basic (latest version)

```yaml
steps:
  - uses: actions/checkout@v4
    with:
      fetch-depth: 0

  - uses: Mattel-Limbo/larasense-limbo/action@main
  
  - run: larasense-limbo analyze --base origin/${{ github.base_ref }} --format github
    env:
      AI_API_KEY: ${{ secrets.AI_API_KEY }}
```

### Pin to specific version

```yaml
  - uses: Mattel-Limbo/larasense-limbo/action@main
    with:
      version: "0.5.1"
```

### Full example — PR review with inline annotations + comment

```yaml
name: AI Code Review

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

      - name: Setup larasense-limbo
        uses: Mattel-Limbo/larasense-limbo/action@main

      - name: Run AI Code Review
        env:
          AI_API_KEY: ${{ secrets.AI_API_KEY }}
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          larasense-limbo analyze \
            --base origin/${{ github.base_ref }} \
            --head ${{ github.sha }} \
            --format github \
            --github-pr ${{ github.repository }}#${{ github.event.pull_request.number }}
```

### Full codebase scan (scheduled)

```yaml
name: Weekly Code Audit

on:
  schedule:
    - cron: "0 9 * * 1"

jobs:
  audit:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: Mattel-Limbo/larasense-limbo/action@main

      - run: larasense-limbo scan --json > audit-report.json
        env:
          AI_API_KEY: ${{ secrets.AI_API_KEY }}

      - uses: actions/upload-artifact@v4
        with:
          name: code-audit-report
          path: audit-report.json
```

## Inputs

| Input | Required | Default | Description |
|-------|----------|---------|-------------|
| `version` | No | `latest` | Version to install (e.g. `0.5.1`). Fetches latest release if not specified. |
| `github-token` | No | `${{ github.token }}` | GitHub token for API calls (avoids rate limits when fetching latest version). |

## Outputs

| Output | Description |
|--------|-------------|
| `version` | The installed version of larasense-limbo |
| `path` | Full path to the installed binary |

## How It Works

1. Resolves the version (fetches latest from GitHub API if `version: latest`)
2. Detects runner OS and architecture (`runner.os` / `runner.arch`)
3. Checks cache (`actions/cache`) — skips download if already cached
4. Downloads the correct pre-built binary from GitHub Releases
5. Extracts and installs to `runner.tool_cache`
6. Adds to `$GITHUB_PATH` so `larasense-limbo` is available in subsequent steps
7. Verifies installation

## Supported Platforms

| Runner OS | Architecture | Archive Format |
|-----------|-------------|----------------|
| `ubuntu-latest` | x64, arm64 | `.tar.gz` |
| `macos-latest` | x64, arm64 | `.tar.gz` |
| `windows-latest` | x64, arm64 | `.zip` |

## License

MIT
