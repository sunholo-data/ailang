# AI Developer Experience Learnings

**Date**: 2026-02-13 (v0.8.0)
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

### CRITICAL: `ailang check` has NO `--json` flag

**Problem**: Type errors and parse errors are human-formatted strings. An AI must regex-parse
lines like:
```
PAR_UNEXPECTED_TOKEN at file.ail:14:1: expected next token to be {, got ensures instead
```

**Impact**: This is the FIRST command an AI runs. If it can't parse errors, the whole
write→check→fix loop is fragile. Currently an AI must:
1. Parse `Error:` prefix
2. Extract error code (e.g., `PAR_UNEXPECTED_TOKEN`)
3. Extract file:line:col from free text
4. Parse the suggestion line

**Suggestion**: Add `ailang check --json` that outputs:
```json
{
  "success": false,
  "errors": [
    {
      "code": "PAR_UNEXPECTED_TOKEN",
      "file": "file.ail",
      "line": 14,
      "column": 1,
      "message": "expected next token to be {, got ensures instead",
      "suggestion": "Add or correct the { token"
    }
  ]
}
```

**Priority**: P0 — this blocks the most common AI workflow.

### HIGH: Multiple `ensures` clauses silently fail as parse error

**Problem**: Writing two `ensures` blocks (natural for an AI) produces a confusing
parse error about "expected {, got ensures". There's no hint that only ONE ensures
block is allowed.

**What an AI writes** (naturally):
```ailang
ensures { result >= 0 }
ensures { result <= 50 }
```

**What it should write**:
```ailang
ensures { result >= 0 }
```

**Suggestion**: Either:
- (a) Support multiple ensures clauses (combine with AND)
- (b) Emit a specific error: "Only one ensures clause per function. Combine with: `ensures { result >= 0 && result <= 50 }`"

**Priority**: P1 — AIs will hit this constantly since multi-constraint contracts are natural.

### HIGH: `verify` doesn't accept `--relax-modules`

**Problem**: When an AI generates code in /tmp (common for sandboxed execution),
`ailang check --relax-modules` works but `ailang verify` rejects the flag entirely:
```
flag provided but not defined: -relax-modules
```

The verify command auto-relaxes for temp dirs (with a WARNING), but the flag mismatch
is confusing. An AI that learned `--relax-modules` from `check` will try it on `verify`
and get an error.

**Suggestion**: Either accept (and ignore) `--relax-modules` on all subcommands,
or add it to `verify` for consistency.

**Priority**: P1 — flag inconsistency across subcommands is a common AI trap.

### MEDIUM: Warning noise in stdout/stderr

**Problem**: Every command outputs `Warning: stdlib version mismatch: expected dev, found v0.8.0`.
This is dev-environment noise that pollutes JSON output parsing.

**Impact**: An AI parsing `ailang verify --json` output must filter warning lines before
JSON parsing. The JSON itself is clean, but preceding text isn't.

**Suggestion**: Suppress warnings when `--json` is used, or ensure warnings go to stderr
only (verify currently mixes them).

### MEDIUM: `ailang check` has no `--quiet` / `--silent` flag

**Problem**: The `→ Type checking...` and `→ Effect checking...` progress lines are useful
for humans but noise for AI parsing. There's no way to suppress them.

**Suggestion**: Add `--quiet` flag that only outputs errors (or nothing on success).

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

### Tier 1 (High Impact, Low Effort)
1. **`ailang check --json`** — Structured error output for AI parsing
2. **Better error for multiple ensures** — Specific parser error with fix suggestion
3. **`--relax-modules` on verify** — Flag consistency across subcommands
4. **`--quiet` on check/run** — Suppress progress lines

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
ailang prompt --compact > /tmp/syntax.md      # ~5KB
ailang devtools-prompt --compact > /tmp/tools.md  # ~3KB

# 2. AI writes code
# ... generates discount_calculator.ail ...

# 3. AI validates in one shot
ailang ai-check --json discount_calculator.ail
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

Currently steps 1 and 3 don't exist — the AI must manually chain multiple commands
and parse heterogeneous output formats.

---

## Raw Test Data

### Command Output Sizes
| Command | Lines | Bytes | JSON? |
|---------|-------|-------|-------|
| `ailang prompt` | 1,596 | 49,201 | No |
| `ailang devtools-prompt` | 743 | 27,024 | No |
| `ailang check` (success) | 4 | ~150 | No |
| `ailang check` (error) | 8-15 | ~500 | No |
| `ailang verify --json` | 20 | ~600 | Yes |
| `ailang run --emit-trace jsonl` | 28 | ~4,000 | Yes (JSONL) |
| `ailang replay --json` | 3 | ~60 | Yes |
| `ailang export-training --score --json` | 40 | ~800 | Yes |
| `ailang run --budget-report json` | 15 | ~300 | Yes |

### Error Format Comparison
| Tool | Error Format | Parseable? |
|------|-------------|------------|
| `check` (parse error) | Free text with PAR_ codes | Regex required |
| `check` (type error) | Free text with position | Regex required |
| `verify --json` (violation) | Clean JSON with model | Trivial |
| `verify --json` (skip) | Clean JSON with reason | Trivial |
| `replay --json` (mismatch) | Clean JSON with diffs | Trivial |

The gap is clear: **`check` is the only critical tool without JSON output.**
