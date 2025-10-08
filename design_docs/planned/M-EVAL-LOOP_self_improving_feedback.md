# M-EVAL-LOOP: Self-Improving AI Feedback Loop

**Status**: 📋 Planned
**Version**: 1.0
**Author**: AI-Assisted (Claude + User + GPT-5 + Gemini 2.5 Pro)
**Date**: 2025-10-08
**Milestone**: v0.3.0-alpha4
**Priority**: HIGH - Core infrastructure for AI-first language development

---

## Executive Summary

Transform the AILANG eval harness from a passive benchmarking tool into a **self-improving feedback loop** that:

1. **Automatically identifies language weaknesses** via AI code generation failures
2. **Generates actionable design documents** from failure patterns (✅ Already implemented)
3. **Teaches AI models to self-repair** via error taxonomy and retry guidance
4. **Validates improvements** by re-running affected benchmarks
5. **Tracks performance evolution** across models, prompts, and language versions

**Key Innovation**: The eval harness becomes a "tutor loop" where compiler errors teach AI models how to use AILANG, and failure patterns drive language improvements.

**Immediate Value**:
- Measure **teachability**: Can AI learn from error messages? (first-attempt vs repair success)
- Optimize **prompts**: A/B test teaching strategies across GPT, Claude, Gemini
- Validate **fixes**: Prove design docs actually improve success rates
- Track **evolution**: Maintain performance tables showing progress over time

---

## Motivation

### Current State (v0.3.0-alpha3)

**✅ What Works:**
- Multi-model eval harness (OpenAI, Anthropic, Google)
- Automated design doc generation from failures (`eval-analyze`)
- Deduplication of similar design docs
- Context-rich prompts with actual implementation examples

**❌ Critical Gaps:**
1. **No self-repair**: Single-shot only; can't test if AI learns from errors
2. **No prompt versioning**: Can't A/B test teaching strategies
3. **No validation loop**: Can't prove fixes actually work
4. **No performance tracking**: Can't measure language evolution over time
5. **No error taxonomy**: Generic errors don't teach AI how to fix mistakes

### The Problem

Traditional language development optimizes for **human** programmers. AILANG optimizes for **AI-assisted** programming. But we lack:

- **Teachability metrics**: Can AI models learn from AILANG's error messages?
- **Prompt evolution tracking**: Which teaching strategies work best?
- **Causal validation**: Did our fix actually improve AI performance?
- **Historical trends**: Is the language getting easier for AI to use?

### The Solution: Close the Loop

```
┌─────────────────────────────────────────────────────────────┐
│                    SELF-IMPROVING LOOP                       │
└─────────────────────────────────────────────────────────────┘

1. EVAL          → Run benchmarks, collect failures
   ↓
2. ANALYZE       → Generate design docs (✅ Already implemented)
   ↓
3. ITERATE       → Review with GPT-5/Claude/Gemini (manual)
   ↓
4. IMPLEMENT     → Fix language/compiler/stdlib
   ↓
5. VALIDATE      → Re-run affected benchmarks, measure improvement
   ↓
6. TRACK         → Update performance tables, badges, charts
   ↓
   └─────────────→ Back to EVAL (continuous improvement)
```

**New Features Enable:**
- **Self-Repair Loop**: AI retries with error-specific guidance
- **Prompt A/B Testing**: Compare teaching strategies
- **Validation Tools**: Baseline → Fix → Diff → Measure
- **Performance Tracking**: Historical tables with 0-shot, 1-shot, n-shot metrics

---

## Design Goals

### 1. Tutor Loop: Teach AI via Error Taxonomy

**Goal**: Transform compiler from "pass/fail gate" to "interactive tutor"

**Mechanism**:
```
AI generates code → Compile fails → Extract error code
                                      ↓
                            TC_REC_001: "Record field missing"
                                      ↓
                            Inject repair guidance:
                            "Add the field, or use row polymorphism"
                                      ↓
                            AI retries ONCE
                                      ↓
                            Measure: first_attempt vs repair_success
```

**Why This Matters**:
- Real-world coding is iterative, not single-shot
- Error messages are the primary teaching tool
- Repair success rate = proxy for "Developer Experience"

### 2. Proof Loop: Validate Fixes Causally

**Goal**: Ensure design docs → implementation → measurable improvement

**Mechanism**:
```
Store baseline → Implement fix → Re-run benchmarks → Compare delta
                                                        ↓
                                            "Float equality: 60% → 95%"
```

**Why This Matters**:
- Accountability: Did we solve the problem?
- Regression detection: Did we break something else?
- Data-driven prioritization: Focus on high-impact fixes

### 3. Control Loop: Isolate Variables via Versioning

**Goal**: Distinguish prompt improvements from language improvements

**Mechanism**:
```
v0.3.0-baseline (minimal) → 85% success
v0.3.0-hints (+ reminders) → 92% success
                               ↓
                    Prompt improvement: +7%
```

**Why This Matters**:
- Scientific rigor: Isolate what actually works
- Prompt evolution: Improve teaching strategies
- Model comparison: Which AI benefits from which prompt style?

### 4. Track Loop: Measure Evolution Over Time

**Goal**: Maintain performance tables showing language maturity

**Mechanism**:
- Store results in `eval_results/history/v0.3.0/`
- Generate tables: model × benchmark × shot-type
- Track trends: success rate, token efficiency, repair success

**Why This Matters**:
- Marketing material: "AILANG saves 13% tokens vs Python"
- Development focus: "CAP_001 accounts for 24% of failures"
- Stakeholder visibility: Charts show measurable progress

---

## Implementation Plan

### Phase 1: Core Loop (12-16 hours, 1.5-2 days)

#### Task 1: Error Taxonomy (~2-3 hours)

**File**: `internal/eval_harness/errors.go` (NEW, ~150 LOC)

**Drop-in starter table:**

```go
package eval_harness

import (
    "fmt"
    "regexp"
)

type ErrCode string

const (
    PAR_001    ErrCode = "PAR_001"    // Parse error
    TC_REC_001 ErrCode = "TC_REC_001" // Record field not found
    TC_INT_001 ErrCode = "TC_INT_001" // Not an instance of Integral
    EQ_001     ErrCode = "EQ_001"     // Wrong Eq dictionary
    CAP_001    ErrCode = "CAP_001"    // Capability missing
    MOD_001    ErrCode = "MOD_001"    // Undefined module/entry
)

type RepairHint struct {
    Title string // Short description
    Why   string // Root cause explanation
    How   string // Concrete fix instruction
}

var Rules = []struct {
    Code ErrCode
    Re   *regexp.Regexp
    Hint RepairHint
}{
    {
        PAR_001,
        regexp.MustCompile(`parse error.*unexpected .* near`),
        RepairHint{
            "Block/semicolon issue",
            "Parser expects semicolons in `{ ... }` blocks.",
            "Add `;` between expressions inside `{}` or unwrap single-expression blocks.",
        },
    },
    {
        TC_REC_001,
        regexp.MustCompile(`field '([^']+)' not found in record.*\{([^}]*)\}`),
        RepairHint{
            "Record field missing",
            "Type checker requires the field to exist.",
            "Add the field, or generalize the function type to `{ <field>: T | ρ }`.",
        },
    },
    {
        TC_INT_001,
        regexp.MustCompile(`Float .* is not an instance of Integral|mod not defined for Float`),
        RepairHint{
            "Modulo on Float",
            "`%` requires `Integral` (Int).",
            "Use integers for `%`, or use `/` and `floor` for floats.",
        },
    },
    {
        EQ_001,
        regexp.MustCompile(`Eq dictionary resolution failed|using eq_Int for Float`),
        RepairHint{
            "Float equality dictionary",
            "The Eq dictionary must match Float.",
            "Annotate as `: float` or ensure both sides are Float.",
        },
    },
    {
        CAP_001,
        regexp.MustCompile(`effect '(\w+)' requires capability`),
        RepairHint{
            "Missing capability",
            "Effect calls require explicit caps.",
            "Run with `--caps IO,FS,Clock,Net` (only what you need).",
        },
    },
    {
        MOD_001,
        regexp.MustCompile(`entrypoint '(\w+)' not found|module .* not found`),
        RepairHint{
            "Entrypoint/module resolution",
            "Runner couldn't find your export.",
            "Export a zero-arg `main`, or pass `--entry yourFunc`.",
        },
    },
}

