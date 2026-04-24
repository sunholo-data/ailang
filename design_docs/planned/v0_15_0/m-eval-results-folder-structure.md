# M-EVAL-RESULTS-FOLDER-STRUCTURE: Model-First Eval Results Layout

**Status**: Planned
**Target**: v0.15.x
**Priority**: P2 — Medium
**Estimated**: 1 day
**Dependencies**: None

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Path is deterministic: mode/model/benchmark_lang_ts.json |
| A2: Replayability | +1 | Cleaner structure makes replaying a specific model run easier |
| A3: Effect Legibility | 0 | No effect changes |
| A4: Explicit Authority | 0 | No authority changes |
| A5: Bounded Verification | 0 | No verification changes |
| A6: Safe Concurrency | 0 | Parallel eval still writes distinct files |
| A7: Machines First | +1 | `find agent/claude-sonnet-4-6/` is faster than grepping 1500-file flat dir |
| A8: Minimal Syntax | 0 | CLI flags unchanged |
| A9: Cost Visibility | 0 | Cost fields stay in JSON content |
| A10: Composability | 0 | No interface changes |
| A11: Structured Failure | +1 | skip-existing now reads from authoritative path — no silent wrong-glob |
| A12: System Boundary | 0 | No boundary changes |

**Net score: +4** ✅ No hard violations.

---

## Problem Statement

The eval results directory has two problems:

### 1. Flat model-mixed layout doesn't scale

Current structure puts all results for all models in one directory:
```
eval_results/v0141_full/agent/
  api_call_json_ailang_claude-sonnet-4-6_1776948312.json
  api_call_json_ailang_opencode-haiku_1777009226.json
  api_call_json_ailang_gpt5-4_1776949193.json
  ... (494 files, 8 models, all mixed)
```

At 8 models × 22 benchmarks × 4 langs = ~700 files per eval run. Projecting to 20+ models (Ollama local + future providers): **1,800+ files in one flat dir**. `ls`, `grep`, and shell globbing all degrade.

### 2. ~70 old test/experiment directories pollute eval_results/

Old runs from `benchmark_test`, `cost_test`, `agent_showcase`, `pilot_final_v2` etc. are still present and consume ~1GB of disk, contaminate `find` results, and confuse analysis scripts that walk the tree.

---

## Goal

**Primary**: Restructure new writes to `agent/{model}/` and `standard/{model}/` subdirectories.

**Secondary**: Delete or archive old experiment directories; add `.gitignore` rules to keep `eval_results/` clean going forward.

**Success metrics**:
- `ls eval_results/v0141_full/agent/` shows model subdirs, not 500 files
- `ailang eval-suite -skip-existing` correctly skips existing results in new layout
- `ailang eval-report` produces identical JSON output from new layout
- All eval-analyzer scripts pass `make test` with new layout
- 70+ old dirs deleted, `eval_results/` ≤ 5 top-level entries

---

## Current State: Full Consumer Audit

### Writers

| File | What it does | Change needed |
|------|-------------|---------------|
| `internal/eval_harness/metrics.go:155` | Writes `{outputDir}/agent/{bench}_{lang}_{model}_{ts}.json` | Add `m.Model` subdir before filename; drop model from filename |

### Readers

| File | How it reads | Change needed |
|------|-------------|---------------|
| `internal/eval_analysis/loader.go:23` | `filepath.WalkDir(dir, ...)` — recurses all subdirs | **None** — already handles nested dirs |
| `cmd/ailang/eval_suite.go:460` | `filepath.Glob("{outputDir}/agent/{bench}_{lang}_{model}_*.json")` — skip-existing | **Must update** glob pattern to `{outputDir}/agent/{model}/{bench}_{lang}_*.json` |
| `.claude/skills/post-release/scripts/run_eval_baseline.sh:363` | `find "$RESULTS_DIR/agent" -name "*.json"` — reads `.id` from JSON | **None** — `find` recurses, reads from JSON content not filename |
| `.claude/skills/eval-analyzer/scripts/examine_code.sh:38` | `find "$BASELINE_DIR" -name "{bench}_ailang_{model}_*.json"` — model in glob | **Must update** to `find "$BASELINE_DIR/agent/{model}" -name "{bench}_ailang_*.json"` |
| `.claude/skills/eval-analyzer/scripts/analyze_failures.sh` | Reads from `summary.jsonl` via `eval-summary` CLI | **None** — depends on loader, not filenames |
| `.claude/skills/eval-analyzer/scripts/compare_models.sh` | Reads from `summary.jsonl` via `eval-summary` CLI | **None** — depends on loader, not filenames |
| `.claude/skills/eval-analyzer/scripts/find_language_gaps.sh` | `find` + reads JSON `.lang`, `.stdout_ok` fields | **None** — uses `find` recursively + JSON content |
| `.claude/skills/eval-analyzer/scripts/benchmark_health.sh` | Walks dir, reads JSON fields | **None** — content-based |
| `.claude/skills/eval-analyzer/scripts/fair_comparison.py` | Loads from `standard/` subdir via `os.listdir` | **None if** we keep `standard/` flat (only agent gets model subdirs for now), or **minor update** if both change |
| `.claude/skills/eval-analyzer/scripts/quick_summary.sh` | Uses `eval-summary` CLI | **None** |

