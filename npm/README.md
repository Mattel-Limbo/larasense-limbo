# larasense-limbo

AI-powered code review CLI for Laravel projects. Detects security vulnerabilities, performance issues, bad practices, and convention violations — with auto-fix.

## Install

```bash
# npm
npm install -g larasense-limbo

# bun
bun add -g larasense-limbo

# pnpm
pnpm add -g larasense-limbo

# npx (run without installing)
npx larasense-limbo scan
```

## Quick Start

```bash
# 1. Generate config in your Laravel project
cd /path/to/your/laravel-project
larasense-limbo init

# 2. Set your AI API key
export AI_API_KEY=sk-your-api-key

# 3. Scan your codebase
larasense-limbo scan

# 4. Auto-fix issues
larasense-limbo scan --auto
```

## Usage

```bash
# Review git diff (PR review)
larasense-limbo analyze --base origin/main

# Scan entire codebase
larasense-limbo scan

# Scan specific file
larasense-limbo scan app/Http/Controllers/UserController.php

# Preview fixes without applying
larasense-limbo scan --preview

# Auto-fix everything
larasense-limbo scan --auto

# Undo applied fixes
larasense-limbo undo
```

## What It Detects

- **Security** — SQL injection, XSS, CSRF, mass assignment, hardcoded secrets
- **Performance** — N+1 queries, missing eager loading, unbounded queries
- **Bad Practices** — Fat controllers, logic in views, missing validation
- **Logic Errors** — Undefined variables, unreachable code, wrong comparisons
- **Conventions** — Naming violations, missing route model binding, dead code

## Documentation

Full documentation: [github.com/Mattel-Limbo/larasense-limbo](https://github.com/Mattel-Limbo/larasense-limbo)

## License

MIT
