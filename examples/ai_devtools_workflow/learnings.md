# AI Developer Experience Learnings

**Date**: 2026-02-13 (v0.8.0, updated for v0.8.1-dev)
**Method**: Simulated being an AI agent using AILANG devtools end-to-end.
**Module**: `discount_calculator.ail` — contracts, ADTs, effects, cross-function calls.

---

## What Works Well

### 1. `ailang verify --json` is excellent
Clean, structured JSON with function-level results, counterexample models, and durations.
An AI can parse this trivially and decide what to fix.

```json
{"function": "brokenDiscount", "status": "counterexample",
 "model": [{"name": "price", "sort": "Int", "value": "1"}]}
```
**Verdict**: Production-ready for AI consumption.

### 2. `ailang verify --verbose` shows the actual SMT-LIB
An AI can read the generated SMT-LIB and understand *why* something was verified or not.
This is unusually transparent — most verification tools are opaque.

### 3. Trace JSONL is clean and self-describing
Each event has `version`, `event`, `timestamp_ns`, and relevant payload.
Function entry/exit pairs with args and results are perfect for training data.

### 4. `ailang replay --json` is simple and correct
`{"match": true, "baseline_events": 28, "replay_events": 28}` —
exactly what an AI needs to verify determinism.

### 5. `ailang export-training --score --json` gives multi-dimensional quality scores
The breakdown (completion, complexity, contracts, budget, effects) lets an AI understand
*which dimension* is low and target improvement.

### 6. Teaching prompts are pipe-friendly
`ailang prompt > syntax.md` and `ailang devtools-prompt > tools.md` work perfectly.
An AI can inject these into its context window.

### 7. Cross-function verification inlines callees automatically
`applyDiscount` calls `discountPercent` — Z3 resolves this automatically.
No manual annotation needed. Very AI-friendly.

---

## Friction Points (Ordered by Severity)

### ~~CRITICAL~~ FIXED (v0.8.1): `ailang check --json`

**Problem**: Type errors and parse errors were human-formatted strings requiring regex parsing.

**Fix**: Added `ailang check --json` flag. Output format:
```json
{
  "file": "file.ail",
  "passed": false,
  "error_count": 1,
  "errors": [
    {
      "code": "PAR_UNEXPECTED_TOKEN",
      "message": "expected next token to be {, got ensures instead",
      "file": "file.ail",
      "line": 14,
      "column": 1,
      "suggestion": "Add or correct the { token"
    }
  ]
}
```

Also added `--quiet` flag that suppresses progress lines — useful for scripts that
only check exit code.

### ~~HIGH~~ FIXED (v0.8.1): Multiple `ensures` clauses now produce clear error

**Problem**: Writing two `ensures` blocks produced a confusing "expected {, got ensures" error.

**Fix**: Parser now detects duplicate `ensures`/`requires` blocks and emits a specific
`PAR_DUPLICATE_ENSURES` / `PAR_DUPLICATE_REQUIRES` error with suggestion:
```
PAR_DUPLICATE_ENSURES: only one ensures block per function; combine with commas: ensures { cond1, cond2 }
Suggestion: Merge conditions into a single ensures block separated by commas
```

The parser also error-recovers by merging the duplicate blocks, so the AST remains valid
for downstream tooling.

### ~~HIGH~~ FIXED (v0.8.1): `verify --relax-modules`

**Problem**: `ailang verify` rejected the `--relax-modules` flag that `check` accepted.

**Fix**: Added `--relax-modules` flag to `verify` for consistency. Also respects
`AILANG_RELAX_MODULES` environment variable.

### MEDIUM: Warning noise in stdout/stderr

**Problem**: Every command outputs `Warning: stdlib version mismatch: expected dev, found v0.8.0`.
This is dev-environment noise that pollutes JSON output parsing.

**Impact**: An AI parsing `ailang verify --json` output must filter warning lines before
JSON parsing. The JSON itself is clean, but preceding text isn't.

**Suggestion**: Suppress warnings when `--json` is used, or ensure warnings go to stderr
only (verify currently mixes them).

### MEDIUM: Budget report escapes `<global>` as `\u003cglobal\u003e`

**Problem**: JSON budget report HTML-escapes angle brackets. An AI must decode this.
```json
{"functions": {"\u003cglobal\u003e": {"effects": ...}}}
```

**Suggestion**: Use standard JSON encoding without HTML escaping.

### LOW: No unified "AI lint" command

**Problem**: An AI must run 3 separate commands to fully validate code:
```bash
ailang check file.ail          # Type errors
ailang verify --json file.ail  # Contract violations
ailang run --emit-trace jsonl --caps IO --entry main file.ail  # Runtime behavior
```

**Suggestion**: Consider `ailang ai-check --json file.ail` that runs check+verify
in one pass with unified JSON output:
```json
{
  "type_check": {"success": true},
  "contracts": {"verified": 3, "violations": 0},
  "suggestions": []
}
```

### LOW: Prompt size is large (49KB syntax + 27KB devtools = 76KB)