// CategorizeError matches stderr against error patterns
func CategorizeError(stderr string) (ErrCode, *RepairHint) {
    for _, rule := range Rules {
        if rule.Re.MatchString(stderr) {
            return rule.Code, &rule.Hint
        }
    }
    return "", nil // Unknown error
}

// FormatRepairPrompt creates the injection for retry
func FormatRepairPrompt(code ErrCode, hint *RepairHint, benchmarkID, lang string) string {
    return fmt.Sprintf(`Your previous program failed with:
<%s>: %s
Why: %s
How to fix: %s

Please produce a corrected %s program that compiles and runs
for the benchmark "%s". Keep it minimal, single file,
no extra commentary.`, code, hint.Title, hint.Why, hint.How, lang, benchmarkID)
}
```

**Tests**: `internal/eval_harness/errors_test.go` (~100 LOC)
- Test each regex with real stderr examples
- Verify prompt formatting
- Test unknown errors return nil

**Acceptance Criteria**:
- [ ] All 6 error codes match expected patterns
- [ ] Unknown errors return `("", nil)`
- [ ] Repair prompts include all 3 components (Why, How, corrected program)
- [ ] Unit tests cover all error categories

---

#### Task 2: Self-Repair Loop (~3-4 hours)

**File**: `internal/eval_harness/metrics.go` (MODIFY)

**Extend RunMetrics**:
```go
type RunMetrics struct {
    // ... existing fields ...

    // Self-repair metrics (NEW)
    FirstAttemptOk  bool    `json:"first_attempt_ok"`
    RepairUsed      bool    `json:"repair_used"`
    RepairOk        bool    `json:"repair_ok"`
    ErrCode         ErrCode `json:"err_code,omitempty"`
    RepairTokensIn  int     `json:"repair_tokens_in,omitempty"`
    RepairTokensOut int     `json:"repair_tokens_out,omitempty"`

    // Prompt versioning (NEW)
    PromptVersion   string  `json:"prompt_version"`

    // Reproducibility (NEW)
    BinaryHash      string  `json:"binary_hash"`
    StdlibHash      string  `json:"stdlib_hash"`
    Caps            []string `json:"caps"`
}
```

**File**: `internal/eval_harness/runner.go` (MODIFY)

**Add retry logic**:
```go
// RunWithRepair executes benchmark with optional self-repair
func (r *Runner) RunWithRepair(spec *BenchmarkSpec, selfRepair bool) *RunMetrics {
    // First attempt
    metrics := r.runOnce(spec, "")
    metrics.FirstAttemptOk = metrics.StdoutOk

    // If failed and self-repair enabled
    if !metrics.FirstAttemptOk && selfRepair {
        // Categorize error
        errCode, hint := CategorizeError(metrics.Stderr)

        if hint != nil {
            metrics.ErrCode = errCode

            // Build repair prompt
            repairPrompt := FormatRepairPrompt(errCode, hint, spec.ID, spec.Lang)

            // Retry ONCE
            repairMetrics := r.runOnce(spec, repairPrompt)

            // Update metrics
            metrics.RepairUsed = true
            metrics.RepairOk = repairMetrics.StdoutOk
            metrics.RepairTokensIn = repairMetrics.InputTokens
            metrics.RepairTokensOut = repairMetrics.OutputTokens

            if repairMetrics.StdoutOk {
                // Use repaired code
                metrics.Code = repairMetrics.Code
                metrics.StdoutOk = true
            }
        }
    }

    return metrics
}

// runOnce executes a single attempt (with optional repair guidance)
func (r *Runner) runOnce(spec *BenchmarkSpec, repairGuidance string) *RunMetrics {
    // Load prompt
    prompt := spec.PromptForLanguage()

    // Inject repair guidance if provided
    if repairGuidance != "" {
        prompt = prompt + "\n\n## Previous Attempt Failed\n" + repairGuidance
    }

    // Generate code via AI model
    result, err := r.agent.Generate(prompt, r.model, r.seed)

    // ... rest of execution logic (compile, run, compare) ...
}
```

**File**: `cmd/ailang/eval.go` (MODIFY)

**Add flags**:
```go
var (
    selfRepair    bool
    promptVersion string
)

func init() {
    evalCmd.Flags().BoolVar(&selfRepair, "self-repair", false,
        "Enable single-shot self-repair on errors")
    evalCmd.Flags().StringVar(&promptVersion, "prompt-version", "",
        "Use specific prompt version (e.g., v0.3.0-hints)")
}
```

**Update command logic**:
```go
func runEval(cmd *cobra.Command, args []string) error {
    // ... existing setup ...

    // Use RunWithRepair instead of Run
    metrics := runner.RunWithRepair(spec, selfRepair)

    // ... save results ...
}
```

**Acceptance Criteria**:
- [ ] `--self-repair` flag triggers retry on failure
- [ ] Repair metrics tracked: `repair_used`, `repair_ok`, `repair_tokens_*`
- [ ] Error code attached to metrics: `err_code`
- [ ] Repaired code saved if successful
- [ ] Token count includes both attempts

---

#### Task 3: Prompt Versioning (~2-3 hours)

**File**: `prompts/versions.json` (NEW)

```json
{
  "v0.3.0-baseline": {
    "desc": "Plain task, minimal guidance",
    "file": "v0.3.0.md",
    "hash": "sha256:f4a7b3c9e1d2..."
  },
  "v0.3.0-hints": {
    "desc": "Includes block/semicolon & caps reminders",
    "file": "v0.3.0-hints.md",
    "hash": "sha256:a1b2c3d4e5f6..."
  },
  "python-v1": {
    "desc": "Standard Python instructions",
    "file": "python.md",
    "hash": "sha256:9876543210ab..."
  }
}
```

**File**: `prompts/v0.3.0-hints.md` (NEW)

Copy `v0.3.0.md` and add explicit reminders:

```markdown
# AILANG v0.3.0 Teaching Prompt (with Hints)

... [existing content] ...

## ⚠️ Common Pitfalls (IMPORTANT)

### 1. Block Expressions Require Semicolons
**Wrong:**
```ailang
{ println("Hello") println("World") }
```

**Correct:**
```ailang
{ println("Hello"); println("World") }
```

### 2. Effects Require Capabilities
If your code uses `println`, `readFile`, etc., you MUST:
- Add `! {IO}` or `! {FS}` to function signature
- The runner will use `--caps IO,FS` when executing

### 3. Record Update NOT Implemented
**Wrong:**
```ailang
{person | age: 31}  -- NOT supported yet
```

**Correct:**
```ailang
{name: person.name, age: 31, city: person.city}  -- Create new record
```