### Summary: 2 files need code changes, rest are safe

---

## Solution Design

### New directory layout

```
eval_results/{version}/
  agent/
    {model}/                          ← NEW: one subdir per model
      {bench}_{lang}_{ts}.json        ← model removed from filename
  standard/
    {model}/                          ← NEW (parallel to agent)
      {bench}_{lang}_{ts}.json
```

Model name is still available via the `"model"` field inside each JSON file — it doesn't need to be in the filename.

### Migration for existing flat dirs

Existing `v0141_full/agent/*.json` files stay as-is — they are read by `WalkDir` which recurses into new subdirs AND reads the flat files in the root. No migration required for reads. The only issue is skip-existing will not find them at the new path — but since existing results were written flat, re-running with `-skip-existing` on a new-layout run will correctly NOT skip them (fresh run), which is the safe behavior.

If a mid-run resume is needed for an existing flat layout, run the migration script (see below) first.

### Files to change

#### `internal/eval_harness/metrics.go`

```go
// Before (line ~161):
if m.Condition != "" {
    targetDir = filepath.Join(targetDir, m.Condition)
}

// After:
sanitizedModel := strings.ReplaceAll(m.Model, ":", "_")
targetDir = filepath.Join(targetDir, sanitizedModel)
if m.Condition != "" {
    targetDir = filepath.Join(targetDir, m.Condition)
}
```

Filename drops model (already stored in JSON body):
```go
// Before:
filename := fmt.Sprintf("%s_%s_%s_%d.json", m.ID, m.Lang, sanitizedModel, m.Timestamp.Unix())

// After:
filename := fmt.Sprintf("%s_%s_%d.json", m.ID, m.Lang, m.Timestamp.Unix())
```

#### `cmd/ailang/eval_suite.go` — skip-existing (line ~471)

```go
// Before:
patterns = append(patterns, filepath.Join(*outputDir, modeDir, fmt.Sprintf("%s_%s_%s_*.json", benchmark, lang, model)))

// After:
sanitizedModel := strings.ReplaceAll(model, ":", "_")
patterns = append(patterns, filepath.Join(*outputDir, modeDir, sanitizedModel, fmt.Sprintf("%s_%s_*.json", benchmark, lang)))
// Keep legacy flat fallback for backward compat with old result dirs:
patterns = append(patterns, filepath.Join(*outputDir, modeDir, fmt.Sprintf("%s_%s_%s_*.json", benchmark, lang, model)))
```

#### `.claude/skills/eval-analyzer/scripts/examine_code.sh`

```bash
# Before:
PATTERN="${BASELINE_DIR}/${BENCH_ID}_ailang_${MODEL}_*.json"
FILES=$(find "$BASELINE_DIR" -name "${BENCH_ID}_ailang_*.json" ...)

# After:
PATTERN="${BASELINE_DIR}/agent/${MODEL}/${BENCH_ID}_ailang_*.json"
FILES=$(find "$BASELINE_DIR/agent/${MODEL}" -name "${BENCH_ID}_ailang_*.json" ...)
# Fallback to flat layout if model subdir not found:
if [ -z "$FILES" ]; then
  FILES=$(find "$BASELINE_DIR" -name "${BENCH_ID}_ailang_*.json" ...)
fi
```

### Old directory cleanup

**Delete outright** (old experiment noise, safe to remove):
```
agent-runs/ agent_comparison/ agent_fizzbuzz/ agent_showcase/
agent_test/ agent_test_20251028_072441/ agent_timeout_test/
benchmark_test{,2,3}/ chain_test{,2,3,4}/ changed_benchmarks/
cost_test{,2,3,4}/ dashboard_test/ dev_v0.6.5_chars/
full_test_20251028_093739/ haiku_v0.4.0_prompt/ haiku_v0.4.1_verify/
harness_smoke_v2/ hierarchy_test/ linkedin_validation/ microrag_ab/
model_test_gpt5-1_gemini-3-pro/ new_benchmarks/ performance_tables/
pilot_final_v2/ pilot_stream_json_final/ pilot_streaming_final/
prompt_test_v0.3.19/ prompt_test_v0.4.0/ prompt_test_v0.4.4/
race_condition_test/ session_test/ split_validation{,_v2,_v3}/
standard/ string_split_python_compare/ string_split_test/ summary.jsonl
test_caps_verification/ test_floatToInt{,_v2}/ test_json_helpers/
test_json_parse/ test_tree_fix/ test_v0.3.17_{final,quick,v2}/
test_v0.3.18_quick/ test_v0.4.1_http_fix{,_gpt}/ test_v0.4.1_{phase2,simplified}/
test_v0.4.8{,b,c,d}/ trace_test/ v0.3.12_fixed/ v0.3.21_{dev,quick}_test/
v0.3.9_{gemini_no_repair,gemini_only,retest}/ v0.6.5-{g3-haiku,gaps,gaps-2model,v2}/
validation/ WITH_WARNING/
```

