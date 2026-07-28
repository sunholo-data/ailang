# Gap Analysis Template

Use this template when documenting gaps found during eval analysis.

## Gap Analysis Report: [Version]

**Date:** YYYY-MM-DD
**Eval Directory:** eval_results/baselines/vX.Y.Z
**Models Tested:** [list models]

### Summary

| Metric | Value |
|--------|-------|
| AILANG Success Rate | X.X% |
| Python Success Rate | X.X% |
| Python-Only Gaps | N |
| New Design Docs | N |

### Python-Only Gaps

| Benchmark | Error Type | Root Cause | Fix |
|-----------|------------|------------|-----|
| benchmark_name | WRONG_LANG | Model wrote Python | Prompt emphasis |
| benchmark_name | type_error | Polymorphic ADT | Design doc |
| benchmark_name | logic_error | Wrong algorithm | Example in prompt |

### Error Categories

| Category | Count | Fix Approach |
|----------|-------|--------------|
| WRONG_LANG | N | Stronger "NOT Python" emphasis |
| PAR_001 | N | Syntax examples |
| type_error | N | Type examples or design doc |
| logic_error | N | Algorithm examples |

### Detailed Gap Analysis

#### Gap 1: [Benchmark Name]

**Error Category:** [WRONG_LANG / type_error / etc.]

**Model Output:**
```ailang
-- Model's broken code here
```

**Expected Output:**
```ailang
-- Working code here
```

**Root Cause:**
- [ ] Prompt doesn't cover this pattern
- [ ] Language limitation (needs design doc)
- [ ] Model hallucination

**Proposed Fix:**
- [ ] Add example to prompt
- [ ] Add entry to "What AILANG Does NOT Have"
- [ ] Create design doc

**Verification:**
```bash
# Test the working example
cat > /tmp/test.ail << 'EOF'
module benchmark/solution
-- Your fix here
EOF
.claude/skills/eval-gap-finder/scripts/test_example.sh /tmp/test.ail
```

### Design Docs Created

| Doc ID | Title | Location |
|--------|-------|----------|
| M-XXX | Feature Name | design_docs/planned/vX_Y_Z/m-xxx.md |

### Prompt Updates Made

| Section | Change |
|---------|--------|
| "NOT Python" warning | Added/Strengthened |
| Quick Reference | Added example for X |
| What AILANG Does NOT Have | Added entry for Y |

### Verification Results

After prompt updates:

| Metric | Before | After | Delta |
|--------|--------|-------|-------|
| AILANG Success | X.X% | Y.Y% | +Z.Z% |
| Python-Only Gaps | N | M | -K |

### Notes

- Any observations or lessons learned
- Patterns to watch for in future
- Related issues or design docs

---

## Checklist

- [ ] All Python-only gaps analyzed
- [ ] Examples tested before adding to prompt
- [ ] Design docs created for language limitations
- [ ] Prompt hash updated in versions.json
- [ ] Follow-up evals run to verify improvement