### 4. Modulo (%) Requires Int
**Wrong:**
```ailang
3.14 % 2.0  -- Float not supported
```

**Correct:**
```ailang
10 % 3  -- Use Int for modulo
```

... [rest of prompt] ...
```

**File**: `internal/eval_harness/prompt_loader.go` (NEW, ~100 LOC)

```go
package eval_harness

import (
    "crypto/sha256"
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
)

type PromptMeta struct {
    Desc string `json:"desc"`
    File string `json:"file"`
    Hash string `json:"hash"`
}

type VersionsFile struct {
    Versions map[string]PromptMeta `json:"versions"`
}

// LoadPromptVersion loads a specific prompt version and verifies hash
func LoadPromptVersion(version string) (string, error) {
    // Load versions.json
    data, err := os.ReadFile("prompts/versions.json")
    if err != nil {
        return "", fmt.Errorf("failed to load versions.json: %w", err)
    }

    var versions VersionsFile
    if err := json.Unmarshal(data, &versions); err != nil {
        return "", fmt.Errorf("failed to parse versions.json: %w", err)
    }

    // Find version
    meta, ok := versions.Versions[version]
    if !ok {
        return "", fmt.Errorf("prompt version %q not found", version)
    }

    // Load prompt file
    promptPath := filepath.Join("prompts", meta.File)
    promptData, err := os.ReadFile(promptPath)
    if err != nil {
        return "", fmt.Errorf("failed to load prompt %s: %w", promptPath, err)
    }

    // Verify hash
    actualHash := fmt.Sprintf("sha256:%x", sha256.Sum256(promptData))
    if actualHash != meta.Hash {
        return "", fmt.Errorf("prompt hash mismatch: expected %s, got %s",
            meta.Hash, actualHash)
    }

    return string(promptData), nil
}

// GetCurrentVersion returns the default prompt version
func GetCurrentVersion(lang string) string {
    if lang == "python" {
        return "python-v1"
    }
    return "v0.3.0-baseline" // Default for AILANG
}
```

**Tool**: `tools/eval_prompt_ab.sh` (NEW, ~50 LOC)

```bash
#!/usr/bin/env bash
# Compare two prompt versions side-by-side

set -euo pipefail

A="${1:-v0.3.0-baseline}"
B="${2:-v0.3.0-hints}"

echo "Comparing prompt versions: $A vs $B"

# Run evals for version A
echo "Running evals with $A..."
mkdir -p "eval_results/${A}"
ailang eval --all-benchmarks --prompt-version "$A" --output-dir "eval_results/${A}"

# Run evals for version B
echo "Running evals with $B..."
mkdir -p "eval_results/${B}"
ailang eval --all-benchmarks --prompt-version "$B" --output-dir "eval_results/${B}"

# Compare results
echo ""
echo "===== COMPARISON ====="
./tools/compare_results.sh "eval_results/${A}" "eval_results/${B}"
```

**Tool**: `tools/compare_results.sh` (NEW, ~80 LOC)

```bash
#!/usr/bin/env bash
# Compare two result directories

DIR_A="$1"
DIR_B="$2"

