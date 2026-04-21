# Common jq Queries for Eval Analysis

This reference provides additional jq queries beyond those in SKILL.md.

## Setup

All queries assume you have the summary.jsonl file:

```bash
SUMMARY=eval_results/baselines/v0.3.16/summary.jsonl

# Generate if missing
ailang eval-summary eval_results/baselines/v0.3.16
```

## Advanced Queries

### Benchmark-Specific Analysis

**All models that succeeded on a specific benchmark:**
```bash
jq -s --arg bench "fizzbuzz" '
  map(select(.id == $bench and .lang == "ailang" and .stdout_ok)) |
  map(.model)
' $SUMMARY
```

**Failure reasons for a specific benchmark:**
```bash
jq -s --arg bench "fizzbuzz" '
  map(select(.id == $bench and .lang == "ailang" and .stdout_ok == false)) |
  group_by(.error_category) |
  map({
    error: .[0].error_category,
    models: map(.model),
    count: length
  })
' $SUMMARY
```

### Model-Specific Analysis

**All benchmarks a model failed:**
```bash
jq -s --arg model "gpt5" '
  map(select(.model == $model and .lang == "ailang" and .stdout_ok == false)) |
  map(.id) |
  unique
' $SUMMARY
```

**Model's error distribution:**
```bash
jq -s --arg model "gpt5" '
  map(select(.model == $model and .lang == "ailang" and .stdout_ok == false)) |
  group_by(.error_category) |
  map({
    category: .[0].error_category,
    count: length,
    benchmarks: map(.id) | unique
  })
' $SUMMARY
```

### Cost Analysis

**Cost per benchmark:**
```bash
jq -s '
  map(select(.lang == "ailang")) |
  group_by(.id) |
  map({
    benchmark: .[0].id,
    total_cost: (map(.cost_usd) | add * 100 | round / 100),
    avg_cost: (map(.cost_usd) | add / length * 1000 | round / 1000)
  }) |
  sort_by(-.total_cost)
' $SUMMARY
```

**Most expensive failures:**
```bash
jq -s '
  map(select(.lang == "ailang" and .stdout_ok == false)) |
  sort_by(-.cost_usd) |
  .[:10] |
  map({
    benchmark: .id,
    model: .model,
    cost: (.cost_usd * 100 | round / 100),
    tokens: .total_tokens,
    error: .error_category
  })
' $SUMMARY
```

### Time Analysis

**Slowest runs:**
```bash
jq -s '
  map(select(.lang == "ailang")) |
  sort_by(-.duration_ms) |
  .[:10] |
  map({
    benchmark: .id,
    model: .model,
    duration_ms: .duration_ms,
    success: .stdout_ok
  })
' $SUMMARY
```

**Average duration by model:**
```bash
jq -s '
  map(select(.lang == "ailang")) |
  group_by(.model) |
  map({
    model: .[0].model,
    avg_duration_ms: (map(.duration_ms) | add / length | round)
  }) |
  sort_by(-.avg_duration_ms)
' $SUMMARY
```

### Repair Analysis

**Repair success by error category:**
```bash
jq -s '
  map(select(.lang == "ailang" and .repair_used == true)) |
  group_by(.error_category) |
  map({
    category: .[0].error_category,
    total_repairs: length,
    successful: map(select(.repair_ok)) | length,
    success_rate: (map(select(.repair_ok)) | length) / length * 100 | round
  })
' $SUMMARY
```

**Which models benefit most from repair:**
```bash
jq -s '
  map(select(.lang == "ailang")) |
  group_by(.model) |
  map({
    model: .[0].model,
    first_attempt_rate: (map(select(.first_attempt_ok)) | length) / length * 100 | round,
    final_rate: (map(select(.stdout_ok)) | length) / length * 100 | round,
    improvement: ((map(select(.stdout_ok)) | length) - (map(select(.first_attempt_ok)) | length))
  }) |
  sort_by(-.improvement)
' $SUMMARY
```

### Correlation Analysis

**Benchmarks where all models fail:**
```bash
jq -s '
  map(select(.lang == "ailang")) |
  group_by(.id) |
  map({
    benchmark: .[0].id,
    success_count: map(select(.stdout_ok)) | length,
    total: length
  }) |
  map(select(.success_count == 0))
' $SUMMARY
```

**Benchmarks with high model variance:**
```bash
jq -s '
  map(select(.lang == "ailang")) |
  group_by(.id) |
  map({
    benchmark: .[0].id,
    success_by_model: (
      group_by(.model) |
      map({
        model: .[0].model,
        success: map(select(.stdout_ok)) | length > 0
      })
    ),
    variance: (
      group_by(.model) |
      map(map(select(.stdout_ok)) | length) |
      (max - min)
    )
  }) |
  sort_by(-.variance) |
  .[:10]
' $SUMMARY
```

### Token Efficiency

