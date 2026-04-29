# AI Fixer Enhancement Tracker

Progress tracking untuk perbaikan dan peningkatan fitur `--fix --apply` pada larasense-limbo.

> **Legend:** ✅ Done | 🚧 In Progress | 📋 Planned | 💡 Future Idea

---

## Root Cause Issues (Resolved)

| # | Issue | Status | Root Cause | Fix |
|---|-------|--------|------------|-----|
| 1 | `--fix --apply` selalu 0 applied | ✅ Done | `before` matching terlalu strict (exact match only) | 6-strategy fuzzy matching + full-file content search |
| 2 | AI line number off by 2-10 lines | ✅ Done | AI menghitung dari awal method, bukan baris aktual | Nearby search ±15 lines + `searchByContent` full-file scan |
| 3 | `max_tokens: 1024` terlalu kecil | ✅ Done | Response terpotong (`finish_reason: "length"`), AI gagal generate fix JSON | Default dinaikkan ke `16384` |
| 4 | Token scaling menurunkan `max_tokens` | ✅ Done | Logic `scaledMax = maxTokens * 2000 / promptTokens` mengurangi budget | Scaling logic dihapus — user controls via config |
| 5 | Claude Opus mengembalikan Markdown bukan JSON | ✅ Done | Model mengabaikan `response_format: json_object` dan prompt JSON-only | Enhanced markdown parser + file hints inference |
| 6 | Truncated JSON response gagal parse | ✅ Done | `finish_reason: "length"` → JSON terpotong di tengah | `repairTruncatedJSON()` — find last complete issue, close brackets |
| 7 | Fix kedua di file sama selalu skip | ✅ Done | Fix pertama mengubah file, `before` fix kedua tidak match lagi | Re-read file setelah setiap fix + overlapping detection |
| 8 | Overlapping fixes saling merusak | ✅ Done | Fix #2 range mencakup fix #1 range, konten sudah berubah | `overlapsApplied()` — detect & skip dengan pesan jelas |

## Matching Engine (6 Strategies)

| # | Strategy | Deskripsi | Status |
|---|----------|-----------|--------|
| 1 | Exact match | `actual == expected` | ✅ Done |
| 2 | Trimmed match | `TrimSpace(actual) == TrimSpace(expected)` | ✅ Done |
| 3 | Normalized whitespace | Trim trailing spaces per line | ✅ Done |
| 4 | Stripped indentation | Ignore ALL leading whitespace per line | ✅ Done |
| 5 | Contains match | `actual` contains `expected` (min 10 chars) | ✅ Done |
| 6 | Full-file content search | Scan seluruh file by anchor line + block verify | ✅ Done |

## Interactive Mode

| # | Fitur | Status | Catatan |
|---|-------|--------|---------|
| 1 | Interactive prompt (y/n/q) | ✅ Done | Colored diff preview per fix |
| 2 | Non-interactive mode (`--yes`) | ✅ Done | Apply semua tanpa prompting |
| 3 | Skip logging dengan reason | ✅ Done | Tampilkan kenapa fix di-skip |
| 4 | Tip message saat 0 applied | ✅ Done | Suggest `--no-cache` untuk fresh data |
| 5 | Quit option (`q`) | ✅ Done | Abort remaining fixes |

## Markdown Fallback Parser

| # | Fitur | Status | Catatan |
|---|-------|--------|---------|
| 1 | Parse `### ` issue headers | ✅ Done | Standard markdown format |
| 2 | Parse `#### ` issue headers | ✅ Done | Claude Opus format |
| 3 | Extract severity dari `**Severity: High**` | ✅ Done | Inline severity markers |
| 4 | Extract line dari `(Line ~108-110)` | ✅ Done | Parenthetical line references |
| 5 | Extract `before`/`after` dari code blocks | ✅ Done | ` ```php ``` ` → Fix struct |
| 6 | Parse `**Problem:**` / `**Fix:**` markers | ✅ Done | Separate description vs suggestion |
| 7 | File path inference dari user content | ✅ Done | `extractFileHints()` — map class name → file path |
| 8 | Skip `### Issues Found` group headers | ✅ Done | Claude Opus groups issues under one header |

## File Handling

| # | Fitur | Status | Catatan |
|---|-------|--------|---------|
| 1 | Re-read file setelah setiap fix | ✅ Done | Ensure fresh content untuk fix berikutnya |
| 2 | Write file setelah setiap fix | ✅ Done | Immediate persist, bukan batch |
| 3 | Overlapping fix detection | ✅ Done | `overlapsApplied()` — skip dengan pesan jelas |
| 4 | Nearby search ±15 lines | ✅ Done | Handle AI line number offset |
| 5 | EndLine clamping | ✅ Done | AI sometimes overshoots file length |

## Model Compatibility

| Model | JSON Output | Fix Data | Apply Success | Catatan |
|-------|-------------|----------|---------------|---------|
| `gemini-2.5-flash` | ✅ Native JSON | ✅ Accurate | ✅ Works | **Recommended** — fast, cheap, reliable JSON |
| `claude-opus-4.6` | ❌ Returns Markdown | ⚠️ Via parser | ⚠️ Partial | Ignores `response_format: json_object`, needs markdown fallback |
| `gpt-5.4` | ✅ Native JSON | ✅ Accurate | ✅ Expected | Best accuracy, higher cost |
| `claude-sonnet-4.5` | ✅ Native JSON | ✅ Accurate | ✅ Works | Good balance of quality and speed |

---

## Planned Improvements

| # | Fitur | Status | Catatan |
|---|-------|--------|---------|
| 1 | Retry with higher `max_tokens` on truncation | 📋 Planned | Auto-retry saat `finish_reason: "length"` dengan 2x token budget |
| 2 | Dry-run mode (`--fix --dry-run`) | 📋 Planned | Show what would be applied tanpa mengubah file |
| 3 | Undo/rollback support | 📋 Planned | Backup file sebelum apply, `--undo` untuk revert |
| 4 | Fix confidence score | 📋 Planned | AI rate confidence per fix, skip low-confidence |
| 5 | Batch fix summary | 📋 Planned | Summary tabel di akhir: applied/skipped/overlapping counts |
| 6 | Git-aware apply | 📋 Planned | Auto-create branch sebelum apply, easy revert via `git checkout` |

## Future Ideas

| # | Fitur | Status | Catatan |
|---|-------|--------|---------|
| 7 | Multi-pass fix | 💡 Future | Re-run AI setelah fix #1 applied untuk generate fix #2 yang akurat |
| 8 | Fix dependency graph | 💡 Future | Detect fix A depends on fix B, apply in correct order |
| 9 | Semantic diff display | 💡 Future | Syntax-highlighted diff di terminal |
| 10 | Fix templates | 💡 Future | Pre-defined fix patterns tanpa AI (e.g., `Model::all()` → `Model::cursor()`) |

---

## Timeline

| Date | Changes |
|------|---------|
| 2026-04-28 | Initial `--fix`, `--patch`, `--apply` implementation |
| 2026-04-29 | Interactive mode (y/n/q), `--yes` flag, fuzzy matching (4 strategies) |
| 2026-04-29 | Full-file content search, nearby search ±10 lines, stripped indentation matching |
| 2026-04-30 | `max_tokens` 1024 → 4096 → 16384, scaling logic removed |
| 2026-04-30 | Markdown fallback parser rewrite (Claude Opus support) |
| 2026-04-30 | Truncated JSON repair (`repairTruncatedJSON`) |
| 2026-04-30 | Re-read file per fix, overlapping fix detection |
| 2026-04-30 | Nearby search expanded to ±15 lines |