# Count successes
SUCCESS_A=$(jq -r 'select(.stdout_ok == true) | .id' "$DIR_A"/*.json | wc -l)
SUCCESS_B=$(jq -r 'select(.stdout_ok == true) | .id' "$DIR_B"/*.json | wc -l)

TOTAL=$(ls "$DIR_A"/*.json | wc -l)

# Calculate percentages
PCT_A=$(echo "scale=1; $SUCCESS_A * 100 / $TOTAL" | bc)
PCT_B=$(echo "scale=1; $SUCCESS_B * 100 / $TOTAL" | bc)

# Calculate token averages
TOKENS_A=$(jq -s 'map(.output_tokens) | add / length' "$DIR_A"/*.json)
TOKENS_B=$(jq -s 'map(.output_tokens) | add / length' "$DIR_B"/*.json)

echo "Directory A: $DIR_A"
echo "  Success: $SUCCESS_A/$TOTAL ($PCT_A%)"
echo "  Avg Output Tokens: $TOKENS_A"
echo ""
echo "Directory B: $DIR_B"
echo "  Success: $SUCCESS_B/$TOTAL ($PCT_B%)"
echo "  Avg Output Tokens: $TOKENS_B"
echo ""

# Calculate delta
DELTA=$(echo "$PCT_B - $PCT_A" | bc)
if (( $(echo "$DELTA > 0" | bc -l) )); then
    echo "✅ Version B improved by ${DELTA}%"
elif (( $(echo "$DELTA < 0" | bc -l) )); then
    echo "⚠️  Version B regressed by ${DELTA}%"
else
    echo "→ No change"
fi
```

**Acceptance Criteria**:
- [ ] `prompts/versions.json` defines at least 2 versions
- [ ] `prompts/v0.3.0-hints.md` includes 4+ explicit reminders
- [ ] `--prompt-version` flag loads correct prompt
- [ ] Hash verification prevents tampering
- [ ] `eval-prompt-ab.sh` compares two versions side-by-side

---

#### Task 4: AI-Friendly Formats (~2-3 hours)

**Tool**: `tools/generate_summary_jsonl.sh` (NEW, ~80 LOC)

```bash
#!/usr/bin/env bash
# Convert eval_results/*.json to summary.jsonl (one line per run)

set -euo pipefail

RESULTS_DIR="${1:-eval_results}"
OUTPUT="${RESULTS_DIR}/summary.jsonl"

echo "Generating summary.jsonl from $RESULTS_DIR..."

# Clear output file
> "$OUTPUT"

# Convert each JSON to single-line JSONL entry
for json in "${RESULTS_DIR}"/*.json; do
    [ -f "$json" ] || continue

    jq -c '{
        ts: .timestamp,
        benchmark: .id,
        model: .model,
        lang: .lang,
        ok: .stdout_ok,
        first_attempt_ok: .first_attempt_ok,
        repair_used: .repair_used,
        repair_ok: .repair_ok,
        tokens_in: .input_tokens,
        tokens_out: .output_tokens,
        repair_tokens_in: .repair_tokens_in,
        repair_tokens_out: .repair_tokens_out,
        ms: .duration_ms,
        err_code: .err_code,
        caps: .caps,
        prompt_version: .prompt_version
    }' "$json" >> "$OUTPUT"
done

echo "✓ Generated: $OUTPUT ($(wc -l < "$OUTPUT") entries)"
```

**Tool**: `tools/generate_matrix_json.sh` (NEW, ~120 LOC)

```bash
#!/usr/bin/env bash
# Generate matrix.json for quick lookups

set -euo pipefail

RESULTS_DIR="${1:-eval_results}"
SUMMARY="${RESULTS_DIR}/summary.jsonl"
OUTPUT="${RESULTS_DIR}/matrix.json"

echo "Generating matrix.json from summary.jsonl..."

# Use jq to pivot data into matrix structure
jq -s '
{
  meta: {
    generated: (now | strftime("%Y-%m-%dT%H:%M:%SZ")),
    models: (map(.model) | unique | sort),
    benchmarks: (map(.benchmark) | unique | sort)
  },
  benchmarks: (
    group_by(.benchmark) | map({
      key: .[0].benchmark,
      value: (
        group_by(.model) | map({
          key: .[0].model,
          value: (
            group_by(.lang) | map({
              key: .[0].lang,
              value: {
                ok: (map(.ok) | all),
                tokens: (map(.tokens_out) | add / length),
                ms: (map(.ms) | add / length),
                repair_rate: (
                  (map(select(.repair_used == true and .repair_ok == true)) | length) /
                  (map(select(.repair_used == true)) | length | if . == 0 then 1 else . end)
                )
              }
            }) | from_entries)
          }
        }) | from_entries)
      }
    }) | from_entries)
  )
}
' "$SUMMARY" > "$OUTPUT"

echo "✓ Generated: $OUTPUT"
```

**Acceptance Criteria**:
- [ ] `summary.jsonl` has one line per benchmark run
- [ ] `matrix.json` pivots by benchmark → model → lang
- [ ] Both files validate as proper JSON/JSONL
- [ ] Matrix includes repair_rate calculation

---

#### Task 5: Validation Loop (~2-3 hours)

**Tool**: `tools/eval_store_baseline.sh` (NEW, ~30 LOC)

```bash
#!/usr/bin/env bash
# Store current results as baseline for comparison

set -euo pipefail

RESULTS_DIR="${1:-eval_results}"
BASELINE_DIR="${RESULTS_DIR}/baseline"

echo "Storing baseline from $RESULTS_DIR..."

# Create baseline directory
mkdir -p "$BASELINE_DIR"

# Copy results
cp "$RESULTS_DIR"/*.json "$BASELINE_DIR/" 2>/dev/null || true
cp "$RESULTS_DIR"/summary.jsonl "$BASELINE_DIR/" 2>/dev/null || true
cp "$RESULTS_DIR"/matrix.json "$BASELINE_DIR/" 2>/dev/null || true

# Store timestamp
echo "Baseline stored: $(date -u +%Y-%m-%dT%H:%M:%SZ)" > "$BASELINE_DIR/timestamp.txt"

echo "✓ Baseline stored in $BASELINE_DIR"
```

**Tool**: `tools/eval_diff.sh` (NEW, ~100 LOC)

```bash
#!/usr/bin/env bash
# Compare current results against baseline

set -euo pipefail

RESULTS_DIR="${1:-eval_results}"
BASELINE_DIR="${RESULTS_DIR}/baseline"
OUTPUT="${RESULTS_DIR}/diff_report.md"

if [ ! -d "$BASELINE_DIR" ]; then
    echo "❌ No baseline found. Run: make eval-baseline"
    exit 1
fi

echo "Comparing current vs baseline..."

# Use jq to compare summary.jsonl files
jq -s '
def by_id: group_by(.benchmark + .model + .lang) | map({key: (.[0].benchmark + "_" + .[0].model + "_" + .[0].lang), value: .[0]}) | from_entries;

.[0] as $baseline | .[1] as $current |
($baseline | by_id) as $base_map |
($current | by_id) as $curr_map |

($curr_map | keys) as $all_keys |

{
  improvements: [
    $all_keys[] |
    select($curr_map[.].ok == true and $base_map[.].ok == false) |
    {
      benchmark: $curr_map[.].benchmark,
      model: $curr_map[.].model,
      lang: $curr_map[.].lang,
      status: "failed → passed"
    }
  ],
  regressions: [
    $all_keys[] |
    select($curr_map[.].ok == false and $base_map[.].ok == true) |
    {
      benchmark: $curr_map[.].benchmark,
      model: $curr_map[.].model,
      lang: $curr_map[.].lang,
      status: "passed → failed"
    }
  ],
  token_changes: [
    $all_keys[] |
    select($base_map[.] and $curr_map[.]) |
    {
      benchmark: $curr_map[.].benchmark,
      model: $curr_map[.].model,
      lang: $curr_map[.].lang,
      baseline_tokens: $base_map[.].tokens_out,
      current_tokens: $curr_map[.].tokens_out,
      delta: ($curr_map[.].tokens_out - $base_map[.].tokens_out),
      pct_change: (($curr_map[.].tokens_out - $base_map[.].tokens_out) / $base_map[.].tokens_out * 100)
    }
  ]
}
' "$BASELINE_DIR/summary.jsonl" "$RESULTS_DIR/summary.jsonl" > "$RESULTS_DIR/diff.json"

# Generate markdown report
cat > "$OUTPUT" <<EOF
# Eval Diff Report

**Baseline**: $(cat "$BASELINE_DIR/timestamp.txt")
**Current**: $(date -u +%Y-%m-%dT%H:%M:%SZ)

## Improvements ✅

$(jq -r '.improvements[] | "- \(.benchmark) (\(.model), \(.lang)): \(.status)"' "$RESULTS_DIR/diff.json")

## Regressions ⚠️

$(jq -r '.regressions[] | "- \(.benchmark) (\(.model), \(.lang)): \(.status)"' "$RESULTS_DIR/diff.json")

## Token Changes

$(jq -r '.token_changes[] | select(.pct_change > 5 or .pct_change < -5) | "- \(.benchmark) (\(.model), \(.lang)): \(.baseline_tokens) → \(.current_tokens) (\(.pct_change | floor)%)"' "$RESULTS_DIR/diff.json")
EOF

echo "✓ Diff report: $OUTPUT"
cat "$OUTPUT"
```

**Tool**: `tools/eval_validate_fix.sh` (NEW, ~100 LOC)

```bash
#!/usr/bin/env bash
# Re-run specific benchmarks and validate fix

set -euo pipefail

BENCH="${1:-}"

if [ -z "$BENCH" ]; then
    # Re-run all previously failed benchmarks
    echo "No benchmark specified, re-running all failed benchmarks from baseline..."
    FAILED=$(jq -r 'select(.ok == false) | .benchmark' eval_results/baseline/summary.jsonl | sort -u)
else
    FAILED="$BENCH"
fi

echo "Re-evaluating benchmarks: $FAILED"

for bench in $FAILED; do
    echo ""
    echo "📊 Running: $bench"
    ailang eval --benchmark "$bench" --model gpt-4o-mini --self-repair
done

# Generate summary formats
./tools/generate_summary_jsonl.sh
./tools/generate_matrix_json.sh

# Compare against baseline
./tools/eval_diff.sh

echo ""
echo "✅ Validation complete. See eval_results/diff_report.md"
```

**Acceptance Criteria**:
- [ ] `eval-baseline` stores current results
- [ ] `eval-diff` generates markdown report with improvements/regressions
- [ ] `eval-validate-fix` re-runs failed benchmarks
- [ ] Diff report shows specific benchmarks that improved/regressed

---

#### Task 6: Makefile Targets (~30 min)

Add to `Makefile`:

```makefile
# ===== AI EVAL FEEDBACK LOOP =====

# Self-repair
eval-suite-repair: build
	@echo "Running eval suite with self-repair enabled..."
	@$(BUILD_DIR)/$(BINARY) eval --all-benchmarks --self-repair \
		--model gpt-4o-mini

# Prompt testing
eval-prompt-test: build
	@echo "A/B testing prompt versions..."
	./tools/eval_prompt_ab.sh v0.3.0-baseline v0.3.0-hints

eval-prompt-compare: build
	@if [ -z "$(A)" ] || [ -z "$(B)" ]; then \
		echo "Usage: make eval-prompt-compare A=v0.3.0-baseline B=v0.3.0-hints"; \
		exit 1; \
	fi
	./tools/eval_prompt_ab.sh $(A) $(B)

# Validation workflow
eval-baseline:
	@echo "Storing current results as baseline..."
	./tools/eval_store_baseline.sh

eval-diff:
	@echo "Comparing current vs baseline..."
	./tools/eval_diff.sh

eval-validate-fix:
	@echo "Re-running failed benchmarks to validate fix..."
	./tools/eval_validate_fix.sh $(BENCH)

# AI-friendly formats
eval-formats:
	@echo "Generating AI-friendly formats..."
	./tools/generate_summary_jsonl.sh
	./tools/generate_matrix_json.sh

# Complete iteration workflow
eval-iterate: eval-suite eval-analyze eval-formats
	@echo ""
	@echo "✅ Eval results analyzed. Design docs in design_docs/planned/"
	@echo ""
	@echo "Next steps:"
	@echo "  1. Review design docs (design_docs/planned/EVAL_ANALYSIS_*.md)"
	@echo "  2. Store baseline: make eval-baseline"
	@echo "  3. Implement fixes (manual)"
	@echo "  4. Validate: make eval-validate-fix"
	@echo "  5. Review diff: cat eval_results/diff_report.md"
```

**Acceptance Criteria**:
- [ ] All 8 new make targets work
- [ ] `make help` shows new targets with descriptions
- [ ] `eval-iterate` runs full pipeline without errors

---

#### Task 7: Documentation (~1-2 hours)

**File**: `docs/guides/eval_workflow.md` (NEW)

See full content in appendix. Key sections:
- Quick Start (5 steps)
- Self-Repair Usage
- Prompt A/B Testing
- Validation Workflow
- Performance Tracking Tables
- Example Session

**File**: `docs/guides/performance_tracking.md` (NEW)

See full content in appendix. Key sections:
- Performance Tables Schema
- 0-shot, 1-shot, n-shot Definitions
- How to Update Tables
- Historical Tracking

**Acceptance Criteria**:
- [ ] Documentation covers all new features
- [ ] Includes copy-paste command examples
- [ ] Shows expected output for each command
- [ ] Performance tracking tables documented

---

### Phase 2: Performance Tracking System (~4-6 hours)

#### Task 8: Performance Tables Infrastructure

**File**: `eval_results/performance_tables/schema.md` (NEW)

Documents the schema for tracking evolution over time.

**File**: `eval_results/performance_tables/v0.3.0_baseline.json` (NEW)

```json
{
  "version": "v0.3.0-alpha3",
  "date": "2025-10-08",
  "prompt_version": "v0.3.0-baseline",
  "models": {
    "gpt-4o-mini": {
      "benchmarks": {
        "recursion_factorial": {
          "0-shot": { "success_rate": 1.0, "avg_tokens": 83, "avg_ms": 56 },
          "1-shot": { "success_rate": 1.0, "avg_tokens": 44, "avg_ms": 32 }
        },
        "float_eq": {
          "0-shot": { "success_rate": 0.6, "avg_tokens": 95, "avg_ms": 48 },
          "1-shot": { "success_rate": 0.95, "avg_tokens": 52, "avg_ms": 38 }
        }
      },
      "aggregates": {
        "0-shot_success": 0.85,
        "1-shot_success": 0.92,
        "avg_tokens": 89,
        "repair_success_rate": 0.61
      }
    }
  },
  "comparisons": {
    "ailang_vs_python": {
      "token_reduction_pct": 13.2,
      "execution_speedup": 3.8,
      "success_rate_delta": -0.05
    }
  }
}
```

**Tool**: `tools/generate_performance_table.sh` (NEW, ~150 LOC)

Generates performance JSON from current eval results, including:
- 0-shot (first attempt only)
- 1-shot (with self-repair)
- Aggregates per model
- AILANG vs Python comparisons

**Tool**: `tools/update_performance_tables.sh` (NEW, ~100 LOC)

After validation, updates historical tables:
```bash
#!/usr/bin/env bash
# Update performance tables with latest results

VERSION="${1:-v0.3.0-alpha4}"

# Generate current performance
./tools/generate_performance_table.sh > "eval_results/performance_tables/${VERSION}.json"

# Update README badges
./tools/generate_badges.sh "eval_results/performance_tables/${VERSION}.json"

echo "✓ Performance table updated: ${VERSION}.json"
```

**Acceptance Criteria**:
- [ ] Performance tables track 0-shot, 1-shot metrics
- [ ] Historical versions stored in `performance_tables/`
- [ ] README badges auto-generated from latest table

---

### Phase 3: Enhancements (Optional, 6-8 hours)

#### Task 9: Root Cause Hypothesis (Gemini's Suggestion)

Add to `internal/eval_analyzer/design_generator.go`:

```go
type RootCauseHypothesis struct {
    Category    string  `json:"category"`    // "Prompt Ambiguity", "Language Design", "Compiler Bug"
    Confidence  float64 `json:"confidence"`  // 0.0-1.0
    Description string  `json:"description"`
}

// In GPT-5 prompt, add:
Additionally, hypothesize WHY the AI generated incorrect code:
1. Was the benchmark prompt ambiguous?
2. Is the AILANG syntax confusing or unintuitive?
3. Is the error message unclear?
4. Is this a compiler bug?

Provide a root cause hypothesis with confidence (0.0-1.0) and reasoning.
Output as JSON field: root_cause_hypothesis
```

This creates a **secondary feedback loop** for improving prompts/benchmarks.

#### Task 10: HTML Dashboard & Badges

Generate visual reports for marketing/stakeholders.

---

## File Structure

```
ailang/
├── internal/
│   └── eval_harness/
│       ├── errors.go           # NEW: Error taxonomy + repair hints
│       ├── errors_test.go      # NEW: Error matching tests
│       ├── prompt_loader.go    # NEW: Prompt versioning
│       ├── metrics.go          # MODIFIED: Add repair/versioning fields
│       └── runner.go           # MODIFIED: Add RunWithRepair()
│
├── prompts/
│   ├── versions.json           # NEW: Prompt metadata
│   ├── v0.3.0.md               # EXISTING
│   ├── v0.3.0-hints.md         # NEW: With explicit reminders
│   └── python.md               # EXISTING
│
├── tools/
│   ├── eval_prompt_ab.sh       # NEW: A/B test prompts
│   ├── compare_results.sh      # NEW: Compare two result dirs
│   ├── generate_summary_jsonl.sh  # NEW: JSONL format
│   ├── generate_matrix_json.sh    # NEW: Matrix pivot
│   ├── eval_store_baseline.sh     # NEW: Store baseline
│   ├── eval_diff.sh               # NEW: Compare vs baseline
│   ├── eval_validate_fix.sh       # NEW: Re-run failed benchmarks
│   ├── generate_performance_table.sh  # NEW: Performance JSON
│   └── update_performance_tables.sh   # NEW: Update history
│
├── eval_results/
│   ├── *.json                  # Individual run metrics
│   ├── summary.jsonl           # NEW: One-line per run
│   ├── matrix.json             # NEW: Pivoted view
│   ├── diff_report.md          # NEW: Comparison report
│   ├── baseline/               # NEW: Stored baseline
│   │   ├── *.json
│   │   ├── summary.jsonl
│   │   └── timestamp.txt
│   └── performance_tables/     # NEW: Historical tracking
│       ├── schema.md
│       ├── v0.3.0-alpha3.json
│       └── v0.3.0-alpha4.json
│
├── docs/guides/
│   ├── eval_workflow.md        # NEW: Complete workflow guide
│   └── performance_tracking.md # NEW: How to track evolution
│
├── cmd/ailang/
│   └── eval.go                 # MODIFIED: Add --self-repair, --prompt-version
│
└── Makefile                    # MODIFIED: Add 8 new targets
```

---

## Success Metrics

### Quantitative Metrics

#### 1. Self-Repair Effectiveness
- **Metric**: `repair_success_rate = repair_ok / (total_runs - first_attempt_ok)`
- **Target**: >50% of first-attempt failures fixed by single retry
- **Example**: "CAP_001 has 95% repair success rate" ✅

#### 2. Prompt Quality
- **Metric**: Success rate delta between prompt versions
- **Target**: `v0.3.0-hints` improves first-attempt by >5%
- **Example**: "Semicolon reminder reduces PAR_001 by 40%" ✅

#### 3. Language Evolution
- **Metric**: Success rate improvement after fix
- **Target**: Design doc → implementation → >10% improvement
- **Example**: "Float equality fix: 60% → 95% success" ✅

#### 4. Token Efficiency
- **Metric**: Median output tokens (AILANG vs Python)
- **Target**: Maintain <15% reduction
- **Example**: "AILANG: 13% fewer tokens than Python" ✅

#### 5. Error Taxonomy Coverage
- **Metric**: `categorized_errors / total_errors`
- **Target**: >80% of failures mapped to error codes
- **Example**: "PAR_001, TC_REC_001, EQ_001 cover 72% of failures" ✅

### Qualitative Metrics

#### 1. Developer Experience
- Can AI learn from error messages? (repair success rate)
- Are repair hints actionable? (manual review)
- Do prompts teach effectively? (A/B testing)

#### 2. Workflow Efficiency
- Time to validate fix: <5 minutes
- Time to run A/B test: <10 minutes
- Time to generate design doc: <2 minutes

#### 3. Stakeholder Visibility
- Performance tables show trends
- Badges show current status
- Diff reports show impact of fixes

---

## Performance Tracking Tables

### Table Schema

Each language version gets a performance table tracking:

**Dimensions**:
- Model: `gpt-4o-mini`, `claude-3.5-sonnet`, `gemini-2.0-flash-exp`
- Benchmark: `recursion_factorial`, `float_eq`, `records_person`, etc.
- Shot Type: `0-shot` (first attempt), `1-shot` (with self-repair)

**Metrics**:
- `success_rate`: % of passing runs
- `avg_tokens`: Mean output tokens
- `avg_ms`: Mean execution time
- `repair_success_rate`: % of first-failures fixed by repair

**Aggregates**:
- Per-model overall success rates (0-shot, 1-shot)
- AILANG vs Python comparisons (token reduction %, speedup)
- Top error codes by frequency

### Definitions

#### 0-Shot (First Attempt)
- AI receives only the benchmark prompt
- No error feedback or retry
- Measures: "Can AI use AILANG correctly from prompt alone?"
- Metric: `first_attempt_ok` field

#### 1-Shot (Self-Repair)
- AI receives error message + repair guidance
- Single retry allowed
- Measures: "Can AI learn from AILANG error messages?"
- Metric: `repair_ok` field

#### N-Shot (Future Enhancement)
- Multiple retries with accumulated context
- Measures: "Can AI iteratively refine AILANG code?"
- Not implemented in Phase 1

### Example Table Structure

**File**: `eval_results/performance_tables/v0.3.0-alpha4.json`

```json
{
  "version": "v0.3.0-alpha4",
  "date": "2025-10-08",
  "prompt_version": "v0.3.0-hints",

  "models": {
    "gpt-4o-mini": {
      "benchmarks": {
        "recursion_factorial": {
          "0-shot": {
            "success_rate": 1.0,
            "avg_tokens_out": 83,
            "avg_ms": 56,
            "runs": 5
          },
          "1-shot": {
            "success_rate": 1.0,
            "avg_tokens_out": 44,
            "avg_repair_tokens": 44,
            "avg_ms": 88,
            "runs": 5
          }
        },
        "float_eq": {
          "0-shot": {
            "success_rate": 0.6,
            "avg_tokens_out": 95,
            "avg_ms": 48,
            "runs": 5
          },
          "1-shot": {
            "success_rate": 0.95,
            "avg_tokens_out": 52,
            "avg_repair_tokens": 48,
            "avg_ms": 96,
            "runs": 5,
            "repair_success_rate": 0.875
          }
        }
      },
      "aggregates": {
        "total_benchmarks": 12,
        "0-shot_success": 0.85,
        "1-shot_success": 0.92,
        "avg_tokens_out": 89,
        "repair_success_rate": 0.61,
        "top_errors": [
          {"code": "EQ_001", "count": 8, "repair_success": 0.90},
          {"code": "CAP_001", "count": 5, "repair_success": 0.95},
          {"code": "PAR_001", "count": 3, "repair_success": 0.85}
        ]
      }
    },
    "claude-3.5-sonnet": {
      "benchmarks": { /* ... */ },
      "aggregates": { /* ... */ }
    }
  },

  "comparisons": {
    "ailang_vs_python": {
      "token_reduction_pct": 13.2,
      "execution_speedup": 3.8,
      "success_rate_delta": -0.05,
      "note": "AILANG generates 13% fewer tokens, runs 3.8x faster, but has 5% lower success (teaching phase)"
    }
  },

  "prompt_experiments": [
    {
      "experiment": "baseline_vs_hints",
      "versions": ["v0.3.0-baseline", "v0.3.0-hints"],
      "result": {
        "baseline_success": 0.85,
        "hints_success": 0.92,
        "delta": 0.07,
        "conclusion": "Explicit reminders improve first-attempt by 7%"
      }
    }
  ]
}
```

### Updating Tables

**After implementing a design doc:**

```bash
# 1. Store baseline before fix
make eval-baseline

