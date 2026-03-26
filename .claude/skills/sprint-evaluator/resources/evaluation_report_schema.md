# Evaluation Report JSON Schema

Reports are stored at `.ailang/state/evaluations/eval_<sprint-id>_round_<n>.json`.

## Schema

```json
{
  "sprint_id": "M-CACHE",
  "evaluation_round": 1,
  "timestamp": "2026-03-26T14:30:00Z",
  "design_doc": "design_docs/planned/v0_7/M-CACHE.md",
  "sprint_plan": "design_docs/planned/v0_7/M-CACHE-sprint-plan.md",

  "total_score": 85,
  "pass_threshold": 70,
  "result": "pass",

  "score_breakdown": {
    "tests_pass": 20,
    "lint_clean": 10,
    "acceptance_criteria": 25,
    "code_quality": 12,
    "documentation": 10,
    "design_fidelity": 8
  },

  "hard_fails": [],

  "automated_checks": {
    "tests_pass": true,
    "tests_output": "",
    "lint_clean": true,
    "lint_output": "",
    "file_sizes_ok": true,
    "coverage_pct": "72.3%",
    "todo_hack_count": 2
  },

  "acceptance_criteria": [
    {
      "feature_id": "M1",
      "criterion": "Cache store supports TTL expiration",
      "met": true,
      "evidence": "TestCacheTTLExpiration passes in internal/cache/store_test.go"
    },
    {
      "feature_id": "M2",
      "criterion": "Cache invalidation on write",
      "met": true,
      "evidence": "TestCacheInvalidation verifies write-through behavior"
    }
  ],

  "documentation_check": {
    "changelog_updated": true,
    "examples_created": true,
    "design_doc_status": "matches"
  },

  "design_fidelity": {
    "score": 8,
    "notes": "Implementation matches design goals. Minor deviation: used LRU instead of FIFO as specified, but justified by performance benchmarks."
  },

  "feedback": [],

  "notes": "Clean implementation with good test coverage. Two minor TODO comments for future optimization."
}
```

## Field Descriptions

| Field | Type | Description |
|-------|------|-------------|
| `sprint_id` | string | Sprint identifier (e.g., "M-CACHE") |
| `evaluation_round` | int | Which evaluation round (1, 2, or 3) |
| `timestamp` | ISO 8601 | When evaluation was performed |
| `design_doc` | string | Path to original design document |
| `sprint_plan` | string | Path to sprint plan markdown |
| `total_score` | int | Aggregate score (0-100) |
| `pass_threshold` | int | Always 70 |
| `result` | string | "pass" or "fail" |
| `score_breakdown` | object | Per-category scores |
| `hard_fails` | array | List of hard fail conditions triggered |
| `automated_checks` | object | Raw results from evaluate_sprint.sh |
| `acceptance_criteria` | array | Per-criterion verification |
| `documentation_check` | object | Documentation completeness |
| `design_fidelity` | object | AI judgment on design match |
| `feedback` | array | Actionable feedback items (populated on fail) |
| `notes` | string | Free-form evaluator notes |

## Feedback Item Schema (on fail)

```json
{
  "file": "internal/cache/store.go",
  "line": 142,
  "function": "evictExpired",
  "category": "acceptance_criteria",
  "issue": "TTL expiration not implemented — function is a no-op stub",
  "suggestion": "Implement time-based eviction using a priority queue keyed by expiry timestamp",
  "severity": "high"
}
```

## Hard Fail Item Schema

```json
{
  "condition": "tests_broken",
  "details": "3 tests failing in internal/cache/store_test.go: TestCachePut, TestCacheGet, TestCacheTTL",
  "output_snippet": "--- FAIL: TestCachePut (0.00s)\n    store_test.go:42: expected OK, got nil"
}
```

## Example: Failing Report

```json
{
  "sprint_id": "M-CACHE",
  "evaluation_round": 1,
  "timestamp": "2026-03-26T14:30:00Z",
  "total_score": 52,
  "pass_threshold": 70,
  "result": "fail",

  "score_breakdown": {
    "tests_pass": 20,
    "lint_clean": 5,
    "acceptance_criteria": 15,
    "code_quality": 7,
    "documentation": 0,
    "design_fidelity": 5
  },

  "hard_fails": [],

  "feedback": [
    {
      "file": "internal/cache/store.go",
      "category": "acceptance_criteria",
      "issue": "TTL expiration criterion not met — evictExpired is stub",
      "suggestion": "Implement time.After-based eviction",
      "severity": "high"
    },
    {
      "file": "CHANGELOG.md",
      "category": "documentation",
      "issue": "No changelog entry for cache feature",
      "suggestion": "Add entry under current version section",
      "severity": "medium"
    },
    {
      "file": "internal/cache/store.go",
      "category": "code_quality",
      "issue": "File is 892 lines (exceeds 800 limit)",
      "suggestion": "Split into store.go and eviction.go",
      "severity": "medium"
    }
  ],

  "notes": "Tests pass but 50% of acceptance criteria unmet. No documentation updates. File size violation."
}
```
