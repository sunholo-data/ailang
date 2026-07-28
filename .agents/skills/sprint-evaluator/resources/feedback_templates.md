# Feedback Templates for Sprint Evaluation

Guidelines for writing actionable feedback when evaluation fails. Per the Anthropic blog: specific findings become concrete feedback, not vague criticism.

## Principles

1. **Specific over general** — "Function X in file Y" not "some functions"
2. **Evidence-based** — Cite test output, file paths, line numbers
3. **Actionable** — Include a suggestion for how to fix
4. **Severity-tagged** — high/medium/low helps sprint-executor prioritize
5. **Skeptical but fair** — Assume nothing works until proven otherwise

## Templates by Category

### Tests Broken (HARD FAIL)

```
**HARD FAIL: Tests broken**

`make test` failed with {N} test failures.

Failing tests:
- `TestCachePut` in `internal/cache/store_test.go:42` — expected OK, got nil
- `TestCacheTTL` in `internal/cache/store_test.go:87` — timeout after 5s

**Suggestion:** Run `go test ./internal/cache/ -v -run TestCachePut` to isolate the failure. The nil return suggests the Put method isn't returning the stored value.
```

### Acceptance Criteria Not Met

```
**Criterion not met:** "{criterion text}"
**Feature:** {feature_id}
**File:** `{file_path}`

**Evidence:** The acceptance criterion requires {expected behavior}, but {actual behavior observed}. Specifically:
- Function `{function_name}` at line {N} is a stub/no-op
- No test covers this specific behavior
- The git diff shows this was not implemented

**Suggestion:** {specific implementation guidance}
```

### Lint Failures

```
**Lint errors in `{file_path}`:**
- Line {N}: {lint error message}
- Line {M}: {lint error message}

**Suggestion:** Run `make fmt` first, then address remaining errors. The {specific error type} can be fixed by {guidance}.
```

### File Size Violation

```
**File size violation:** `{file_path}` is {N} lines (limit: 800)

The file grew during this sprint from {before} to {N} lines. The largest functions:
- `{func1}` ({lines} lines)
- `{func2}` ({lines} lines)

**Suggestion:** Split into `{proposed_file1}` ({what it contains}) and `{proposed_file2}` ({what it contains}). Use the `codebase-organizer` skill for guided refactoring.
```

### Documentation Missing

```
**Missing documentation: {type}**

- CHANGELOG: No entry found under active changelog for this sprint's changes
- Examples: No `examples/runnable/{feature}.ail` file found
- Design doc: Status still shows "PLANNED" but implementation is complete

**Suggestion:**
1. Add CHANGELOG entry: `ls changelogs/ | grep current` to find active file
2. Create example: `examples/runnable/{feature}.ail` with comprehensive usage
3. Update design doc status to reflect implementation state
```

### Design Fidelity Issues

```
**Design deviation detected:**

The design doc specifies: "{design doc quote}"
The implementation does: "{what was actually built}"

This is a {minor/significant/major} deviation because {reasoning}.

**Suggestion:** Either:
(a) Update implementation to match design: {how}
(b) Update design doc to reflect the justified deviation with rationale
```

### TODO/HACK in New Code

```
**TODO/HACK markers in new code:**

- `{file}:{line}`: `// TODO: {message}`
- `{file}:{line}`: `// HACK: {message}`

These suggest incomplete implementation. Per AILANG coding standards, TODOs in shipped code indicate unfinished work.

**Suggestion:** Either implement the TODO now (if blocking acceptance criteria) or create a GitHub issue to track it and remove the TODO comment.
```

## Feedback Message Format

When sending feedback to sprint-executor inbox:

```bash
ailang messages send sprint-executor '{
  "type": "evaluation_feedback",
  "correlation_id": "eval_{sprint_id}_round_{n}",
  "sprint_id": "{sprint_id}",
  "evaluation_round": {n},
  "score": {score},
  "result": "fail",
  "issues": [
    {
      "file": "{path}",
      "line": {n},
      "category": "{tests|acceptance|lint|quality|docs|design}",
      "issue": "{description}",
      "suggestion": "{how to fix}",
      "severity": "{high|medium|low}"
    }
  ],
  "max_rounds": 3,
  "report_path": ".ailang/state/evaluations/eval_{sprint_id}_round_{n}.json"
}' --title "Evaluation Failed (Round {n}/3) — Score: {score}/100" --from "sprint-evaluator"
```

## Escalation Message (Round 3 Failure)

```bash
ailang messages send user '{
  "type": "evaluation_escalation",
  "sprint_id": "{sprint_id}",
  "rounds_completed": 3,
  "score_progression": [{round1_score}, {round2_score}, {round3_score}],
  "persistent_issues": [
    "{issue that persisted across all 3 rounds}"
  ],
  "recommendation": "Human review needed — automated feedback loop exhausted"
}' --title "Sprint {sprint_id} needs human review (3 rounds failed)" --from "sprint-evaluator"
```
