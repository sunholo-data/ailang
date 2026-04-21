# M-SONAR-GATE-CLEANUP: Drive SonarCloud Quality Gate to PASS

**Status**: Planned
**Target**: v0.13.1
**Priority**: P2 (Medium — code hygiene, not blocking any feature)
**Estimated**: ~1 week (Phase 1 fixes + coverage; UI a11y dropped — see below)
**Dependencies**: None. Hotspot triage already complete (`sonarcloud-triage` skill, v0.13.0).

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

This is a code-hygiene cleanup — no language or runtime semantics change.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No runtime behavior change. |
| A2: Replayability | 0 | No trace impact. |
| A3: Effect Legibility | 0 | Existing effect declarations untouched. |
| A4: Explicit Authority | 0 | No capability changes. |
| A5: Bounded Verification | +1 | More test coverage = stronger local verification. |
| A6: Safe Concurrency | 0 | No concurrency changes. |
| A7: Machines First | +1 | A log-injection-safe log helper is uniformly machine-parseable; removing duplicate branches makes control flow easier for static analysis. |
| A8: Minimal Syntax | 0 | No new syntax. |
| A9: Cost Visibility | 0 | No cost model changes. |
| A10: Composability | 0 | No API shape changes. |
| A11: Structured Failure | +1 | Fixing `go:S1764` cases may fix real bugs in error paths. |
| A12: System Boundary | +1 | Sanitizing user-controlled log input is exactly an A12 concern. |

**Net Score: +4** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No nondeterminism introduced.
- [x] A3 (Effects): No hidden side effects; logging behavior only narrowed.
- [x] A4 (Authority): No ambient access added.
- [x] A7 (Machines First): Changes improve, not degrade, machine analysis.

## Problem Statement

SonarCloud's quality gate for `sunholo-data_ailang` is currently **FAIL**. Hotspot triage (171 hotspots marked SAFE across 10 rules + 7 issues marked False Positive) has already been completed using the `sonarcloud-triage` skill — `new_security_hotspots_reviewed` is now at 100% PASS.

Three gate conditions still fail, and these require **code changes**, not triage:

**Current State (2026-04-21):**

| Condition | Actual | Threshold | Gap |
|-----------|--------|-----------|-----|
| `new_reliability_rating` | C | ≤ A | 40 new-code bugs (0 BLOCKER/CRITICAL) |
| `new_security_rating` | C | ≤ A | 34 new-code vulnerabilities (0 BLOCKER/CRITICAL, concentrated in one rule) |
| `new_coverage` | 34.2% | ≥ 80% | ~11.7k uncovered new lines to close |

**New-code issues (74 total, gate scope):**

| Rule | Count | Kind | Location pattern |
|------|------:|------|------------------|
| `gosecurity:S5145` | 33 | Log injection — logging user-controlled strings | `internal/coordinator/approval_watcher.go`, `daemon_http.go` |
| `typescript:S1082` | 29 | a11y — clickable element without keyboard listener | `EvolutionTreeSlideovers.tsx` — **bulk WontFix, see below** |
| `typescript:S3923` | 7 | Branches return the same value | `EvolutionTree.tsx` — **bulk WontFix** |
| `go:S1764` | 3 | Identical operands in expression (possible real bug) | TBD — spot-check required |
| `go:S3923` | 1 | Branches return same value | TBD |
| `gosecurity:S6350` | 1 | "User-controlled command argument" | `internal/coordinator/daemon_github.go:100` |

Severity: 12 MAJOR, 62 MINOR. Zero BLOCKER/CRITICAL remain.

**EvolutionTree UI exclusion:** The 36 `typescript:S1082` + `typescript:S3923` findings in `EvolutionTree*.tsx` are low-priority visualization a11y / duplicate-branch concerns in an internal observability component. They will be marked WontFix in SonarCloud (Phase 0 below) rather than fixed — this removes 36 of the 40 "new bugs" in one move and flips `new_reliability_rating` from C to A without touching code.

