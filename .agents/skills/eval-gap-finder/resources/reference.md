# Eval Gap Finder Reference

Quick reference for eval gap analysis commands and queries.

## Command Reference

### Running Evals

```bash
# Run with dev models (fast, cheap)
ailang eval-suite --models gemini-3-flash,claude-haiku-4-5

# Run with output directory
ailang eval-suite --models gemini-3-flash,claude-haiku-4-5 --output eval_results/gap-test

# Generate summary (required before jq queries)
ailang eval-summary eval_results/baselines/v0.6.5
```

### Skill Scripts

```bash
# Full gap analysis
.claude/skills/eval-gap-finder/scripts/run_gap_analysis.sh eval_results/baselines/v0.6.5

# List Python-only gaps
.claude/skills/eval-gap-finder/scripts/identify_python_only.sh eval_results/baselines/v0.6.5

# Categorize errors
.claude/skills/eval-gap-finder/scripts/categorize_errors.sh eval_results/baselines/v0.6.5

# Test an example (CRITICAL: always test before adding to prompt!)
.claude/skills/eval-gap-finder/scripts/test_example.sh /tmp/test.ail
```

## Useful jq Queries

### Overall Statistics

```bash
# AILANG success rate
jq -rs 'map(select(.lang == "ailang")) |
  {total: length, pass: (map(select(.stdout_ok == true)) | length)}' \
  eval_results/baselines/v0.6.5/summary.jsonl

# Python success rate
jq -rs 'map(select(.lang == "python")) |
  {total: length, pass: (map(select(.stdout_ok == true)) | length)}' \
  eval_results/baselines/v0.6.5/summary.jsonl
```

### Finding Gaps

```bash
# Python-only passes (Python succeeds, AILANG fails for ALL models)
jq -rs '
  group_by(.benchmark) |
  map({
    benchmark: .[0].benchmark,
    python_passes: (map(select(.lang == "python" and .stdout_ok == true)) | length > 0),
    ailang_passes: (map(select(.lang == "ailang" and .stdout_ok == true)) | length > 0)
  }) |
  map(select(.python_passes and (.ailang_passes | not))) |
  .[].benchmark' eval_results/baselines/v0.6.5/summary.jsonl

# AILANG failures by error category
jq -rs 'map(select(.lang == "ailang" and .stdout_ok == false)) |
  group_by(.error_category) |
  map({cat: .[0].error_category, count: length}) |
  sort_by(-.count)' eval_results/baselines/v0.6.5/summary.jsonl
```

### Examining Specific Benchmarks

```bash
# Get all results for a specific benchmark
jq -r 'select(.benchmark == "error_handling")' \
  eval_results/baselines/v0.6.5/summary.jsonl

# Get AILANG errors for a benchmark
jq -r 'select(.benchmark == "error_handling" and .lang == "ailang" and .stdout_ok == false) |
  {model, error: .error_category, stderr}' \
  eval_results/baselines/v0.6.5/summary.jsonl
```

### Comparing Models

```bash
# Success rate per model (AILANG only)
jq -rs 'map(select(.lang == "ailang")) |
  group_by(.model) |
  map({
    model: .[0].model,
    total: length,
    pass: (map(select(.stdout_ok == true)) | length)
  })' eval_results/baselines/v0.6.5/summary.jsonl
```

## Prompt Verification

### Checking Current Prompt

```bash
# Display current prompt
ailang prompt

# Display specific version
ailang prompt --version v0.6.5

# List available versions
ailang prompt --list

# Check prompt hash (versions.json)
cat prompts/versions.json | jq '.versions[-1]'
```

### Testing Examples

```bash
# Create test file
cat > /tmp/test.ail << 'EOF'
module benchmark/solution
import std/io (println)

export func main() -> () ! {IO} {
  println("Hello")
}
EOF

# Test it
.claude/skills/eval-gap-finder/scripts/test_example.sh /tmp/test.ail

# Or use ailang directly
ailang check /tmp/test.ail
ailang run --caps IO --entry main /tmp/test.ail
```

### Updating Prompt Hash

After modifying the prompt, update the hash:

```bash
# Calculate new hash
shasum -a 256 prompts/v0.6.5.md | cut -d' ' -f1

# Update versions.json
# (manually edit the hash for the version)
```

## Error Categories

| Category | Meaning | Fix Approach |
|----------|---------|--------------|
| WRONG_LANG | Model wrote Python | Prompt: "NOT Python" |
| PAR_001 | Parse error | Add syntax examples |
| compile_error | General compile failure | Check error details |
| type_error | Type unification failed | Type examples or design doc |
| logic_error | Compiles but wrong output | Algorithm examples |
| runtime_error | Crashes at runtime | Check capabilities |
| EOF | Incomplete code | Model limitation |

## Design Doc Locations

```bash
# Planned features
design_docs/planned/v0_6_6/

# Implemented features
design_docs/implemented/

# Create new design doc
# Use design-doc-creator skill or manually create:
# design_docs/planned/vX_Y_Z/m-feature-name.md
```

## File Locations

| File | Purpose |
|------|---------|
| `prompts/v0.6.5.md` | Current teaching prompt |
| `prompts/versions.json` | Prompt version tracking |
| `eval_results/baselines/` | Eval baseline results |
| `design_docs/planned/` | Planned features |
| `benchmarks/` | Benchmark definitions |
