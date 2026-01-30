# M-ARCH3: Task Classification Consolidation

**Status**: Planned
**Target**: v0.6.5
**Priority**: P1 (Medium-High)
**Estimated**: 12-16 hours
**Dependencies**: None

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Single classification path = deterministic routing |
| A2: Replayability | +1 | Classification decisions logged in one place |
| A3: Effect Legibility | 0 | No change to effect handling |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | Single source of truth enables verification |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Removes redundant classification logic |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | 0 | No cost changes |
| A10: Composability | +1 | Unified classifier composes with all providers |
| A11: Structured Failure | +1 | Classification errors in one place |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +6** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): Single classification path is deterministic
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Reduces code duplication

## Problem Statement

Task type classification logic is duplicated in 3 locations with different implementations, leading to inconsistent routing and maintenance burden.

**Current State:**

1. **TaskAnalyzer** (`analyzer.go:65-111`)
   - Uses keyword matching with priority: bug > test > docs > research > refactor > feature
   - 27 keyword lists
   - Location: `internal/coordinator/analyzer.go:239-249`

2. **CapabilityDetector** (`capability_detector.go:40-80`)
   - ALSO does keyword matching for capability detection
   - Overlapping logic with TaskAnalyzer
   - Location: `internal/coordinator/capability_detector.go:84-200`

3. **Provider CanHandle()** methods
   - Each provider has its own switch statement for task types
   - Duplicates classification concepts
   - Locations:
     - `internal/coordinator/provider_claude.go:39-45`
     - `internal/coordinator/provider_gemini.go:42-48`
     - `internal/coordinator/task_executor.go:145-151`

**Identical switch pattern appears 3 times:**
```go
case TaskTypeBugFix, TaskTypeFeature, TaskTypeRefactor, TaskTypeTest:
    // coding tasks
case TaskTypeDocs, TaskTypeResearch:
    // non-coding tasks
```

**Impact:**
- Inconsistent task routing when implementations drift
- Bug fixes must be applied to 3+ locations
- Hard to understand complete classification logic
- New task types require changes in multiple files

## Goals

**Primary Goal:** Create single `TaskClassifier` service that owns all task type classification, reducing 3 implementations to 1.

**Success Metrics:**
- Single source of truth for task classification
- Provider CanHandle() delegates to classifier
- CapabilityDetector uses classifier
- All routing tests pass
- Classification logic in one file (<300 lines)

## Solution Design

### Overview

Create `TaskClassifier` service that encapsulates all task classification logic. Providers and analyzers delegate to this service instead of implementing their own logic.

### Architecture

```
internal/coordinator/
├── classifier/
│   ├── classifier.go     # TaskClassifier service (~200 LOC)
│   ├── keywords.go       # Keyword definitions (~100 LOC)
│   └── classifier_test.go # Comprehensive tests (~300 LOC)
├── analyzer.go           # Delegates to classifier
├── capability_detector.go # Delegates to classifier
├── provider_claude.go    # CanHandle uses classifier
└── provider_gemini.go    # CanHandle uses classifier
```

**Components:**

1. **TaskClassifier**: Central service with `Classify(directive string) TaskType`
2. **Keywords**: Centralized keyword lists for each task type
3. **TaskTypeGroups**: `IsCodingTask(TaskType)`, `IsResearchTask(TaskType)` helpers

### Implementation Plan

**Phase 1: Create Classifier** (~4 hours)
- [ ] Create `internal/coordinator/classifier/classifier.go`
- [ ] Move keyword lists from analyzer.go to `keywords.go`
- [ ] Implement `Classify(directive string) TaskType`
- [ ] Implement helper functions: `IsCodingTask()`, `IsResearchTask()`
- [ ] Add comprehensive unit tests

**Phase 2: Migrate Analyzer** (~3 hours)
- [ ] Refactor `analyzer.go` to use TaskClassifier
- [ ] Remove duplicate keyword definitions
- [ ] Verify analyzer tests pass

**Phase 3: Migrate CapabilityDetector** (~3 hours)
- [ ] Refactor `capability_detector.go` to use TaskClassifier
- [ ] Remove overlapping keyword matching
- [ ] Verify capability tests pass

**Phase 4: Migrate Providers** (~4 hours)
- [ ] Refactor `provider_claude.go` CanHandle to use classifier
- [ ] Refactor `provider_gemini.go` CanHandle to use classifier
- [ ] Refactor `task_executor.go` switch to use classifier helpers
- [ ] Remove duplicate switch statements
- [ ] Verify provider tests pass

### Files to Modify/Create

**New files:**
- `internal/coordinator/classifier/classifier.go` (~200 LOC)
- `internal/coordinator/classifier/keywords.go` (~100 LOC)
- `internal/coordinator/classifier/classifier_test.go` (~300 LOC)

