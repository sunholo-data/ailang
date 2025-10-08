# Tools Directory Audit

**Date**: 2025-10-08
**Total Scripts**: 21
**Total Lines**: 2,732

## Summary of Findings

### ✅ Core Tools (Keep - Used by Makefile)
| Script | Lines | Purpose | Used By |
|--------|-------|---------|---------|
| `run_benchmark_suite.sh` | 132 | Run all benchmarks across all models | `make eval-suite` |
| `report_eval.sh` | 138 | Generate human-readable eval report | `make eval-report` |
| `eval_baseline.sh` | 146 | Store baseline with git metadata | `make eval-baseline` |
| `eval_diff.sh` | 234 | Compare two eval runs (with colors) | `make eval-diff` |
| `eval_validate_fix.sh` | 180 | Validate specific fix against baseline | `make eval-validate-fix` |
| `eval_prompt_ab.sh` | 172 | A/B test two prompt versions | `make eval-prompt-ab` |
| `eval_auto_improve.sh` | 198 | Automated fix implementation | `make eval-auto-improve` |

**Subtotal**: 7 scripts, 1,200 lines

### 🔧 Utility Scripts (Keep - Called by core tools)
| Script | Lines | Purpose | Used By |
|--------|-------|---------|---------|
| `generate_summary_jsonl.sh` | 115 | Convert results to JSONL | `eval_baseline.sh`, `eval_diff.sh` |
| `generate_matrix_json.sh` | 212 | Performance matrix JSON | `eval_diff.sh`, make targets |

**Subtotal**: 2 scripts, 327 lines

### ⚠️ DUPLICATE/DEPRECATED (Review for Deletion)

#### Comparison Scripts (4 scripts doing similar things)
| Script | Lines | Purpose | Status |
|--------|-------|---------|--------|
| `compare_results.sh` | 283 | Compare eval results with colors | ⚠️ **DUPLICATE** of `eval_diff.sh` |
| `eval_diff.sh` | 234 | Compare two eval runs | ✅ **KEEP** (used by Makefile) |
| `run_comparison.sh` | 38 | Run AILANG vs Python for marketing | ⚠️ **NICHE** - marketing only |
| `run_eval_comparison.sh` | 33 | Run eval across 3 models | ⚠️ **DUPLICATE** of `run_benchmark_suite.sh` |

**Analysis**:
- `compare_results.sh` (283 LOC) is basically same as `eval_diff.sh` (234 LOC)
- `run_eval_comparison.sh` duplicates functionality of `run_benchmark_suite.sh`
- `run_comparison.sh` is only for marketing table generation

**Recommendation**:
- ❌ **DELETE** `compare_results.sh` → use `eval_diff.sh` instead
- ❌ **DELETE** `run_eval_comparison.sh` → use `run_benchmark_suite.sh` instead
- 🤔 **KEEP or MERGE** `run_comparison.sh` into `generate_marketing_table.sh`

#### llms.txt Generation (3 scripts!)
| Script | Lines | Purpose | Status |
|--------|-------|---------|--------|
| `generate-llms-txt.sh` | 182 | Generate llms.txt from docs | ✅ **KEEP** (latest version) |
| `generate-llms-txt-updated.sh` | 45 | Older version? | ⚠️ **DUPLICATE** |
| `generate-llms-txt.sh.bak` | 132 | Backup file | ❌ **DELETE** (backup) |

**Analysis**: All three scripts do the same thing - generate `llms.txt`

**Recommendation**:
- ✅ **KEEP** `generate-llms-txt.sh` (182 LOC, most complete)
- ❌ **DELETE** `generate-llms-txt-updated.sh` (redundant)
- ❌ **DELETE** `generate-llms-txt.sh.bak` (backup file)

### 📊 Special Purpose (Keep)
| Script | Lines | Purpose | Used By |
|--------|-------|---------|---------|
| `generate_marketing_table.sh` | 192 | Generate marketing comparison table | Manual/docs |
| `eval-to-design.sh` | 150 | Full workflow: eval → design docs | `make eval-to-design` |
| `audit-examples.sh` | 95 | Audit example files | Manual |
| `sync-prompts.sh` | 73 | Sync prompts to versions.json | Manual |
| `freeze-stdlib.sh` | 44 | Freeze stdlib for release | Manual |
| `verify-stdlib.sh` | 70 | Verify stdlib integrity | Manual |

**Subtotal**: 6 scripts, 624 lines

## Cleanup Recommendations