**Impact:**

- **CI signal**: Quality gate FAIL shows on every PR, which dilutes the signal when a *real* regression appears (the "boy who cried wolf" problem).
- **Coverage metric is noisy**: 271,636 "new lines" reported in the last 30 days is implausibly high for the actual change volume — suggests the leak period configuration is interacting badly with the large v0.11–v0.13 release window. Needs investigation separately from raw coverage improvement.
- **A12 / security**: Log injection (S5145) is a *real* concern for structured trace data. If an attacker can inject `\n[ERROR] fake log line` into a task ID, they can forge log lines that tooling trusts. Minor on each site, but the pattern is worth fixing uniformly.

## Goals

**Primary Goal:** Drive the SonarCloud quality gate from FAIL to PASS by fixing real issues, auditing false positives, and raising new-code coverage — without lowering the gate thresholds or bulk-suppressing rules.

**Success Metrics:**
- `new_reliability_rating` → A (0 new bugs, or all remaining marked FP with spot-check evidence)
- `new_security_rating` → A (0 new vulnerabilities, or all remaining marked FP)
- `new_coverage` ≥ 80% on new code (after correcting the leak-period config)
- Overall quality gate → PASS
- Zero blind suppressions: every FP mark has a one-line justification recorded in the `sonarcloud-triage` skill's `known_fp_rules.md`

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Log-injection fix: central sanitizer vs. per-site refactor to structured logs | Touches all coordinator logging; wrong pick creates a maintenance tax or a leaky abstraction | human | design | med |
| Leak period correction: `previous_version` vs fixed 30-day window | Changes which lines count as "new" → changes what the gate measures | human | design | low |
| Coverage lift strategy: broad test-writing sprint vs narrow targeting of files with highest uncovered-new-lines | Broad sprint is ~2 weeks of work; targeted is ~3 days but leaves long-tail uncovered | human | design | med |

**Decided (2026-04-21):** EvolutionTree UI findings (`S1082`, `S3923`) will be bulk-WontFixed, not fixed in code.

### Design Freeze

- [ ] Log-injection fix shape decided: central `sanitizeForLog(string) string` helper used at call sites, OR switch coordinator logging to `log/slog` with structured fields (kv-only, no user-controlled format strings).
- [x] UI a11y findings dropped — bulk-mark WontFix in Phase 0.
- [ ] Leak period config verified or corrected (currently `sonar.leak.period=30` with type `previous_version`, which is inconsistent).

## Solution Design

### Overview

Three workstreams, landed incrementally:

0. **Bulk WontFix UI findings (Phase 0)** — `typescript:S1082` × 29 and `typescript:S3923` × 7 in `EvolutionTree*.tsx`. Zero code change; triage-only. Moves `new_reliability_rating` C → B.
1. **Security fixes + remaining real bugs (S5145, S6350, S1764 × 3, go:S3923 × 1)** — small, well-bounded Go code. Lands in one PR. Finishes the move to A on both ratings.
2. **Coverage lift** — longest tail. First correct the leak-period metric; then target uncovered new lines in high-churn packages (`internal/coordinator/`, `internal/eval_harness/`).

Order matters: **Phase 0 + Phase 1 flip the reliability/security ratings without touching coverage**, so we can validate gate movement before investing in the long-tail coverage work.

### Expected rating trajectory

| Milestone | `new_reliability_rating` | `new_security_rating` | `new_coverage` | Overall gate |
|-----------|--------------------------|-----------------------|----------------|--------------|
| Today (2026-04-21) | **C** (40 bugs: 36 UI + 4 Go) | **C** (34 vulns: 33 S5145 + 1 S6350) | 34.2% | FAIL |
| After Phase 0 (UI WontFix) | **B** (4 Go bugs remain) | C (unchanged) | 34.2% | FAIL |
| After Phase 1 (Go fixes + sanitizer) | **A** (0 bugs) | **A** (0 vulns) | 34.2% | FAIL (coverage only) |
| After Phase 2 (leak-period fix + coverage) | A | A | **≥ 80%** | **PASS** |