# 2. Implement fix
# ... edit code ...

# 3. Validate fix
make eval-validate-fix

# 4. Update performance table
./tools/update_performance_tables.sh v0.3.0-alpha4

# 5. Review changes
git diff eval_results/performance_tables/
```

**What gets updated:**
- Success rates for affected benchmarks
- Aggregate metrics (if overall improvement)
- Comparisons (if AILANG vs Python changes)
- Top errors (if new error patterns emerge)

### Visualizing Trends

Future enhancement: Generate charts showing:
- Success rate over time (v0.2.0 → v0.3.0 → v0.4.0)
- Token efficiency evolution
- Repair success rate by error code
- Model-specific improvements

---

## Usage Examples

### Example 1: Run Evals with Self-Repair

```bash
# Basic run
make eval-suite

# With self-repair enabled
make eval-suite-repair

# Check results
cat eval_results/summary.jsonl | jq 'select(.repair_used == true)'
# Output: Shows all runs that used repair
```

### Example 2: A/B Test Prompt Versions

```bash
# Compare baseline vs hints
make eval-prompt-test

# Output:
# Comparing prompt versions: v0.3.0-baseline vs v0.3.0-hints
# Running evals with v0.3.0-baseline...
# Running evals with v0.3.0-hints...
#
# ===== COMPARISON =====
# Directory A: eval_results/v0.3.0-baseline
#   Success: 17/20 (85.0%)
#   Avg Output Tokens: 94.2
#
# Directory B: eval_results/v0.3.0-hints
#   Success: 18/20 (92.0%)
#   Avg Output Tokens: 92.1
#
# ✅ Version B improved by 7.0%
```

### Example 3: Validate Fix Workflow

```bash
# 1. Run initial evals
make eval-suite-repair

