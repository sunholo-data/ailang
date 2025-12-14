# Post-Release Version Notes

Historical improvements and lessons learned from past releases.

## Recent Improvements (v0.3.15)

### Version Format Handling
All scripts now accept version with or without 'v' prefix:
- `0.3.15` works
- `v0.3.15` works
- Scripts pass version to underlying tools exactly as given
- No more "directory not found" errors due to version format mismatch

### Auto-Comparison in Metrics
The `extract_changelog_metrics.sh` script now:
- Automatically finds previous version from history
- Calculates AILANG performance difference
- Formats comparison text ("+X% improvement" or "-X% regression")
- Shows before/after percentages: "(48.2% → 33.0%)"
- **Zero manual jq queries needed!**

### Eliminated Manual Work
Before v0.3.15, you needed to:
- Run 15+ manual jq queries to extract metrics
- Manually compare with previous version
- Format comparison text yourself
- Handle version prefix inconsistencies

After v0.3.15:
- **3 scripts do everything automatically**
- No jq queries needed
- No manual calculations
- Version format doesn't matter

## Lessons Learned (v0.4.8)

### Always Validate Before Long-Running Operations

The pre-release checks now include:
1. **Golden file validation**: `make test-import-errors` to catch stale goldens
2. **Agent eval config validation**: `--validate` flag to verify benchmarks list

**Issue discovered**: Agent eval requires explicit `--benchmarks` list (safety feature), but script didn't have it defined. Now fixed with `AGENT_BENCHMARKS` at top of script.

### Agent Eval Requirements

- Agent mode REQUIRES explicit `--benchmarks` list (46 benchmarks as of v0.4.8)
- The list is defined in `AGENT_BENCHMARKS` variable at top of `run_eval_baseline.sh`
- Keep in sync with `benchmarks/` directory

### Golden Files Can Become Stale

When error behavior changes (e.g., module now exists → different error code), golden files need regeneration:
```bash
make regen-import-error-goldens
```

Pre-release checks now validate goldens match current behavior.

### Validate Script Before Running

Use `--validate` flag to check configuration without running full eval:
```bash
.claude/skills/post-release/scripts/run_eval_baseline.sh --validate
```

This checks:
- AGENT_BENCHMARKS is defined
- Benchmark count (46 expected)
- ailang command exists
- Benchmark files exist

## CHANGELOG Template Example

When updating CHANGELOG.md with benchmark results, use this format:

```markdown
### Benchmark Results (M-EVAL)

**Overall Performance**: 68.1% success rate (480 total runs)

**Standard Eval (0-shot + self-repair):**

| Metric | 0.4.4 | 0.4.5 | Change |
|--------|--------|--------|--------|
| **0-shot (first attempt)** | 55.6% | 64.0% (182/284) | **+8.4%** |
| **Final (with repair)** | 60.5% | 68.6% (195/284) | **+8.1%** |
| **Repair effectiveness** | +4.9pp | +4.6pp | -.3pp |
| **Python (final)** | 73.1% | 76.4% (208/272) | +3.3% |

**Agent Eval (multi-turn iterative problem solving):**

| Language | 0.4.4 | 0.4.5 | Change |
|----------|--------|--------|--------|
| **AILANG** | 92.1% | 100.0% (38/38) | **+7.9%** |
| **Python** | 100.0% | 100.0% (38/38) | 0% |

**Key Findings:**
- Major improvement in 0-shot performance
- Perfect agent eval score achieved
```

## Agent Eval Analysis Guide

### Check agent efficiency metrics:
```bash
# Get KPIs (turns, tokens, cost by language)
.claude/skills/eval-analyzer/scripts/agent_kpis.sh eval_results/baselines/X.X.X
```

### Key metrics to track:
- **Avg Turns**: AILANG vs Python (target: ≤1.5x gap)
- **Avg Tokens**: AILANG vs Python (target: ≤2.0x gap)
- **Success Rate**: Both should be 100% (agent mode corrects mistakes)

### Example good result:
```
🐍 Python: 10.6 avg turns, 58k tokens, 100% success
🔷 AILANG: 12.0 avg turns, 72k tokens, 100% success
Gap: 1.13x turns, 1.24x tokens ✅ (within target!)
```

### Example needs work:
```
🐍 Python: 10.6 avg turns, 58k tokens, 100% success
🔷 AILANG: 18.0 avg turns, 178k tokens, 100% success
Gap: 1.7x turns, 3.0x tokens ⚠️ (needs optimization!)
```

### If AILANG significantly worse than Python:
1. Identify expensive benchmarks (from "Most Expensive" section)
2. View transcripts: `.claude/skills/eval-analyzer/scripts/agent_transcripts.sh eval_results/baselines/X.X.X <benchmark>`
3. File optimization tasks for next release