### Phase 1: Delete Obvious Duplicates (Safe)
```bash
# Backup files
rm tools/generate-llms-txt.sh.bak

# Duplicates of core functionality
rm tools/compare_results.sh          # Use eval_diff.sh instead
rm tools/run_eval_comparison.sh      # Use run_benchmark_suite.sh instead
rm tools/generate-llms-txt-updated.sh # Use generate-llms-txt.sh instead
```
**Impact**: Remove 491 lines (18% reduction)

### Phase 2: Consolidate Marketing Scripts (Optional)
```bash
# Merge run_comparison.sh into generate_marketing_table.sh
# Or delete if marketing table generation is sufficient
```
**Impact**: Remove 38 lines

### Phase 3: Verify No Orphaned References
```bash
# Check if any deleted scripts are referenced
grep -r "compare_results.sh" .
grep -r "run_eval_comparison.sh" .
grep -r "generate-llms-txt-updated.sh" .
```

## Final State After Cleanup

### Core Eval Tools (7 scripts, 1,200 LOC)
- `run_benchmark_suite.sh` - Run all benchmarks
- `report_eval.sh` - Generate reports
- `eval_baseline.sh` - Store baselines
- `eval_diff.sh` - Compare runs (**consolidated**)
- `eval_validate_fix.sh` - Validate fixes
- `eval_prompt_ab.sh` - A/B testing
- `eval_auto_improve.sh` - Auto-fix

### Utilities (2 scripts, 327 LOC)
- `generate_summary_jsonl.sh` - JSONL export
- `generate_matrix_json.sh` - Matrix JSON

### Special Purpose (6 scripts, 624 LOC)
- `generate_marketing_table.sh` - Marketing
- `generate-llms-txt.sh` - LLM docs (**consolidated**)
- `eval-to-design.sh` - Full workflow
- `audit-examples.sh` - Example auditing
- `sync-prompts.sh` - Prompt versioning
- `freeze-stdlib.sh` + `verify-stdlib.sh` - Release tools

**Total After Cleanup**: 15 scripts, 2,151 LOC (21% reduction)

## Action Plan

1. **Immediate** (Safe deletions):
   - Delete `.bak` file
   - Delete `compare_results.sh` → update any docs to use `eval_diff.sh`
   - Delete `run_eval_comparison.sh` → update any docs to use `run_benchmark_suite.sh`
   - Delete `generate-llms-txt-updated.sh`

2. **Review** (Need user input):
   - Keep or merge `run_comparison.sh` into marketing table script?

3. **Document** (Update references):
   - Update CLAUDE.md to reference consolidated scripts
   - Update eval-orchestrator.md agent to use correct script names
   - Add comment in tools/README.md about script purposes

## Script Purpose Reference

For future reference, here's what each kept script does:

### Evaluation Workflow
- **run_benchmark_suite.sh**: Run all benchmarks × all models → results in `eval_results/`
- **report_eval.sh**: Human-readable summary of results
- **eval_baseline.sh**: Store current results as baseline with git metadata
- **eval_diff.sh**: Compare two result directories (baseline vs new)
- **eval_validate_fix.sh**: Validate specific benchmark fix shows improvement
- **eval_prompt_ab.sh**: A/B test two prompt versions
- **eval_auto_improve.sh**: Automated fix implementation from design docs
- **eval-to-design.sh**: Full pipeline: eval → analyze → design docs

### Data Export
- **generate_summary_jsonl.sh**: Convert results to JSONL for AI analysis
- **generate_matrix_json.sh**: Generate performance matrix with aggregates

### Documentation & Marketing
- **generate-llms-txt.sh**: Aggregate all docs into single LLM-consumable file
- **generate_marketing_table.sh**: Create AILANG vs Python comparison table

### Quality Assurance
- **audit-examples.sh**: Check example files work/fail correctly
- **sync-prompts.sh**: Update prompt versions.json registry
- **freeze-stdlib.sh**: Freeze stdlib for release
- **verify-stdlib.sh**: Verify stdlib integrity

## Lessons Learned

**Why did this happen?**
1. Multiple developers/AI sessions creating similar tools
2. No central registry of existing tools
3. Scripts not well-documented in CLAUDE.md
4. No periodic audits

**How to prevent:**
1. ✅ Add this AUDIT.md to track tool inventory
2. ✅ Update CLAUDE.md with "CHECK TOOLS FIRST" reminder
3. ✅ Create tools/README.md with purpose of each script
4. ✅ Quarterly audits to catch duplication early
5. ✅ Eval-orchestrator agent should reference this audit

---
**Last Updated**: 2025-10-08
**Next Audit**: 2026-01-08 (quarterly)