**Keep**:
```
eval_results/
  baselines/       ← historical baselines used by eval-report history
  agent/           ← older agent runs (pre-v0141), useful for trend data
  v0141_full/      ← current active eval run
  v0141_core/      ← keep if contains distinct data
  v0141_stretch/   ← keep
  v0141_vision/    ← keep
```

**Add `.gitignore` rules**:
```gitignore
# eval_results: only track baselines and named version dirs
eval_results/*_test*/
eval_results/*_test/
eval_results/agent-runs/
eval_results/cost_test*/
eval_results/benchmark_test*/
```

### Migration script

```bash
#!/bin/bash
# tools/migrate_eval_results.sh
# Reorganizes flat agent/standard dirs into model subdirs
# Usage: ./tools/migrate_eval_results.sh eval_results/v0141_full

DIR="${1:?usage: $0 <eval_dir>}"

for mode in agent standard; do
  FLAT="$DIR/$mode"
  [ -d "$FLAT" ] || continue

  find "$FLAT" -maxdepth 1 -name "*.json" | while read -r f; do
    base=$(basename "$f" .json)
    # Extract model from filename: {bench}_{lang}_{model}_{ts}
    model=$(echo "$base" | rev | cut -d_ -f2 | rev)
    # Handle multi-part model names (opencode-gemini-3-flash etc)
    # Model is everything between lang and timestamp
    ts=$(echo "$base" | rev | cut -d_ -f1 | rev)
    # Remove bench_ prefix and _ts suffix to get lang_model
    without_ts="${base%_$ts}"
    bench_lang=$(echo "$without_ts" | awk -F_ '{print $1"_"$2}')
    lang_model="${without_ts#${bench_lang}_}"
    model="${lang_model#*_}"  # everything after first lang segment

    MODEL_DIR="$FLAT/$model"
    mkdir -p "$MODEL_DIR"
    mv "$f" "$MODEL_DIR/"
    echo "Moved $base → $mode/$model/"
  done
done
echo "Migration complete."
```

---

## Implementation Plan

### M1: Audit + tests (0.5 days)

- [ ] Confirm all consumers listed above (re-grep after any new scripts added)
- [ ] Write a test in `internal/eval_harness/metrics_test.go` that verifies new path format
- [ ] Write a test in `cmd/ailang/eval_suite_flags_test.go` that verifies skip-existing finds files in model subdir AND legacy flat dir

### M2: Update writer + skip-existing (0.25 days)

- [ ] Edit `internal/eval_harness/metrics.go` — add model subdir, drop model from filename
- [ ] Edit `cmd/ailang/eval_suite.go` — update skip-existing glob + keep legacy fallback
- [ ] `make test` passes

### M3: Update scripts + cleanup (0.25 days)

- [ ] Edit `.claude/skills/eval-analyzer/scripts/examine_code.sh` — model-aware path + fallback
- [ ] Run deletion script for old experiment dirs (list above)
- [ ] Add `.gitignore` rules
- [ ] Run migration script on `v0141_full/` if mid-run resume needed

---

## Success Criteria

- [ ] `ls eval_results/v0141_full/agent/` shows model subdirs (e.g., `claude-sonnet-4-6/`, `opencode-haiku/`)
- [ ] `ailang eval-suite -skip-existing` correctly skips results in new model-subdir layout
- [ ] `ailang eval-suite -skip-existing` still correctly skips results in old flat layout (backward compat)
- [ ] `ailang eval-report eval_results/v0141_full v0.14.1 --format=json` produces identical output to current
- [ ] `make test` passes
- [ ] 70+ old directories deleted from `eval_results/`
- [ ] `.gitignore` prevents test dirs from accumulating again

---

## Related Documents

- [M-EVAL-CROSS-HARNESS](m-eval-cross-harness-sprint-plan.md) — adds `model_family` to results (separate concern)
- [M-EVAL-LANG-JSGO](m-eval-lang-jsgo-sprint-plan.md) — adds go/js language support
- [M-CLOUD-EVAL-WORKERS](../v0_13_0/m-cloud-eval-workers.md) — cloud eval (worker path changes would compound with this)