*Ratings assume the Go S1764/S3923 findings are either real bugs (fixable) or defensible FPs (markable) — any rule legitimately marked FP does not count against the rating.*

### Architecture

No architectural change. Specific patterns:

**Log-injection mitigation (preferred):**
```go
// internal/coordinator/sanitize.go (NEW, ~40 LOC)
package coordinator

import "strings"

// SanitizeLog strips CR/LF and control characters that could forge log lines.
// Use at every call site that logs a user-controlled string (task ID, label, repo).
func SanitizeLog(s string) string {
    return strings.Map(func(r rune) rune {
        if r == '\n' || r == '\r' || r < 0x20 { return -1 }
        return r
    }, s)
}
```

Then replace:
```go
log.Printf("[ApprovalWatcher] Processing %s for task %s", event.EventType, event.TaskID)
```
with:
```go
log.Printf("[ApprovalWatcher] Processing %s for task %s",
    SanitizeLog(event.EventType), SanitizeLog(event.TaskID))
```

### Implementation Plan

**Phase 0: Bulk WontFix EvolutionTree UI findings (~15 min, zero code)**
- [ ] Extend `sonarcloud-triage` skill with a `mark_wontfix.sh RULE_KEY "comment"` script (mirrors `mark_safe.sh` but for issues, not hotspots — transitions via `/api/issues/do_transition` with `transition=wontfix`).
  *(Alternative: one-off invocation of the existing `mark_fp.sh` per issue, but a rule-scoped bulk script is reusable.)*
- [ ] `mark_wontfix.sh typescript:S1082 "EvolutionTree visualization component; a11y keyboard-listener parity is not a product requirement for this internal observability view."`
- [ ] `mark_wontfix.sh typescript:S3923 "Duplicate branches are intentional for readability in EvolutionTree render logic; refactor deferred to larger UI rework."`
- [ ] Update `sonarcloud-triage/resources/known_fp_rules.md` with these two entries.
- [ ] Re-run `gate_status.sh` — confirm `new_reliability_rating` moves C → B.

**Phase 1: Security + remaining Go bugs (~1 day)**
- [ ] Add `internal/coordinator/sanitize.go` + table-test for CR/LF/control-char stripping.
- [ ] Apply `SanitizeLog` at all 33 `S5145` sites (script-drive with grep; verify by re-running `fetch_issues.sh` → 0 remaining).
- [ ] Audit `daemon_github.go:100` — `exec.CommandContext(os.Args[0], args...)` with labeled `args` — either mark FP (args go through `exec.Command`, not shell, so injection is infeasible) or sanitize label values.
- [ ] Inspect the 3 `go:S1764` sites; fix genuine duplication bugs; mark FP if intentional (e.g., a `x == x` NaN check).
- [ ] Inspect single `go:S3923` case.
- [ ] Re-run `gate_status.sh` — confirm `new_reliability_rating` = A and `new_security_rating` = A.

**Phase 2: Leak period + coverage (~3–5 days)**
- [ ] Reconcile `sonar.leak.period` config. Either set to a consistent date-based window (`date:2026-03-01`) or to `previous_version` without the `=30` override.
- [ ] Rerun analysis, confirm `new_lines` drops from 271k to something plausible (likely 10–30k).
- [ ] Run `go test -coverprofile=coverage.out ./...`, upload to SonarCloud via CI so it scores coverage against lines rather than reporting 0.
- [ ] If coverage is still below 80% after upload: target the top 3 files by `new_uncovered_lines` (query via `/api/measures/component_tree`).
- [ ] Re-run `gate_status.sh` — confirm overall gate = PASS.

### Files to Modify/Create