**Cost per successful token:**
```bash
jq -s '
  map(select(.lang == "ailang" and .stdout_ok)) |
  group_by(.model) |
  map({
    model: .[0].model,
    cost_per_1k_tokens: (
      (map(.cost_usd) | add) / (map(.total_tokens) | add) * 1000 * 100 | round / 100
    )
  })
' $SUMMARY
```

**Output token efficiency (when successful):**
```bash
jq -s '
  map(select(.lang == "ailang" and .stdout_ok)) |
  group_by(.model) |
  map({
    model: .[0].model,
    avg_output_tokens: (map(.output_tokens) | add / length | round),
    min_output: (map(.output_tokens) | min),
    max_output: (map(.output_tokens) | max)
  })
' $SUMMARY
```

## Exporting Data

### CSV Export for Spreadsheets

**All runs:**
```bash
jq -r '
  [.id, .lang, .model, .stdout_ok, .error_category, .total_tokens, .cost_usd, .duration_ms] |
  @csv
' $SUMMARY > results.csv
```

**Only AILANG failures:**
```bash
jq -r '
  select(.lang == "ailang" and .stdout_ok == false) |
  [.id, .model, .error_category, .err_code, .compile_ok, .runtime_ok, .total_tokens] |
  @csv
' $SUMMARY > failures.csv
```

### JSON for Further Processing

**Failure summary by category:**
```bash
jq -s '
  map(select(.lang == "ailang" and .stdout_ok == false)) |
  group_by(.error_category) |
  map({
    category: .[0].error_category,
    count: length,
    benchmarks: (map(.id) | unique),
    models_affected: (map(.model) | unique),
    total_cost: (map(.cost_usd) | add),
    avg_tokens: (map(.total_tokens) | add / length | round)
  })
' $SUMMARY > failure_summary.json
```

## Tips

1. **Pipe to `head`** for quick checks: `jq ... $SUMMARY | head -20`
2. **Use `less -S`** for wide output: `jq ... $SUMMARY | less -S`
3. **Save complex queries** as shell functions in your `~/.bashrc`
4. **Combine with grep** for filtering: `jq ... $SUMMARY | grep "gpt5"`
5. **Use `-c`** for compact output: `jq -c ...`
6. **Pretty print saved JSON**: `jq . failure_summary.json`

## Tier- and tag-aware queries (v0.14.0+)

These queries operate on the published **dashboard JSON**, not `summary.jsonl`:

```bash
DASH=docs/static/benchmarks/latest.json
```

The `.tiers` block is populated by `internal/eval_analysis/export_json.go`
(M-EVAL-SUITE-PREP M6). Every benchmark under `.benchmarks.<id>` also carries
`tier` and `tags` fields pulled from its YAML.

### Tier pass rates at a glance

```bash
jq -r '.tiers | to_entries | .[] |
  "\(.key)\tailang=\(.value.ailang_success_rate * 100 | round)%\tpython=\(.value.python_success_rate * 100 | round)%\t(\(.value.benchmark_count) benchmarks)"
' $DASH
```

### Benchmarks in a specific tier

```bash
jq -r --arg tier core '
  .benchmarks | to_entries |
  map(select(.value.tier == $tier)) |
  map(.key) | sort | .[]
' $DASH
```

### Benchmarks sharing a tag

```bash
jq -r --arg tag recursion '
  .benchmarks | to_entries |
  map(select(.value.tags // [] | index($tag))) |
  map("\(.key)\t\(.value.tier // "core")\t\(.value.successRate * 100 | round)%") |
  sort | .[]
' $DASH
```

### Promotion candidates (stretch ≥ 80% AILANG)

```bash
jq -r '
  .benchmarks | to_entries |
  map(select(
    .value.tier == "stretch"
    and (.value.languageStats.ailang.successRate // 0) >= 0.8
  )) |
  map(.key) | sort | .[]
' $DASH
```

### Weakest core benchmarks (potential demotion)

```bash
jq -r '
  .benchmarks | to_entries |
  map(select(.value.tier == "core")) |
  map({
    id: .key,
    ailang: (.value.languageStats.ailang.successRate // 0),
    python: (.value.languageStats.python.successRate // 0)
  }) |
  sort_by(.ailang) | .[0:5] |
  map("\(.id)\tailang=\(.ailang * 100 | round)%\tpython=\(.python * 100 | round)%") | .[]
' $DASH
```

### Per-tag pass-rate summary

```bash
jq -r '
  .benchmarks | to_entries |
  map({
    tags: (.value.tags // []),
    rate: (.value.languageStats.ailang.successRate // 0)
  }) |
  map(.tags[] as $t | {tag: $t, rate: .rate}) |
  group_by(.tag) |
  map({
    tag: .[0].tag,
    avg_ailang: ((map(.rate) | add / length) * 100 | round),
    benchmarks: length
  }) | sort_by(.avg_ailang) | reverse |
  .[] | "\(.tag)\t\(.avg_ailang)%\t(\(.benchmarks) benchmarks)"
' $DASH
```

Prefer `ailang eval-matrix --by-tags` when you just want the standard table;
use these queries when you need to cross-filter tiers and tags or emit
machine-readable output.
