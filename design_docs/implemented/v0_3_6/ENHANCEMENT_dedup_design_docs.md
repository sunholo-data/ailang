# Enhancement: Design Doc Deduplication & Merging

**Priority**: P2 (Medium)
**Estimated**: ~200 LOC
**Parent Feature**: eval-analyze workflow

## Problem

Currently, `ailang eval-analyze` generates new design docs every time it runs, without checking if similar issues have already been documented. This leads to:

1. **Duplicate design docs** for the same underlying issue
2. **Lost context** from previous analysis runs
3. **Wasted API calls** regenerating similar content
4. **No evolution** of design docs as more evidence accumulates

## Proposed Solution

Before generating a new design doc, search `design_docs/planned/` for similar issues and either:
1. **Merge** new evidence into existing doc (update frequency, add examples)
2. **Skip** generation if issue is already well-documented
3. **Link** related issues instead of duplicating

### Implementation Approach

**Phase 1: Similarity Detection**
```go
// In design_generator.go
func FindSimilarDesignDocs(issue IssueReport, plannedDir string) ([]string, error) {
    // Search for docs with:
    // 1. Same error category
    // 2. Same language
    // 3. Similar error messages (fuzzy match)
    // 4. Overlapping benchmarks
}
```

**Phase 2: Merge Strategy**
```go
type MergeStrategy string
const (
    StrategyCreate  MergeStrategy = "create"   // No similar doc exists
    StrategyMerge   MergeStrategy = "merge"    // Update existing doc
    StrategySkip    MergeStrategy = "skip"     // Already documented
    StrategyLink    MergeStrategy = "link"     // Related but distinct
)

func DetermineMergeStrategy(issue IssueReport, similar []string) MergeStrategy
```

**Phase 3: Doc Merging**
- Increment frequency count
- Add new error examples (limit to 5 total)
- Add new affected benchmarks
- Update timestamp: "Last updated: YYYY-MM-DD"
- Preserve original problem statement
- Append new evidence section

### User Experience

```bash
$ ailang eval-analyze --results eval_results/

→ Analyzing 75 eval results...
Found 2 issues:
  1. AILANG: Runtime Errors (7 failures)
     → Found similar doc: 20251006_runtime_error_ailang.md
     → Strategy: MERGE (adding 3 new examples)
  2. PYTHON: Runtime Errors (9 failures)
     → No similar docs found
     → Strategy: CREATE

✓ Updated: design_docs/planned/20251006_runtime_error_ailang.md
✓ Created: design_docs/planned/20251006_runtime_error_python.md
```

## Implementation Tasks

1. **Similarity search** (~80 LOC)
   - Filename pattern matching
   - Content-based similarity (error messages)
   - Benchmark overlap detection

2. **Merge logic** (~70 LOC)
   - Parse existing design doc
   - Merge new evidence
   - Preserve original analysis
   - Update metadata

3. **CLI flags** (~20 LOC)
   - `--force-new` - Always create new docs (skip dedup)
   - `--merge-threshold` - Similarity % for merging (default: 80%)

4. **Tests** (~30 LOC)
   - Test similarity detection
   - Test merge logic
   - Test skip/link strategies

## Benefits

- **Cost savings**: Avoid regenerating similar docs ($0.10-0.50 per doc)
- **Continuity**: Build on previous analysis as evidence grows
- **Clarity**: Single source of truth per issue
- **History**: Track how issue evolves over time

## Future Enhancements

- Track issue resolution (move to `design_docs/implemented/` when fixed)
- Auto-link to related GitHub issues
- Trend analysis across multiple eval runs
- "Stale doc" detection (issue no longer appears in recent evals)