# 2. Analyze failures → generate design docs
make eval-analyze
# Output: Generated design_docs/planned/EVAL_ANALYSIS_float_equality.md

# 3. Store baseline
make eval-baseline

# 4. Implement fix (manual)
# ... edit internal/eval/builtins.go ...
make test && make lint

# 5. Validate fix
make eval-validate-fix BENCH=float_eq

# Output:
# 📊 Running: float_eq
# ✓ float_eq: 3/5 passed (60%)
#
# Comparing current vs baseline...
# ✓ Diff report: eval_results/diff_report.md
#
# # Eval Diff Report
#
# ## Improvements ✅
# - float_eq (gpt-4o-mini, ailang): failed → passed
# - float_eq (gpt-4o-mini, ailang): failed → passed
#
# ## Regressions ⚠️
# (none)
#
# ✅ Validation complete.

# 6. Update performance table
./tools/update_performance_tables.sh v0.3.0-alpha4
```

### Example 4: Track Performance Over Time

```bash
# View current performance
cat eval_results/performance_tables/v0.3.0-alpha4.json | jq '.models["gpt-4o-mini"].aggregates'

# Output:
# {
#   "0-shot_success": 0.92,
#   "1-shot_success": 0.95,
#   "repair_success_rate": 0.68,
#   "avg_tokens_out": 87,
#   "top_errors": [
#     {"code": "EQ_001", "count": 2, "repair_success": 1.0},
#     {"code": "CAP_001", "count": 1, "repair_success": 1.0}
#   ]
# }