**Modified files:**
- `internal/coordinator/analyzer.go` - Remove classification, use classifier (~-60 LOC)
- `internal/coordinator/capability_detector.go` - Remove overlap, use classifier (~-80 LOC)
- `internal/coordinator/provider_claude.go` - Simplify CanHandle (~-20 LOC)
- `internal/coordinator/provider_gemini.go` - Simplify CanHandle (~-25 LOC)
- `internal/coordinator/task_executor.go` - Use classifier helpers (~-15 LOC)

## Examples

### Example 1: TaskClassifier Interface

**New classifier service:**
```go
package classifier

type TaskClassifier struct {
    keywords map[TaskType][]string
}

func NewTaskClassifier() *TaskClassifier {
    return &TaskClassifier{
        keywords: defaultKeywords,
    }
}

// Classify returns the TaskType based on directive content
func (c *TaskClassifier) Classify(directive string) TaskType {
    directive = strings.ToLower(directive)

    // Priority order: bug > test > docs > research > refactor > feature
    priorities := []TaskType{
        TaskTypeBugFix,
        TaskTypeTest,
        TaskTypeDocs,
        TaskTypeResearch,
        TaskTypeRefactor,
        TaskTypeFeature,
    }

    for _, taskType := range priorities {
        for _, keyword := range c.keywords[taskType] {
            if strings.Contains(directive, keyword) {
                return taskType
            }
        }
    }
    return TaskTypeFeature // default
}

// Helper functions
func (c *TaskClassifier) IsCodingTask(t TaskType) bool {
    return t == TaskTypeBugFix || t == TaskTypeFeature ||
           t == TaskTypeRefactor || t == TaskTypeTest
}

func (c *TaskClassifier) IsResearchTask(t TaskType) bool {
    return t == TaskTypeDocs || t == TaskTypeResearch
}
```

### Example 2: Provider Using Classifier

**Before (provider_claude.go):**
```go
func (p *ClaudeCodeProvider) CanHandle(task *AnalyzedTask) bool {
    switch task.Type {
    case TaskTypeBugFix, TaskTypeFeature, TaskTypeRefactor, TaskTypeTest:
        return true
    case TaskTypeDocs, TaskTypeResearch:
        return false
    default:
        return true
    }
}
```

**After:**
```go
func (p *ClaudeCodeProvider) CanHandle(task *AnalyzedTask) bool {
    // Claude handles coding tasks, not research
    return p.classifier.IsCodingTask(task.Type)
}
```

### Example 3: Analyzer Using Classifier

**Before (analyzer.go):**
```go
func (a *TaskAnalyzer) analyzeType(directive string) TaskType {
    lower := strings.ToLower(directive)

    // Bug keywords
    bugKeywords := []string{"bug", "fix", "error", "crash", "broken"...}
    for _, kw := range bugKeywords {
        if strings.Contains(lower, kw) {
            return TaskTypeBugFix
        }
    }

    // Test keywords
    testKeywords := []string{"test", "spec", "coverage"...}
    // ... 50+ more lines of keyword matching
}
```

**After:**
```go
func (a *TaskAnalyzer) analyzeType(directive string) TaskType {
    return a.classifier.Classify(directive)
}
```

## Success Criteria

- [ ] Single `TaskClassifier` service owns all classification
- [ ] All keyword lists in one file (`keywords.go`)
- [ ] Provider CanHandle() uses IsCodingTask()/IsResearchTask() helpers
- [ ] No duplicate switch statements for task type grouping
- [ ] All existing routing tests pass
- [ ] New test cases cover edge cases
- [ ] Documentation explains classification priority

## Testing Strategy

**Unit tests:**
- Test each keyword triggers correct TaskType
- Test priority order (bug > test > docs > etc.)
- Test IsCodingTask/IsResearchTask helpers
- Test edge cases (empty directive, mixed keywords)

**Integration tests:**
- Task routing produces same results as before
- Provider selection unchanged

**Manual testing:**
- Submit tasks via CLI, verify correct routing

## Non-Goals

**Not in this feature:**
- Adding new task types - Focus on consolidation
- ML-based classification - Keep keyword-based for now
- User-configurable keywords - Future enhancement

## Timeline

**Day 1** (4 hours):
- Create classifier package with service and keywords

**Day 2** (4 hours):
- Migrate analyzer and capability detector

**Day 3** (4 hours):
- Migrate providers and task executor
- Final testing

**Total: ~12 hours across 3 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Breaking task routing | High | Add golden tests for classification before refactor |
| Missing edge cases | Medium | Review all 3 implementations for unique cases |
| Import cycles | Low | classifier/ has no coordinator dependencies |

## Related Documents

**Implemented (may inform design):**
- [design_docs/implemented/v0_6_2/m-coord-github-auto-routing.md](design_docs/implemented/v0_6_2/m-coord-github-auto-routing.md) (0.40)

## References

- [Design Axioms](/docs/references/axioms)
- `internal/coordinator/analyzer.go:239-249` - Current classification
- `internal/coordinator/capability_detector.go:84-200` - Duplicate classification

## Future Work

- ML-based classification using task descriptions
- User-configurable keyword lists in config.yaml
- Classification confidence scores

---

**Document created**: 2026-01-05
**Last updated**: 2026-01-05
