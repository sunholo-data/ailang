# M-DX-POST-RELEASE: Post-Release Process Improvements

**Status**: Planned
**Priority**: Medium
**Estimated LOC**: ~150
**Target Version**: v0.4.9

## Problem Statement

During v0.4.8 post-release, several issues were discovered:

1. **Agent eval script missing `--benchmarks` flag** - The agent mode requires an explicit benchmark list but the script didn't have it, causing the eval to fail initially.

2. **Golden file mismatch in CI** - `lnk_unresolved_symbol.json` expected old error code (LDR001) but the module now exists, producing correct error (IMP010). This should have been caught pre-release.

3. **No pre-flight validation** - The post-release scripts ran without validating they would work, leading to mid-run fixes.

## Root Cause Analysis

### Agent Eval Failure
- Agent mode was added later and requires explicit benchmark list (safety feature)
- The `run_eval_baseline.sh` script was not updated when agent mode requirements changed
- No validation step to catch missing configuration before long-running evals

### Golden File Mismatch
- Golden files represent expected error outputs
- When error codes change (e.g., module now exists → different error), goldens become stale
- No automated check in pre-release to verify goldens match current behavior

## Proposed Solution

### 1. Enhanced Pre-Release Checks (Quick Fix)

Update `pre_release_checks.sh` to add:

```bash
# 4/5 Golden file validation
echo "4/5 Validating golden files..."
if make test-import-errors > /tmp/pre_release_goldens.log 2>&1; then
    echo "  ✓ Golden files match current behavior"
else
    echo "  ✗ Golden file mismatch detected"
    echo "  Run: make regen-import-error-goldens"
    FAILURES=$((FAILURES + 1))
fi

# 5/5 Agent eval config validation
echo "5/5 Validating agent eval configuration..."
if .claude/skills/post-release/scripts/run_eval_baseline.sh --validate 2>&1; then
    echo "  ✓ Agent eval configuration valid"
else
    echo "  ✗ Agent eval configuration invalid"
    FAILURES=$((FAILURES + 1))
fi
```

### 2. Add `--validate` Flag to Eval Baseline Script

Add dry-run validation to `run_eval_baseline.sh`:

```bash
if [[ "${1:-}" == "--validate" ]]; then
    echo "Validating agent eval configuration..."
    # Check AGENT_BENCHMARKS is defined
    if [[ -z "${AGENT_BENCHMARKS:-}" ]]; then
        echo "ERROR: AGENT_BENCHMARKS not defined"
        exit 1
    fi
    # Count benchmarks
    BENCHMARK_COUNT=$(echo "$AGENT_BENCHMARKS" | tr ',' '\n' | wc -l | tr -d ' ')
    echo "  Benchmarks defined: $BENCHMARK_COUNT"
    # Dry-run with 1 benchmark
    echo "  Running dry-run with fizzbuzz..."
    if ailang eval-suite --agent --benchmarks fizzbuzz --langs ailang --dry-run 2>/dev/null; then
        echo "  ✓ Agent eval dry-run succeeded"
        exit 0
    else
        echo "  ✗ Agent eval dry-run failed"
        exit 1
    fi
fi
```

### 3. Update Post-Release Skill Documentation

Add lessons learned section to `.claude/skills/post-release/SKILL.md`:

```markdown
## Lessons Learned (v0.4.8)

### Always Validate Before Long-Running Operations
- Run `--validate` flag first to catch configuration issues
- Check golden files match current behavior before release

### Agent Eval Requirements
- Agent mode REQUIRES explicit `--benchmarks` list
- The list is defined in `AGENT_BENCHMARKS` variable in `run_eval_baseline.sh`
- Keep in sync with `benchmarks/` directory (currently 46 benchmarks)
```

## Implementation Plan

| Task | LOC | Priority |
|------|-----|----------|
| Add golden validation to pre-release | 15 | P0 |
| Add `--validate` flag to eval baseline | 25 | P0 |
| Update pre-release script with new checks | 20 | P0 |
| Update post-release SKILL.md | 40 | P1 |
| Add `make post-release-preflight` target | 10 | P2 |

## Acceptance Criteria

- [ ] `pre_release_checks.sh` validates golden files
- [ ] `pre_release_checks.sh` validates agent eval config
- [ ] `run_eval_baseline.sh --validate` works without running full eval
- [ ] Post-release skill documents lessons learned
- [ ] All changes tested manually before merge

## Timeline

- Quick fixes (P0): Same day
- Documentation (P1): Same day
- Make target (P2): Optional, can be deferred

## References

- v0.4.8 post-release issues encountered during session
- `.claude/skills/post-release/scripts/run_eval_baseline.sh`
- `.claude/skills/release-manager/scripts/pre_release_checks.sh`