**New files:**
- `internal/coordinator/sanitize.go` — `SanitizeLog` helper, ~40 LOC
- `internal/coordinator/sanitize_test.go` — table tests, ~60 LOC

**Modified files:**
- `internal/coordinator/approval_watcher.go` — wrap log args, ~12 touch points
- `internal/coordinator/daemon_http.go` — wrap log args, ~8 touch points
- `internal/coordinator/daemon_github.go` — either wrap args or mark S6350 FP
- `ui/src/features/controlplane/components/ExecHierarchy/EvolutionTreeSlideovers.tsx` — a11y (conditional on Phase 2 decision)
- `ui/src/features/controlplane/components/ExecHierarchy/EvolutionTree.tsx` — duplicate-branch cleanup
- `.github/workflows/ci.yml` — add coverage upload step for SonarCloud
- `sonar-project.properties` (or UI config) — leak period reconciliation

## Examples

### Example 1: Log sanitization

**Before:**
```go
log.Printf("[ApprovalWatcher] Processing custom label %q for task %s (issue #%d)",
    event.Label, event.TaskID, event.IssueNumber)
```
A malicious issue label containing `"evil\n[ERROR] fake"` forges a fake log line.

**After:**
```go
log.Printf("[ApprovalWatcher] Processing custom label %q for task %s (issue #%d)",
    SanitizeLog(event.Label), SanitizeLog(event.TaskID), event.IssueNumber)
```
Control characters stripped; log format stays readable.

### Example 2: a11y fix

**Before:**
```tsx
<div onClick={() => setOpen(true)}>Expand</div>
```

**After:**
```tsx
<div
  role="button"
  tabIndex={0}
  onClick={() => setOpen(true)}
  onKeyDown={(e) => (e.key === "Enter" || e.key === " ") && setOpen(true)}
>
  Expand
</div>
```

## Success Criteria

- [ ] `new_reliability_rating` = A
- [ ] `new_security_rating` = A
- [ ] `new_coverage` ≥ 80%
- [ ] Overall quality gate: PASS
- [ ] `fetch_issues.sh BLOCKER,CRITICAL,MAJOR,MINOR` returns 0 unresolved new-code findings (or all marked FP with recorded rationale)
- [ ] `SanitizeLog` has table-test coverage for CR, LF, tab, control chars, unicode passthrough
- [ ] `sonarcloud-triage/resources/known_fp_rules.md` updated with any new FP decisions made here
- [ ] Post-sprint `gate_status.sh` output archived in `design_docs/implemented/v0_13_1/m-sonar-gate-cleanup.md`

## Testing Strategy

**Unit tests:**
- `sanitize_test.go` — table-driven: normal strings pass through, CR/LF/control chars removed, unicode preserved, empty string handled.

**Integration tests:**
- Run `make test` — coordinator tests must still pass after log-arg wrapping.
- `make verify-examples` — no regression in example runs.

**Manual verification:**
- `.claude/skills/sonarcloud-triage/scripts/gate_status.sh` before and after each phase — archive both outputs.
- Start `make services-start`, hit `/health`, trigger a task with a label containing `"\n"` — confirm sanitizer blocks it.

## Deferred Decisions

- **Which specific `go:S1764` sites are real bugs vs intentional idioms** — agent may decide after inspection; flag any ambiguous case to human.
- **Coverage target packages** — agent may choose the top-uncovered files after the leak-period fix clarifies the true new-code set.
- **S6350 in `daemon_github.go`** — agent may mark FP if analysis confirms no shell invocation; otherwise sanitize `args`.

## Non-Goals