# Compare versions
diff \
  <(jq '.models["gpt-4o-mini"].aggregates."0-shot_success"' eval_results/performance_tables/v0.3.0-alpha3.json) \
  <(jq '.models["gpt-4o-mini"].aggregates."0-shot_success"' eval_results/performance_tables/v0.3.0-alpha4.json)

# Output: 0.85 → 0.92 (+7% improvement)
```

---

## Risk Assessment

### Technical Risks

**Risk 1: Repair Hints Too Generic**
- **Issue**: AI might not understand vague guidance
- **Mitigation**: Test hints empirically, refine based on repair success rate
- **Severity**: Medium (reduces self-repair effectiveness)

**Risk 2: Prompt Hash Instability**
- **Issue**: Whitespace changes invalidate hashes
- **Mitigation**: Document that prompts are versioned intentionally
- **Severity**: Low (just re-hash after changes)

**Risk 3: Baseline Drift**
- **Issue**: AI models update, changing baseline
- **Mitigation**: Store model API version in metadata
- **Severity**: Medium (affects long-term comparisons)

**Risk 4: Error Taxonomy Maintenance**
- **Issue**: New error types emerge as language evolves
- **Mitigation**: Review uncategorized errors monthly, expand taxonomy
- **Severity**: Medium (gaps reduce repair coverage)

### Process Risks

**Risk 5: Over-Reliance on Repair**
- **Issue**: Accepting 0-shot failures because repair fixes them
- **Mitigation**: Track 0-shot success separately; aim to improve both
- **Severity**: Low (metrics keep us honest)

**Risk 6: Prompt Overfitting**
- **Issue**: Optimizing prompts for specific benchmarks
- **Mitigation**: Add new benchmarks regularly, test on unseen tasks
- **Severity**: Medium (reduces generalization)

---

## Timeline & Effort

### Phase 1: Core Loop
- **Task 1**: Error Taxonomy - 2-3 hours
- **Task 2**: Self-Repair Loop - 3-4 hours
- **Task 3**: Prompt Versioning - 2-3 hours
- **Task 4**: AI-Friendly Formats - 2-3 hours
- **Task 5**: Validation Loop - 2-3 hours
- **Task 6**: Makefile Targets - 0.5 hours
- **Task 7**: Documentation - 1-2 hours

**Total Phase 1**: 12-16 hours (1.5-2 days)

### Phase 2: Performance Tracking
- **Task 8**: Performance Tables - 4-6 hours

**Total Phase 2**: 4-6 hours (0.5-1 day)

### Phase 3: Enhancements (Optional)
- **Task 9**: Root Cause Hypothesis - 2-3 hours
- **Task 10**: HTML Dashboard - 4-6 hours

**Total Phase 3**: 6-9 hours (1 day)

### **Grand Total**: 22-31 hours (3-4 days)

**Recommended: Start with Phase 1 only (1.5-2 days)**

---

## Dependencies

### Prerequisites
- ✅ Eval harness working (`internal/eval_harness/`)
- ✅ Design doc generation working (`internal/eval_analyzer/`)
- ✅ Multi-model support (OpenAI, Anthropic, Google)
- ✅ Makefile targets for eval suite

### External Dependencies
- None (all tools are bash + jq + Go)

### Soft Dependencies
- GPT-5 API access (for design doc generation)
- Multiple AI model API keys (for multi-model testing)

---

## Success Criteria

### Phase 1 Acceptance
- [ ] `ailang eval --self-repair` retries failed runs with error guidance
- [ ] Error taxonomy covers 80%+ of failures (6+ error codes)
- [ ] Repair success rate >50% for categorized errors
- [ ] `--prompt-version` flag loads different prompt versions
- [ ] `make eval-prompt-test` compares two prompts side-by-side
- [ ] `make eval-baseline` stores snapshot
- [ ] `make eval-diff` shows improvements/regressions
- [ ] `make eval-validate-fix` re-runs failed benchmarks
- [ ] `summary.jsonl` and `matrix.json` generated correctly
- [ ] Documentation covers complete workflow with examples

### Phase 2 Acceptance
- [ ] Performance tables track 0-shot, 1-shot metrics per model
- [ ] Historical tables stored in `performance_tables/`
- [ ] Aggregates include repair success rate by error code
- [ ] AILANG vs Python comparisons included
- [ ] `update_performance_tables.sh` updates history after fixes

### Long-Term Success
- **After 1 month**: 3+ design docs implemented with validated improvements
- **After 3 months**: Repair success rate >60%, 0-shot success >90%
- **After 6 months**: Performance tables show consistent upward trends

---

## Appendix A: Documentation Content

### `docs/guides/eval_workflow.md`

```markdown
# AI Eval Feedback Loop: Complete Workflow

This guide covers the complete workflow for closing the loop: eval → analyze → design → implement → validate.

## Quick Start (5 Steps)

### 1. Run Evals
```bash
make eval-suite                # Basic run
make eval-suite-repair         # With self-repair (recommended)
```

**Output**: JSON files in `eval_results/`, one per (benchmark × model × lang)

### 2. Analyze Failures
```bash
make eval-analyze              # Generate design docs from failures
```

**Output**: Design docs in `design_docs/planned/EVAL_ANALYSIS_*.md`

**Review**: Open generated docs, assess priority, iterate with AI

### 3. Store Baseline
```bash
make eval-baseline             # Snapshot current state
```

**Output**: Baseline stored in `eval_results/baseline/`

**Why**: Enables before/after comparison to validate fixes

### 4. Implement Fixes
(Manual step - implement based on design docs)

```bash
# Example: Fix float equality bug
vim internal/eval/builtins.go
make test && make lint
```

### 5. Validate Fixes
```bash
make eval-validate-fix         # Re-run failed benchmarks
make eval-diff                 # Compare before/after
```

**Output**: `eval_results/diff_report.md` shows improvements

**Update**: If successful, update performance tables:
```bash
./tools/update_performance_tables.sh v0.3.0-alpha4
```

---

## Self-Repair Usage

### Basic Self-Repair
```bash
ailang eval --benchmark float_eq --self-repair
```

**What happens:**
1. AI generates code (first attempt)
2. If fails: Extract error code (e.g., `EQ_001`)
3. Inject repair hint: "Float equality dictionary must match Float"
4. AI retries ONCE
5. Metrics track: `first_attempt_ok`, `repair_ok`

