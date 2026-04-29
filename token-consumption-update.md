# Token Consumption Optimization Plan

> Larasense Limbo — AI-Powered Laravel Code Review CLI

## Baseline

| Metric              | Value |
| ------------------- | ----- |
| `prompt_tokens`     | 655   |
| `completion_tokens` | 969   |
| `total_tokens`      | 1,624 |
| `credit`            | 1.38  |

**Key observation:** Completion tokens (59.7%) significantly outweigh prompt tokens (40.3%), indicating the AI output is the primary cost driver.

---

## Analysis

### Prompt Side (`client.go -> buildPrompt()` + `builder.go -> FormatForAI()`)

| Component           | Current Behavior                              | Token Impact |
| ------------------- | --------------------------------------------- | ------------ |
| System prompt       | ~320 words with verbose category descriptions | Medium       |
| File hints          | Full sentence per file type (18 types)        | Low-Medium   |
| Surrounding context | ±20 lines around first changed line per file  | Medium       |
| Diff text           | Full diff text per file, no truncation        | Variable     |
| JSON schema         | Full example JSON block with all 6 fields     | Low          |

### Completion Side (`buildPrompt()` -> JSON output format)

| Component      | Current Behavior                                        | Token Impact |
| -------------- | ------------------------------------------------------- | ------------ |
| Output format  | 6 fields per issue including verbose `description`      | High         |
| No `max_tokens`| API request has no output length limit                  | High         |
| No `temperature`| Defaults to provider default (often 1.0 = more verbose)| Low-Medium   |

### Infrastructure

| Component   | Current Behavior                             | Token Impact |
| ----------- | -------------------------------------------- | ------------ |
| Caching     | SHA-256 diff cache skips unchanged files     | Positive     |
| Retry logic | 3 retries — failed retries re-consume tokens| Risk         |
| No batching | All files sent in single request             | Variable     |

---

## Optimization Strategies

### Strategy 1: Compress System Prompt

Rewrite `buildPrompt()` to use concise, telegraphic instructions instead of full sentences. Remove redundant phrasing while preserving all review categories and rules.

**Target:** `prompt_tokens` · **Expected reduction:** ~15-20%

### Strategy 2: Add `max_tokens` to API Request

Include `max_tokens` field in `chatRequest` and `responsesRequest` structs. Set a sensible default (e.g., 1024) and make it configurable via `.larasense-limbo.yml`.

**Target:** `completion_tokens` · **Expected reduction:** ~10-30% (caps runaway responses)

### Strategy 3: Instruct Compact Output

Modify prompt to request shorter field values — e.g., `description` max 1 sentence, `suggestion` max 1 sentence. Remove the full JSON example from prompt and replace with a minimal schema reference.

**Target:** `completion_tokens` · **Expected reduction:** ~20-30%

### Strategy 4: Adaptive Surrounding Context

Reduce default context window from ±20 to ±10 lines. Make it configurable via `review.context_lines` in config. For small diffs (< 10 changed lines), use ±5.

**Target:** `prompt_tokens` · **Expected reduction:** ~10-15%

### Strategy 5: Trim File Hints

Replace verbose hint sentences with short keyword tags (e.g., `"Check: N+1, mass-assignment, $fillable"` instead of full sentences). Only include hints for the file types actually present in the diff.

**Target:** `prompt_tokens` · **Expected reduction:** ~5-10%

### Strategy 6: Add Temperature Control

Add `temperature` field to API request (default: 0.3). Lower temperature produces more focused, concise output — reducing unnecessary verbosity in completions.

**Target:** `completion_tokens` · **Expected reduction:** ~5-10%

### Strategy 7: Per-File Result Caching

Extend current cache to store AI review results per file (not just skip/no-skip). On subsequent runs, only re-review files whose diff actually changed. Return cached issues for unchanged files.

**Target:** Both (avoids API calls entirely) · **Expected reduction:** Variable (up to 100% for cached files)

---

## Expected Impact

| Scenario          | prompt_tokens | completion_tokens | total_tokens | credit (est.) |
| ----------------- | ------------- | ----------------- | ------------ | ------------- |
| **Current**       | 655           | 969               | 1,624        | 1.38          |
| **After S1-S6**   | ~490          | ~600              | ~1,090       | ~0.93         |
| **With S7 cache** | 0 (cached)    | 0 (cached)        | 0            | 0             |

**Estimated savings: ~30-35% per request** (strategies 1-6), with cache hits eliminating cost entirely (strategy 7).

---

## Priority

| Priority | Strategy                   | Effort | Impact  |
| -------- | -------------------------- | ------ | ------- |
| P0       | S2: Add `max_tokens`       | Low    | High    |
| P0       | S3: Compact output format  | Low    | High    |
| P1       | S1: Compress system prompt | Medium | Medium  |
| P1       | S6: Temperature control    | Low    | Low-Med |
| P2       | S4: Adaptive context window| Medium | Medium  |
| P2       | S5: Trim file hints        | Low    | Low     |
| P3       | S7: Per-file result caching| High   | High    |

---

## Files to Modify

| File                           | Strategies | Changes                                  |
| ------------------------------ | ---------- | ---------------------------------------- |
| `internal/ai/client.go`       | S1,S2,S3,S6| Prompt, request structs, API params     |
| `internal/config/config.go`   | S2,S4,S6   | New config fields                        |
| `internal/context/builder.go` | S4,S5      | Context window, hint format              |
| `internal/cache/cache.go`     | S7         | Store review results per file            |
| `.larasense-limbo.yml.example`| S2,S4,S6   | Document new config options              |