- **Fixing EvolutionTree UI a11y / duplicate-branch findings** — 36 findings bulk-marked WontFix; accessibility parity is not a product requirement for this internal visualization component.
- **Full repo coverage lift (overall `coverage` metric)** — current 39.1% is tracked separately; this sprint only targets *new-code* coverage, which is what the gate measures.
- **Chasing the 2,766 code-smell findings outside new code** — those are tech debt, tracked via `github-issue-triage` skill, not this sprint.
- **Re-architecting coordinator logging into `log/slog`** — considered, but the `SanitizeLog` helper approach is a smaller, safer change; full slog migration can be a separate sprint if desired.
- **Lowering gate thresholds** — if the defaults are too strict, that's a separate config-level conversation with the team, not a workaround for this sprint.

## Timeline

**Day 1** (~2 hours):
- Phase 0 (bulk WontFix UI findings, add `mark_wontfix.sh` to skill) — 30 min
- Phase 1 (security + Go bug fixes) — 1 day

**Day 2–5** (~6 hours):
- Phase 2a (leak-period correction + CI coverage upload) — 1 day
- Phase 2b (targeted test-writing if still < 80%) — 2 days

**Total: ~8 hours across ~1 week** — Phase 0 alone flips reliability to B; Phase 1 finishes the ratings work.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Wrapping log args at every site creates a stylistic tax on future log edits | Low | Simple helper, one-line wrap at call sites; consider a lint rule later |
| `SanitizeLog` strips characters someone legitimately needs in a log line | Low | Control chars + CR/LF only; normal text untouched. Document with a comment |
| Coverage upload breaks CI on flaky tests | Med | Run `go test -race -count=2` locally before enabling the upload step |
| Leak-period change reveals *more* new-code issues than currently visible | Med | Deliberate — better to see real state than hide behind a mis-configured window. Re-scope if count explodes |
| UI a11y fixes collide with parallel UI refactor sprint | Med | Coordinate explicitly; if refactor is <2 weeks away, defer |

## Related Documents

**Implemented (may inform design):**
- [design_docs/implemented/v0_6_2/m-coordinator-feedback-loop-sprint-plan.md](design_docs/implemented/v0_6_2/m-coordinator-feedback-loop-sprint-plan.md) — prior coordinator hygiene sprint
- [design_docs/implemented/v0_7_0/RETROSPECTIVE-observability-dashboard.md](design_docs/implemented/v0_7_0/RETROSPECTIVE-observability-dashboard.md) — logging patterns in observability layer

**Planned (check for overlap):**
- [design_docs/planned/v0_13_0/m-ui-refactor-sprint-plan.md](design_docs/planned/v0_13_0/m-ui-refactor-sprint-plan.md) — Phase 2 UI work must coordinate with this
- [design_docs/planned/v0_13_0/m-dashboard-simplification.md](design_docs/planned/v0_13_0/m-dashboard-simplification.md) — may touch the same UI components

**Tooling:**
- [.claude/skills/sonarcloud-triage/SKILL.md](.claude/skills/sonarcloud-triage/SKILL.md) — scripts for querying + marking findings
- [.claude/skills/sonarcloud-triage/resources/known_fp_rules.md](.claude/skills/sonarcloud-triage/resources/known_fp_rules.md) — standing FP decisions

## References

- [Design Axioms](/docs/references/axioms) — 12 non-negotiable principles
- [SonarCloud rule go:S5145](https://rules.sonarsource.com/go/RSPEC-5145) — log injection
- [SonarCloud rule typescript:S1082](https://rules.sonarsource.com/javascript/RSPEC-1082) — a11y keyboard listeners
- [SonarCloud rule go:S1764](https://rules.sonarsource.com/go/RSPEC-1764) — identical operands

## Future Work

- Add a `ruleguard`/`gocritic` rule in `make lint` that flags `log.Printf` with non-sanitized user-controlled args at `internal/coordinator/*` to prevent regression.
- Lower new-code coverage threshold stepwise (50% → 65% → 80%) if a single-PR lift to 80% is too disruptive.
- Evaluate migrating coordinator logging to `log/slog` for structured fields (would obsolete `SanitizeLog`).

---

**Document created**: 2026-04-21
**Last updated**: 2026-04-21