**Problem**: Combined prompts consume ~19K tokens. For models with small context windows,
this crowds out actual code.

**Suggestion**: Offer `ailang prompt --compact` that strips examples and keeps only
the syntax reference table (~5KB). Let the AI request the full version only when stuck.

### LOW: `export-training` contract_coverage is 0.5 even with contracts

**Problem**: The quality scorer gives 50% for contract_coverage even though all 3 functions
have verified contracts. The scorer seems to check for runtime contract *events* in the trace,
not static verification.

**Suggestion**: If `ailang verify` passes, boost contract_coverage. Static verification
should count as high-quality for training data (arguably better than runtime checks).

---

## Suggestions for v0.8.1

### Tier 1 (High Impact, Low Effort) — ALL DONE
1. ~~**`ailang check --json`**~~ — DONE: Structured error output for AI parsing
2. ~~**Better error for multiple ensures**~~ — DONE: `PAR_DUPLICATE_ENSURES` with merge recovery
3. ~~**`--relax-modules` on verify**~~ — DONE: Flag consistency across subcommands
4. ~~**`--quiet` on check**~~ — DONE: Suppress progress lines

### Tier 2 (High Impact, Medium Effort)
5. **`ailang ai-check --json`** — Unified check+verify in one JSON output
6. **`ailang prompt --compact`** — Token-efficient syntax reference (~5KB)
7. **Warning suppression in JSON mode** — Clean JSON output without preamble

### Tier 3 (Nice to Have)
8. **Contract_coverage scoring from static verify** — Better training data quality
9. **Tool description JSON schema** — Machine-readable parameter/return specs
10. **`ailang fix --json`** — Given a check/verify error, suggest the fix as a code patch

---

## The Ideal AI Workflow (What We'd Want)

```bash
# 1. AI gets syntax + tools reference (small enough for context)
ailang prompt --compact > /tmp/syntax.md      # ~5KB  (NOT YET: --compact)
ailang devtools-prompt --compact > /tmp/tools.md  # ~3KB  (NOT YET: --compact)

# 2. AI writes code
# ... generates discount_calculator.ail ...

# 3a. AI type-checks (WORKS NOW with v0.8.1)
ailang check --json discount_calculator.ail
# Returns: {"file": "...", "passed": true, "error_count": 0, "errors": []}

# 3b. AI verifies contracts (WORKS NOW)
ailang verify --json discount_calculator.ail
# Returns: {"verified": 3, "counterexample": 0, ...}

# 3c. (FUTURE) Unified check+verify in one shot
# ailang ai-check --json discount_calculator.ail
# Returns: {"type_check": {"ok": true}, "contracts": {"verified": 3}}

# 4. AI collects execution trace
ailang run --emit-trace jsonl --caps IO --entry main discount_calculator.ail > trace.jsonl

# 5. AI scores its own output
ailang export-training --score --json trace.jsonl
# Returns: {"score": {"total": 0.85, ...}}

# 6. If score < threshold, AI iterates
# Loop back to step 2 with error context

# 7. AI verifies determinism
ailang replay --json trace.jsonl
# Returns: {"match": true}
```

**v0.8.1 update**: Steps 3a and 3b now produce clean JSON. The remaining gap is
a unified `ai-check` command (step 3c) and compact prompts (step 1).

---

## Raw Test Data

### Command Output Sizes
| Command | Lines | Bytes | JSON? |
|---------|-------|-------|-------|
| `ailang prompt` | 1,596 | 49,201 | No |
| `ailang devtools-prompt` | 743 | 27,024 | No |
| `ailang check` (success) | 4 | ~150 | No |
| `ailang check --json` (success) | 6 | ~100 | Yes |
| `ailang check` (error) | 8-15 | ~500 | No |
| `ailang check --json` (error) | 15 | ~400 | Yes |
| `ailang check --quiet` (success) | 0 | 0 | No (exit 0) |
| `ailang verify --json` | 20 | ~600 | Yes |
| `ailang run --emit-trace jsonl` | 28 | ~4,000 | Yes (JSONL) |
| `ailang replay --json` | 3 | ~60 | Yes |
| `ailang export-training --score --json` | 40 | ~800 | Yes |
| `ailang run --budget-report json` | 15 | ~300 | Yes |

### Error Format Comparison
| Tool | Error Format | Parseable? |
|------|-------------|------------|
| `check --json` (parse error) | Clean JSON with code/message/file/line/col | Trivial |
| `check --json` (type error) | Clean JSON with code/message | Trivial |
| `check` (parse error) | Free text with PAR_ codes | Regex required |
| `check` (type error) | Free text with position | Regex required |
| `verify --json` (violation) | Clean JSON with model | Trivial |
| `verify --json` (skip) | Clean JSON with reason | Trivial |
| `replay --json` (mismatch) | Clean JSON with diffs | Trivial |

**v0.8.1 update**: All critical tools now have JSON output. The remaining gap is
warning noise (stdlib version mismatch) that precedes JSON on some commands.