### Self-Repair Metrics
```bash
# View repair results
cat eval_results/summary.jsonl | jq 'select(.repair_used == true)'

# Calculate repair success rate
jq -s '
  map(select(.repair_used == true)) |
  {
    total: length,
    successes: map(select(.repair_ok == true)) | length,
    rate: (map(select(.repair_ok == true)) | length) / length
  }
' eval_results/summary.jsonl
```

### Error Code Reference

| Code | Description | Repair Success Rate |
|------|-------------|---------------------|
| `PAR_001` | Block/semicolon syntax | 85% |
| `TC_REC_001` | Record field missing | 70% |
| `EQ_001` | Float equality dictionary | 90% |
| `CAP_001` | Missing capability | 95% |
| `TC_INT_001` | Modulo on Float | 80% |
| `MOD_001` | Entrypoint not found | 60% |

See `internal/eval_harness/errors.go` for full taxonomy.

---

## Prompt A/B Testing

### Compare Two Prompt Versions
```bash
make eval-prompt-test  # Default: baseline vs hints
```

### Compare Specific Versions
```bash
make eval-prompt-compare A=v0.3.0-baseline B=v0.3.0-hints
```

**Output:**
```
Comparing prompt versions: v0.3.0-baseline vs v0.3.0-hints
Running evals with v0.3.0-baseline...
Running evals with v0.3.0-hints...

===== COMPARISON =====
Directory A: eval_results/v0.3.0-baseline
  Success: 17/20 (85.0%)
  Avg Output Tokens: 94.2

Directory B: eval_results/v0.3.0-hints
  Success: 18/20 (92.0%)
  Avg Output Tokens: 92.1

✅ Version B improved by 7.0%
```

### Creating New Prompt Version

1. **Copy existing prompt:**
   ```bash
   cp prompts/v0.3.0.md prompts/v0.3.0-myversion.md
   ```

2. **Edit prompt** (add reminders, examples, etc.)

3. **Calculate hash:**
   ```bash
   shasum -a 256 prompts/v0.3.0-myversion.md
   ```

4. **Update `prompts/versions.json`:**
   ```json
   {
     "v0.3.0-myversion": {
       "desc": "My experimental prompt changes",
       "file": "v0.3.0-myversion.md",
       "hash": "sha256:<hash-from-step-3>"
     }
   }
   ```

5. **Test:**
   ```bash
   ailang eval --benchmark test --prompt-version v0.3.0-myversion
   ```

---

## Validation Workflow

### Complete Validation Example

```bash
# 1. Initial state
make eval-suite-repair
# Result: 17/20 passing (85%)

# 2. Analyze failures
make eval-analyze
# Output: design_docs/planned/EVAL_ANALYSIS_float_equality.md

# 3. Store baseline
make eval-baseline

# 4. Implement fix
vim internal/eval/builtins.go
# ... fix eq_Float dictionary ...
make test && make lint

# 5. Validate specific benchmark
make eval-validate-fix BENCH=float_eq

# Output:
# 📊 Running: float_eq
# ✓ float_eq: 5/5 passed (100%)
#
# Comparing current vs baseline...
#
# # Eval Diff Report
#
# ## Improvements ✅
# - float_eq (gpt-4o-mini, ailang): failed → passed (3 instances)
#
# ## Regressions ⚠️
# (none)

# 6. Update performance table
./tools/update_performance_tables.sh v0.3.0-alpha4

# 7. Commit
git add internal/eval/ eval_results/performance_tables/
git commit -m "Fix: Float equality dictionary (60% → 100% success)"
```

### Re-run All Failed Benchmarks

```bash
# Without specifying BENCH, re-runs all previously failed benchmarks
make eval-validate-fix
```

### Diff Report Contents

`eval_results/diff_report.md` includes:
- **Improvements**: Benchmarks that now pass
- **Regressions**: Benchmarks that now fail
- **Token Changes**: Significant token count changes (>5%)

---

## Performance Tracking

### View Current Performance

```bash
cat eval_results/performance_tables/v0.3.0-alpha4.json | jq '.models["gpt-4o-mini"].aggregates'
```

**Output:**
```json
{
  "0-shot_success": 0.92,
  "1-shot_success": 0.95,
  "repair_success_rate": 0.68,
  "avg_tokens_out": 87,
  "top_errors": [
    {"code": "EQ_001", "count": 2, "repair_success": 1.0},
    {"code": "CAP_001", "count": 1, "repair_success": 1.0}
  ]
}
```

### Compare Versions

```bash
# Before (v0.3.0-alpha3)
jq '.models["gpt-4o-mini"].aggregates."0-shot_success"' \
  eval_results/performance_tables/v0.3.0-alpha3.json
# Output: 0.85

# After (v0.3.0-alpha4)
jq '.models["gpt-4o-mini"].aggregates."0-shot_success"' \
  eval_results/performance_tables/v0.3.0-alpha4.json
# Output: 0.92

# Improvement: +7%
```

### Metrics Definitions

- **0-shot**: First attempt only (no error feedback)
  - Measures: "Can AI use AILANG from prompt alone?"
- **1-shot**: With self-repair (error feedback + retry)
  - Measures: "Can AI learn from AILANG error messages?"
- **Repair Success Rate**: `repair_ok / repair_used`
  - Measures: "How effective are error messages?"

---

## Troubleshooting

### "No baseline found"
```bash
# Solution: Store baseline first
make eval-baseline
```

### "Prompt version not found"
```bash
# Solution: Check available versions
cat prompts/versions.json | jq '.versions | keys'
```

### "Repair hints not triggering"
```bash
# Check if errors are categorized
cat eval_results/summary.jsonl | jq 'select(.err_code != null)'

# If empty, errors are not matching taxonomy
# Review internal/eval_harness/errors.go patterns
```

---

## Best Practices

1. **Store baseline before every fix** - Enables validation
2. **Run self-repair by default** - Measures teachability
3. **A/B test prompt changes** - Isolate what works
4. **Update performance tables after validation** - Track progress
5. **Review uncategorized errors monthly** - Expand taxonomy
6. **Keep benchmarks up-to-date** - Add new test cases

---

## Example Session

```bash
# Day 1: Initial eval
make eval-suite-repair
make eval-analyze
make eval-baseline
# Review design docs, prioritize fixes

# Day 2: Implement float equality fix
vim internal/eval/builtins.go
make test && make lint
make eval-validate-fix BENCH=float_eq
# Success: 60% → 100%

# Day 3: Implement capability detection fix
vim internal/runtime/capabilities.go
make test && make lint
make eval-validate-fix BENCH=test_io
# Success: 80% → 95%

# Day 4: Update performance table & commit
./tools/update_performance_tables.sh v0.3.0-alpha4
git add -A
git commit -m "Eval loop: Float equality + capability fixes (+15% success)"

# Week 2: Test new prompt version
cp prompts/v0.3.0.md prompts/v0.3.0-refined.md
# ... edit prompt ...
make eval-prompt-compare A=v0.3.0-baseline B=v0.3.0-refined
# Result: +8% improvement → Adopt new prompt
```
```

---

## Changelog

**v1.0 (2025-10-08)**
- Initial design document
- Incorporates user feedback (error taxonomy table, minimal retry, performance tracking)
- Incorporates AI feedback (GPT-5: tutor loop; Gemini: root cause hypothesis)
- Defines 3-phase implementation plan
- Documents performance tracking tables with 0-shot, 1-shot metrics

---

**Status**: 📋 Ready for implementation
**Next Steps**: Review plan → Start Phase 1 Task 1 (Error Taxonomy) → Iterate
