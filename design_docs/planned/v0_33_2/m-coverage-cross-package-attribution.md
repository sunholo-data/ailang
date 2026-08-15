# M-COVERAGE-CROSS-PACKAGE-ATTRIBUTION: Should `make test-coverage` adopt `-coverpkg`?

**Status**: Planned
**Target**: v0.33.2
**Priority**: P2 (Low)
**Estimated**: ≤ 1 day for the recommended option (Option C); 3–4 days decomposed if D-COV-1 is answered "EXECUTION" (see Decomposition)
**Dependencies**: None
**Planner-Lane**: codex-ok
**Doc type**: DECISION document. The mission charter classifies this item as "a decision with
tradeoffs, NOT a mechanical edit" and forbids patching `make/coverage.mk` in passing. This
iteration's deliverable is this document only; any edit happens in a ratified follow-up sprint.

---

## Problem Statement

`make test-coverage` runs `go test -coverprofile=... -covermode=atomic <105 test-bearing
packages>` with **no `-coverpkg`** (M1, V9). Under Go's default attribution, each package's
coverage counts only hits from that package's **own** tests. Meanwhile `cmd/ailang/**` sits in
`sonar.coverage.exclusions` (M2, V10).

**Consequence — the gap class:** a helper added to a non-excluded package (e.g. `internal/pkg`)
*in order to serve* `cmd/ailang` reads **0% by construction**, no matter how thoroughly
`cmd/ailang`'s tests exercise it. The exercising side is coverage-excluded; the exercised side
gets no attribution. Iteration 204 hit one live instance: `pkg.ParseManifestFile` read 0.0% on
SonarCloud's new-code gate despite being on `cmd/ailang`'s hot path (commits `3ec1dcb02`,
`54fdcb32c`, issues #720/#722/#723). Iteration 204 fixed the **instance** by writing own-package
tests. This doc decides what, if anything, to do about the **class**.

**Two charter premises are REFUTED by measurement and must not be repeated:**

1. *"Adding `-coverpkg` would slow a 120k-line suite materially"* — **false**. The full-suite
   A/B (3 replicates per arm, cold test cache) measured +1s/+1s/+3s, i.e. ~+1–4% (M4).
2. *"It moves every number in the repo — the badge, the 29% gate, Sonar's baseline"* — the
   numbers move, but **upward and harmlessly for the gate**: 45.5% → 48.1% total, deterministic
   across all three replicates, both arms clearing `COVERAGE_THRESHOLD := 29` with wide headroom
   (M3, M4). The gate does not break; if anything it silently *loosens* (see Recalibration).

**The real measured cost is the artifact, not the CPU** (M5): the merged profile grows from
5.7 MB / 74,336 blocks to **599 MB / 7,808,904 blocks (~105×)**, because with
`-coverpkg=./...` each of the 105 test binaries emits a profile spanning ALL packages and
`go test` concatenates them. The local gate tooling stays cheap (`go tool cover -func`:
539ms → 1463ms), so the exposure is `sonar.go.coverage.reportPaths` ingesting a 599 MB file,
CI disk, and any artifact movement. **Nobody has measured SonarCloud ingesting a 599 MB
profile; this doc does not claim it works or fails.** Note the failure mode would be *silent*:
the SonarCloud scan step in CI is `continue-on-error: true` ("Sonar is a reporting layer, not a
gate", V12) — a rejected ingest loses coverage data on Sonar without turning CI red.

**Impact:** affects agents and humans triaging coverage numbers, the CI critical path (the
coverage suite is 43.7% of the `test` job, M8), and the meaning of the repo's headline
coverage number. Not a release blocker; it is a metric-semantics decision.

---

## Goals

**Primary Goal:** Decide — with measurements, not assertions — whether cross-package coverage
attribution (`-coverpkg`) becomes the repo's coverage semantics, and close the iteration-204
gap **class** in whichever way preserves the signal that caught it.

**Success Metrics:**
- The decision (D-COV-1) is made explicit and recorded, instead of living implicitly in a
  Makefile flag nobody chose deliberately.
- The gated/badged/Sonar-reported metric keeps a documented meaning that a reader can state in
  one sentence.
- The iteration-204 gap class has a named detection path (not "invisible from both sides").
- No acceptance criterion in the follow-up sprint is vacuous ("make X passes" where X is
  configured not to see the effect — the exact vacuity that produced this item).

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D-COV-1: What the repo coverage number MEANS — **LOCALITY** (this package has its own tests) vs **EXECUTION** (any test anywhere exercises this package) | Determines whether `-coverpkg` is an improvement or a regression; flips the interpretation of every historical number | **human (Mark)** — one word | design | high |
| D-COV-2: Whether the profile shipped to Sonar ever changes | 599 MB artifact (M5) with unmeasured ingest behavior; failure mode is silent (V12) | agent, bounded by D-COV-1 | compile | med |
| D-COV-3: Threshold recalibration rule if the metric ever flips | A metric change that raises `total:` without new tests silently weakens the 29% gate | human ratifies the rule below | design | low |
| D-COV-4: CI coverage double-run elimination (M8) | 492s of a 1127s critical-path job; but **independent** of the `-coverpkg` decision | agent | compile | low |

### Design Freeze

Before any implementation sprint begins:

- [ ] D-COV-1 answered by Mark (LOCALITY or EXECUTION — see DECISIONS FOR MARK)
- [x] The `-coverpkg` A/B is measured, replicated, and recorded (M4–M7)
- [x] The Sonar-ingest-of-599MB question is explicitly marked UNMEASURED (no one may assert it)

---

## DECISIONS FOR MARK

**D-COV-1 (one word): does the repo's headline coverage number mean `LOCALITY` or `EXECUTION`?**

- **LOCALITY** — "N% of statements are covered by their own package's tests." The stronger,
  narrower claim. Keeps the current semantics; ratifies Option C below.
- **EXECUTION** — "N% of statements are executed by some test somewhere." The broader, weaker
  claim. Triggers the decomposed Option A sprint below.

This is a product-level call about what the number is *for*. The recommendation below is
**LOCALITY**, with the reasoning; but the doc does not pretend the other answer is wrong — it
is a different, defensible goal. Everything else in this doc is decidable by agents once this
word is chosen.

---

## The Crux (open question 1): is whole-repo attribution the right metric?

This is the question the item exists to resolve. Both directions, argued honestly:

**For `-coverpkg` (EXECUTION):**
- It is more honest about what actually runs. 2.6pp of real, deterministic exercise (M4) is
  currently invisible. A human triaging "untested" code can waste time on a helper that is in
  fact executed on every `cmd/ailang` invocation's test path.
- It removes a perverse incentive: extracting a helper out of `cmd/ailang` into a testable
  internal package *lowers* the visible coverage number today (the code leaves an excluded
  package and arrives at 0%), which punishes exactly the refactoring direction the repo wants.
- The 0%-by-construction reading is a measurement artifact, not a fact about test quality.

**Against `-coverpkg` (LOCALITY) — and why this side wins on the evidence:**
- **The change would have hidden the very defect that motivated it.** Iteration 204's actual
  gap was `ParseManifestFile` having **no own-package test pinning the #720 mechanism**.
  SonarCloud's new-code 0% flag was a **true positive**: it fired, and the fix it forced —
  own-package tests (`54fdcb32c`) — is the fix the repo's testing philosophy wants. Under
  `-coverpkg`, `cmd/ailang`'s incidental exercise would have painted those lines green, the
  flag never fires, and the mechanism ships with zero pinned tests. A metric that reads higher
  while the defect class becomes undetectable is strictly worse at the one job this item asked
  it to do.
- **The "invisible from both sides" premise is half-false.** The helper's 0% *is* visible —
  `internal/pkg` is not in `sonar.coverage.exclusions` (M2), which is precisely how iteration
  204 got caught. What is invisible is the *mitigating context* (cmd/ailang exercises it), and
  losing that context errs in the safe direction: it generates test-writing work, not silence.
- **+2.6pp with zero new tests is dilution, not improvement.** The denominator widens by nine
  packages that have no tests at all (M6), and lines with no local test turn green via distant
  integration paths. Every historical number and trend breaks meaning at the flip point.
- **The live exposure is small post-204.** On the motivating pair, `-coverpkg` changes exactly
  2 of 85 `internal/pkg` functions, partially (M7). The class is real but currently thin; the
  599 MB artifact and unmeasured Sonar ingest (M5) are concrete costs against a thin benefit.

**Resolution offered:** LOCALITY for the gated metric; EXECUTION available on demand as a
*diagnostic*, never as the gate (Option C). This keeps the true-positive generator that caught
#720's coverage gap AND gives triagers the "is this actually exercised?" answer without moving
any gated number.

---

## Options Considered (open questions 2–3: scope and M5 mitigation)

| Option | What changes | M5 exposure | Verdict |
|--------|-------------|-------------|---------|
| **A. `-coverpkg=./...` on the main target** | Gate, badge, and Sonar all flip to EXECUTION semantics | Full: 599 MB profile into Sonar (unmeasured), CI disk | Only if D-COV-1 = EXECUTION; requires the decomposed sprint below, incl. measuring Sonar ingest before switching |
| **B. Narrower `-coverpkg` scope** (e.g. `./internal/...`) | Same flip, smaller span | Reduced but same order of magnitude — the 105× blowup comes from *every test binary emitting a whole-scope profile*, and 96+ of the 105 test-bearing packages are under `internal/` (the profile stays O(binaries × scope)) | Dominated: nearly all of A's cost, a mutilated version of A's semantics |
| **C. RECOMMENDED — keep the main target as-is; add a separate, non-gating diagnostic target** (`test-coverage-xpkg`) writing `coverage/coverage-xpkg.out`, never shipped to Sonar, never gated | Nothing gated moves; EXECUTION view available locally/on-demand for triage | None: the 599 MB file is local and opt-in, and `coverage/coverage-xpkg.out` is git-ignored (measured against `.gitignore`, **V16** — not inferred from Sonar config) | **Default.** Preserves the true-positive signal; answers the triage question (*whether*, not *by whom* — V17); zero unmeasured dependencies |
| **D. Do nothing** | — | — | Rejected: leaves the "is it exercised?" question unanswerable without ad-hoc commands, and leaves this decision undocumented, so the item recurs |
| **E. Post-process/merge the `-coverpkg` profile before Sonar ingest** | A, plus a dedup/merge step to shrink the file | Unknown — no merge tool is selected or measured; `go tool cover` has no merge mode for text profiles | Not evaluated further here; only relevant inside an Option-A sprint, where it becomes a measured sub-task |

**Recalibration (open question 4):** under Option C, `COVERAGE_THRESHOLD := 29` and its meaning
are untouched — no recalibration. If D-COV-1 = EXECUTION (Option A), the rule is: **at the flip
commit, raise the threshold by the measured delta, rounded UP** — 29 + (48.1 − 45.5) → **32** —
so the gate never becomes easier to pass as a side effect of a metric redefinition. The delta
must be re-measured at the flip commit, not copied from M4.

---

## Solution Design (Option C, the recommendation)

### Overview

Keep `make test-coverage` byte-for-byte in its current semantics: it is the LOCALITY metric,
feeding the 29% gate, the badge, and Sonar. Add one sibling target, `test-coverage-xpkg`, that
runs the identical package list with `-coverpkg=./...` into `coverage/coverage-xpkg.out`. It is
not in CI, not a gate input, and not in `sonar.go.coverage.reportPaths`. Its sole purpose is
answering **"is this 0%-locality reading executed by any test elsewhere?"** during triage — the
question iteration 204 had to answer by hand.

**The contract stops there, deliberately.** The diagnostic answers *whether*, never *by whom*.
A merged `-coverpkg` profile is a flat list of `file:range numStatements count` lines under a
single `mode:` header — it carries **no originating-test column**, so per-test provenance is not
recoverable from it (V17). Any wording promising attribution to a specific test or test package
would be an unsupported capability claim, and every assertion in this doc — M15, XC1, the
Testing Strategy — establishes only zero-versus-nonzero aggregate execution. Recovering
provenance would need a different design (bounded per-test-package probes into separate
profiles, plus an index from source blocks to originating test packages); that is recorded as
Future Work, NOT as something this target does.

Separately (D-COV-4), eliminate the CI double-run: `Check coverage gate (M-P2)` (line 139)
already executes the full coverage suite via `test-coverage-gate`'s dependency, and `Generate
test coverage` (lines 217–219) runs `make test-coverage` again (V11) — 492s of a 1127s
critical-path job doing the work twice (M8). Reuse the first run's `coverage/coverage.out` for
the badge extraction. This is a pre-existing inefficiency this doc NAMES; it is independent of
the `-coverpkg` decision and must be its own commit so a revert of either leaves the other
intact.

### Files to Modify/Create

- `make/coverage.mk` (+~12/−1 LOC) — extract the line-18 package-discovery pipeline into ONE
  shared Make variable (e.g. `COVERAGE_PKG_LIST`; see Conflict Surface finding 3 — copy-paste
  is rejected), consumed by `test-coverage` (behavior-identical refactor) and by the new
  `test-coverage-xpkg` target (same list, plus `-coverpkg=./...`, output
  `coverage/coverage-xpkg.out`); add the new output to `coverage-clean`. NOT edited this
  iteration — follow-up sprint only.
- `.github/workflows/ci.yml` (−2/+2 LOC) — D-COV-4: drop the second `make test-coverage` at
  lines 217–219, keep the badge extraction reading the gate run's `coverage/coverage.out`.
- `docs/docs/guides/development-workflow.md` (+~8 LOC) — one paragraph: what LOCALITY means,
  when to reach for `test-coverage-xpkg`.
- `CHANGELOG.md` — entry under Unreleased.

No production code, no parser/types/codegen, no `sonar-project.properties` change.

### Examples

**Today (the iteration-204 triage, done by hand):**
```bash
make test-coverage
go tool cover -func=coverage/coverage.out | grep ParseManifestFile
# internal/pkg/manifest.go: ParseManifestFile  0.0%   ← is this dead, or exercised via cmd/ailang? Unknowable from here.
```

**With Option C:**
```bash
make test-coverage-xpkg
go tool cover -func=coverage/coverage-xpkg.out | grep ParseManifestFile
# non-zero  → exercised cross-package: not dead; still needs OWN tests to satisfy the gate/Sonar
# zero      → genuinely unexercised anywhere: higher-priority gap
```
The gated number never moves in either case; the diagnostic answers the triage question the
current tooling cannot.

**Reading the xpkg output — counts, not presence.** The example above is sound because
`go tool cover -func` percentages are computed FROM execution counts. But never draw the same
inference from a raw-profile grep: with `-coverpkg=./...`, every in-scope package appears in
`coverage-xpkg.out` merely by being instrumented — M15 measured `internal/mcp_client` present
with 1024 profile lines and **every** trailing count zero. A triager who greps the raw profile
and finds lines has learned nothing about execution; the trailing count field of the profile
line (`file.go:start,end numStatements count`) being > 0 is the only execution evidence.

---

## Success Criteria (open question 5: criteria that can FAIL)

Each names the mutation that kills it, and each carries a **vacuity check** answering: *what
would this criterion still pass under, if the thing it claims were false?* None is of the form
"make X passes" for an effect X cannot see.

- [ ] **XC1 — the diagnostic attributes cross-package EXECUTION, not mere instrumentation.**
  Profile line format is `file.go:startLine.col,endLine.col numStatements count` — "appears in
  the profile" and "was executed" are DIFFERENT columns. With `-coverpkg=./...` a testless
  package is instrumented into every binary's emitted profile with all counters at zero (M15:
  `internal/mcp_client` — 1024 profile lines, 0 nonzero; `internal/auth/gcp` — 288 lines,
  0 nonzero), so line presence proves nothing. The assertion is two-sided:
  (a) `coverage/coverage-xpkg.out` contains ≥1 `internal/version` block whose trailing count
  field is **> 0** (M15 baseline: 56 of 168 lines nonzero — genuinely exercised
  cross-package); AND (b) the negative control `internal/mcp_client` is present in the profile
  with **every** trailing count zero — i.e. a present-but-unexecuted package must NOT satisfy
  predicate (a). If `internal/mcp_client` ever gains cross-package execution, re-point the
  control at another all-zero package (M15 lists candidates) — do NOT delete it; the control
  is what keeps this criterion non-vacuous. *Killing mutations:* remove `-coverpkg=./...` —
  `internal/version` vanishes from the profile and (a) fails; OR weaken (a) back to a presence
  grep — (b) fails it, because `mcp_client` is "present" too. *Vacuity check:* the original
  presence-based draft of this criterion passed whether or not cross-package execution
  happened (it passed on `internal/version` for the right reason by luck; it would have passed
  falsely on `internal/mcp_client`) — the count predicate plus the negative control is exactly
  what removes that degree of freedom.
- [ ] **XC2 — the gated metric is untouched.** After the sprint, `coverage/coverage.out`
  contains NO line for `internal/version` (nor any other M6 package), and the profile stays
  single-digit MB. *Killing mutation:* someone adds `-coverpkg` to the main `test-coverage`
  target "while they're in there" — `internal/version` appears, criterion fails. *Vacuity
  check:* this is a presence test used where presence is the effect under test — in a
  no-`-coverpkg` profile, an `internal/version` line can arise ONLY from instrumentation being
  added, so absence is meaningful (unlike XC1, no execution inference is drawn). It would
  still pass if the gate were weakened some OTHER way (threshold edit, package-list shrink) —
  those holes are covered by XC4 and XC5 respectively, not left silent.
- [ ] **XC3 — D-COV-4 actually halves the work.** One CI run post-change shows exactly one
  execution of "Running tests with coverage..." in the `test` job log, and the badge step still
  exports a non-empty `COVERAGE` value. *Killing mutation:* re-adding the second
  `make test-coverage`, or deleting the gate step so the badge reads a stale/absent profile.
  *Vacuity check:* the log-grep alone would still pass if a second full run were re-introduced
  under a DIFFERENT log string — which is why Testing Strategy pairs it with the wall-time
  comparison against the M8 baseline (1127s / 492s of coverage steps); renaming a step cannot
  fool the clock.
- [ ] **XC4 — threshold untouched.** `git diff` of the sprint shows no change to
  `COVERAGE_THRESHOLD` (Option C moves no metric, so no recalibration is licensed). *Vacuity
  check:* it would still pass if the threshold were edited in some LATER commit outside the
  sprint's diff — accepted residual: XC4 pins this sprint only; the standing guard against
  post-sprint drift is the Future Work package-set assertion, and any threshold change remains
  visible in `git log -p make/coverage.mk`.
- [ ] **XC5 — single package-discovery pipeline (see Conflict Surface).** The
  `go list`/`TestGoFiles` discovery pipeline exists exactly ONCE in `make/coverage.mk`, as a
  shared variable consumed by both `test-coverage` and `test-coverage-xpkg`:
  `grep -c 'TestGoFiles' make/coverage.mk` prints `1`. *Killing mutation:* copy-paste the
  pipeline into `test-coverage-xpkg` and edit one of its `grep -v` filters — the count becomes
  2 and the criterion fails; without this criterion that drift is silent, because both targets
  still exit 0 while measuring different package universes. *Vacuity check:* it would still
  pass if the ONE shared pipeline were itself edited to a wrong list — but then both targets
  move together, which XC2 (gated-profile package set) and the M-gate observe; XC5's job is
  divergence-between-targets only, and it can only fail via the duplication it polices.
- [ ] All tests passing; CHANGELOG.md + development-workflow guide updated.

---

## Testing Strategy

- **XC1/XC2/XC5 as a script check, not eyeballs:** a small assertion in the sprint (CI step or
  one-shot verification recorded in the PR). XC1 reads COUNTS, not presence (command shape in
  M15): awk over `coverage-xpkg.out` asserting ≥1 `internal/version` line with trailing count
  > 0 AND zero nonzero-count lines for the `internal/mcp_client` control; XC2 asserts
  `internal/version` absent from the gated `coverage.out`; XC5 asserts
  `grep -c 'TestGoFiles' make/coverage.mk` == 1.
- **D-COV-4:** compare `test` job wall time and step list on the PR run vs. run 31858629366
  (M8 baseline: 1127s with 492s of coverage steps).
- No Go unit tests are appropriate — every change is Make/CI plumbing; the anti-vacuity
  criteria above are the tests.

---

## Conflict Surface

Three separate findings — the earlier draft folded these into one blanket "not applicable",
which was the wrong discharge for finding 3:

1. **Language surface: none (specific).** This design touches **no**
   parser/lexer/AST/types/elaborate/iface/codegen/eval/vm/effects code and no compilation
   entry points — only `make/coverage.mk`, `.github/workflows/ci.yml`, and docs. No syntactic
   position is extended; no existing AILANG program's behavior can change.
2. **`ailang check` gate: vacuously discharged (specific).** This doc contains no AILANG
   snippets and no "AILANG supports/doesn't support X" claims, so the live-`ailang check` hard
   gate has nothing to verify — stated so the gate is discharged explicitly rather than
   skipped.
3. **REAL overlap: the package-discovery pipeline in `make/coverage.mk`.**
   `test-coverage-xpkg` must run the IDENTICAL package list as `test-coverage`, and that list
   is currently produced by an inline shell pipeline at `make/coverage.mk:18` (V9):
   `go list -f '{{if .TestGoFiles}}{{.ImportPath}}{{end}}' ./... | grep -v /scripts | grep -v /examples/agents`.
   Two ways to share it, and the choice must be explicit:
   - **Extract the pipeline to ONE shared Make variable** (e.g. `COVERAGE_PKG_LIST`) consumed
     by both targets — **RECOMMENDED**. The follow-up sprint refactors line 18 into the
     variable (behavior-identical for `test-coverage`) and the new target references it.
   - Copy-paste the pipeline into the new target — **REJECTED**. A duplicated discovery
     pipeline is a guard-the-helper-miss-the-call-site hazard: the day someone adds a
     `grep -v` to one copy, the two targets silently measure different package universes, and
     the drift is invisible because both still exit 0 — the diagnostic then answers "is X
     exercised?" against a different denominator than the gate's, which is exactly the
     split-brain class this doc exists to close.
   XC5 makes the decision enforceable: the pipeline signature must appear exactly once in
   `make/coverage.mk`, and the killing mutation (paste-then-edit divergence) is named there.

The nearest analogue to a conflict surface beyond finding 3 is the *metric* surface, covered
above: which consumers read `coverage/coverage.out` (gate at `make/coverage.mk:41-49`, badge at
`.github/workflows/ci.yml:217-237`, Sonar at `sonar-project.properties:37`) — Option C changes
none of their inputs.

---

## Non-Goals

- Changing `sonar.coverage.exclusions` (e.g. un-excluding `cmd/ailang/**`) — a separate
  decision with its own measurement burden.
- Measuring SonarCloud's behavior on a 599 MB profile — only needed if D-COV-1 = EXECUTION,
  and then it is that sprint's first task, not an assumption.
- Raising the 29% threshold on its own merits — orthogonal to attribution semantics.
- Any edit to `make/coverage.mk` in THIS iteration — charter-forbidden; doc only.

## Deferred Decisions

- Exact name/flags of the diagnostic target — agent may choose.
- Whether the XC1/XC2/XC5 script assertions live in CI permanently or only in the sprint's PR
  verification — agent may choose (permanent CI step preferred if <5s).
- Choice of merge tool under Option E — moot unless D-COV-1 = EXECUTION; human at review then.

---

## Decomposition if D-COV-1 = EXECUTION (Option A path; 3–4 days, split)

Sprint-sizing honesty: flipping the metric is NOT a one-day change, so it decomposes:

1. **Day 1 — measure the unmeasured:** generate the 599 MB profile in a CI run, ship it to
   Sonar on a throwaway branch, and OBSERVE ingest (remember V12: failure is silent — check
   Sonar's analyzed coverage, not CI green). Evaluate Option-E merge/shrink if ingest fails.
2. **Day 2 — flip + recalibrate:** add `-coverpkg` to `test-coverage`, re-measure the delta at
   that commit, raise `COVERAGE_THRESHOLD` by it (rounded up; 29→32 per M4's snapshot).
3. **Day 3 — restore the lost signal:** the own-package-test true-positive generator dies with
   the flip, so add a replacement guard (e.g. Sonar new-code review discipline or a
   locality-view report) — designed then, not assumed here.
4. **Day 4 — buffer + trend annotation:** mark the flip date everywhere coverage trends are
   read, so pre/post numbers are never pooled (same class of confound as the eval split-brain
   memory).

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Diagnostic target gets quietly promoted into the gate later ("the number is bigger!") | Med | XC2 as a standing assertion; D-COV-1 recorded as the ratified meaning |
| D-COV-4 badge step reads a profile the gate step didn't produce (step reorder) | Low | Badge step fails loudly on missing/empty `coverage/coverage.out` (`grep "^total:"` already exits non-zero on absence) |
| 599 MB local profiles accumulate on dev machines | Low | `coverage-clean` covers `coverage-xpkg.out`; `coverage/coverage-xpkg.out` is git-ignored — verified against `.gitignore` itself, **not** inferred from Sonar config (**V16**) |
| The class recurs anyway because triagers don't know the diagnostic exists | Med | Development-workflow guide paragraph; this doc is the searchable record |

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

Build-infrastructure decision; no language surface. Scored for completeness:

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Metric semantics become an explicit, recorded decision; M4 showed both arms deterministic across replicates |
| A2: Replayability | 0 | No impact |
| A3: Effect Legibility | 0 | No impact |
| A4: Explicit Authority | 0 | No impact |
| A5: Bounded Verification | +1 | Gate keeps a checkable, stated meaning; anti-vacuity criteria (XC1–XC5) are locally verifiable |
| A6: Safe Concurrency | 0 | No impact |
| A7: Machines First | +1 | Agents triaging 0% readings get a mechanical answer instead of hand-run `go test` archaeology |
| A8: Minimal Syntax | 0 | No syntax |
| A9: Cost Visibility | +1 | Names the 599 MB artifact and the 492s CI double-run instead of leaving both implicit |
| A10: Composability | 0 | No impact |
| A11: Structured Failure | 0 | No impact |
| A12: System Boundary | 0 | No impact |

**Net Score: +4** ✅ Proceed (threshold ≥ +2)

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism introduced
- [x] A3 (Effects): no hidden side effects
- [x] A4 (Authority): no ambient access granted
- [x] A7 (Machines First): not optimizing human convenience over machine analysis

---

## Related Documents

Duplicate/coverage gate: `ailang docs search` (fell back to SimHash; binary flagged stale) plus
a direct `grep -ric coverpkg design_docs/` found **no** planned or implemented design doc
covering coverage attribution — the only `coverpkg` mentions in `design_docs/` are this item's
own mission-loop records (`v1-mission.md`, `v1-mission-log.md`, `mission-dashboard.md`). Top
SimHash hits (`m-arch-boundaries-eval-exclusion-tighten.md` et al.) are keyword artifacts about
architecture-boundary eval exclusions, not coverage attribution — genuinely distinct. See
Verification Log V13.

- Iteration 204 record and fixes: commits `3ec1dcb02` (#722), `54fdcb32c` (#723); issue #720.
- `sonar-project.properties` rationale comments (why `cmd/ailang/**` is coverage-excluded).
- Memory: "SonarCloud PR analysis can be STALE" — relevant to any Option-A ingest measurement.

---

## Verification Log

Provenance discipline: **M-rows are VERIFIED BY THE MISSION CONTROLLER, first-party, this
iteration, at commit 376e19284, darwin/arm64, go1.26.6** — carried here with their commands as
required. **V-rows are verified by the doc author (this session), same worktree and commit.**
Negative results carry a same-scope positive control.

| # | Claim | Command | Observed |
|---|-------|---------|----------|
| M1 | `test-coverage` has no `-coverpkg` | `grep -rc 'coverpkg' make/` + control `grep -rc 'coverprofile' make/` + `test -d make` | zero `coverpkg` hits; control: `make/coverage.mk:1`; dir exists |
| M2 | Sonar coverage exclusions + report path | read `sonar-project.properties` | exclusions = `cmd/ailang/**, internal/server/**, internal/storage/firestore/**, internal/storage/migrate/**, internal/agentprompt/**, internal/devtoolsprompt/**, internal/docsearch/**`; `sonar.go.coverage.reportPaths=coverage/coverage.out` |
| M3 | Gate threshold and mechanism | read `make/coverage.mk` | `COVERAGE_THRESHOLD := 29`; `test-coverage-gate` parses `total:` from `go tool cover -func`, fails below 29 |
| M4 | Full-suite A/B, 105 pkgs, 3 replicates/arm, `go clean -testcache` each | `make test-coverage` package list ± `-coverpkg=./...` | Arm A: 89/78/82s, total 45.5% every run, 0 FAIL; Arm B: 92/79/83s, total 48.1% every run, 0 FAIL. Delta +1/+1/+3s (~+1–4%), +2.6pp upward, deterministic. **Narrowing:** warm Go build cache on darwin/arm64 — steady-state, not cold-build; CI (restored cache) is the closer analogue |
| M5 | Artifact blowup is the real cost | profile size/block count both arms; `go tool cover -func` timing | 5,992,739 B / 74,336 blocks → 627,585,840 B / 7,808,904 blocks (~105×); `cover -func` 539ms → 1463ms. Sonar ingest of 599 MB: **UNMEASURED** |
| M6 | Denominator widens by exactly 9 test-less packages; none lost | package-dir diff of profiles (`comm -23` → 0 lost) | +`internal/agentprompt, internal/auth/gcp, internal/devtoolsprompt, internal/docsearch, internal/eval_harness/langreg, internal/feedback, internal/mcp_client, internal/storage/migrate, internal/version`; 4 of 9 already Sonar-coverage-excluded → badge and Sonar move by DIFFERENT amounts (Sonar's own delta NOT quantified) |
| M7 | Subject-pair probe: live exposure small post-204 | `go test ./internal/pkg ./cmd/ailang` ± `-coverpkg=./...` | exactly 2 of 85 `internal/pkg` functions change (`LoadLockFileFromPath` 66.7→77.8%, `LoadManifestFile` 88.9→100.0%); `ParseManifestFile` 100.0% BOTH arms; control: distinct pkg dirs in profile 2 → 98 |
| M8 | CI cost + structural double-run | `gh api .../actions/runs/31858629366/jobs` | `test` job = critical path, 1127s; `Check coverage gate (M-P2)` 280s + `Generate test coverage` 212s = 492s = 43.7%; coverage suite executed TWICE per run (gate dep + later step) — independent of the `-coverpkg` decision |
| V9 | Exact current target text; gate mechanism | Read `make/coverage.mk` | line 18: `go test -v -coverprofile=$(COVERAGE_FILE) -covermode=atomic $$(go list -f '{{if .TestGoFiles}}{{.ImportPath}}{{end}}' ./... \| grep -v /scripts \| grep -v /examples/agents)`; line 12: `COVERAGE_THRESHOLD := 29`; lines 41–49: awk float compare on `total:` |
| V10 | Sonar config as quoted in M2; `coverage/**` in `sonar.exclusions` | Read `sonar-project.properties` | lines 37, 44–51 match M2 verbatim; line 28: `coverage/**` excluded from scanning |
| V11 | CI runs the coverage suite twice (mechanism for M8) | `grep -n "test-coverage\|coverage" .github/workflows/ci.yml` + read | line 139: `run: make test-coverage-gate` (dep runs full suite); lines 217–219: later step runs `make test-coverage` again for the badge |
| V12 | Sonar ingest failure would be SILENT, not red CI | Read `.github/workflows/ci.yml` lines 253–260 | SonarCloud scan step is `continue-on-error: true`; comment: "Sonar is a reporting layer, not a gate" |
| V13 | NEGATIVE: no existing design doc covers `-coverpkg`/attribution | `grep -ric 'coverpkg' design_docs/ \| grep -v ':0'` + control `grep -rl 'test-coverage' design_docs/ \| head -5` | hits ONLY in `mission-dashboard.md`, `v1-mission-log.md`, `v1-mission.md` (this item's own mission records); control returns 5+ files, so the instrument sees positives in the same scope. `ailang docs search` (SimHash fallback) top hits all off-topic |
| V14 | Target folder exists; no name collision | `ls design_docs/planned/v0_33_2/` | 4 existing docs (mission-loop telemetry set); no coverage doc present |
| **V16** | `coverage/` output is git-ignored — checked against `.gitignore`, NOT inferred from Sonar config (round-2 objection: Sonar exclusions do not govern Git) | `grep -E '^/?coverage' .gitignore` **and** the authoritative `git check-ignore -v coverage/coverage-xpkg.out` | grep matches `.gitignore:20: coverage.*`; `check-ignore` → `.gitignore:19:*.out  coverage/coverage-xpkg.out`, **rc=0 (IS ignored)**; `coverage/coverage.out` likewise ignored via `.gitignore:20`. **Control**: `git check-ignore -v Makefile` → **rc=1 (not ignored)**, so rc=0 is a measurement, not a default. Consequence: the risk is REFUTED, and `.gitignore` does NOT enter Files to Modify/Create. The doc's earlier conclusion was right for the wrong reason — V10 (a Sonar-config read) was cited for a Git claim it cannot support |
| **V17** | A merged `-coverpkg` profile carries NO per-test provenance (basis for narrowing the Overview contract) | `head -4 /tmp/full_withpkg_205.out`; `grep -c '^mode:'`; `awk '{print NF}' \| sort -u` over the first 200k lines | header `mode: atomic`; data lines are `github.com/…/access_control.go:15.48,16.20 1 0`; **exactly 1** `mode:` line in the whole 599 MB file; field counts are only **{2,3}** — i.e. `file:range numStatements count` with **no originating-test column**. gpt5-6-sol's round-2 objection CONFIRMED first-party |
| M15 | Presence in a `-coverpkg` profile ≠ execution (quorum objection 1, measured on the actual 599 MB ARM-B profile `/tmp/full_withpkg_205.out` from M4/M5). Profile line format `file.go:start.col,end.col numStatements count` — presence and execution are different columns | per package: `tot=$(grep -c "ailang/<pkg>/" $P)`; `nz=$(awk -v p="ailang/<pkg>/" '$0 ~ p { if ($NF+0 > 0) n++ } END{print n+0}' $P)` | Testless (M6) pkgs — `internal/version`: 168 lines / **56 nonzero** (genuinely exercised cross-package); `internal/mcp_client`: 1024 / **0** (present, NEVER executed); `internal/auth/gcp`: 288 / **0** (present, NEVER executed); `internal/feedback`: 147 / 3; `internal/eval_harness/langreg`: 1590 / 350. Known-exercised controls, same instrument+scope: `internal/pkg` 86625 / 930; `internal/loader` 32760 / 1429. Arm A (no `-coverpkg`), same pkgs: `internal/version` 0 lines, `internal/mcp_client` 0, `internal/pkg` 825 |

---

## Quorum Verification Log

**Round 1: BLOCKED (rc=3).** Both reviewers PRESENT (`absent_reviewers` empty) — a real
two-reviewer reject, not a degraded quorum. Neither objection disputed the recommendation
(LOCALITY for the gated/badged/Sonar metric; separate non-gating `-coverpkg` diagnostic); both
targeted defects in how the doc verifies itself. One sanctioned revision pass; no direction
change, no other content touched.

- **Objection 1 (gpt5-6-sol): XC1 vacuous — profile presence ≠ execution.** With
  `-coverpkg=./...`, a testless package can appear in the profile with all counters at zero
  merely because it was instrumented, so the original XC1 ("≥1 line for a testless package")
  passed whether or not cross-package execution happened. **Measured by the mission controller
  (first-party, on the actual ARM-B profile) and CONFIRMED — worse than stated:** the
  mechanism is demonstrated twice (`internal/mcp_client` 1024 lines / 0 nonzero,
  `internal/auth/gcp` 288 / 0 — present, never executed), and the original canary
  `internal/version` happens to be genuinely exercised (56/168 nonzero), so XC1 as written
  passed for the right reason BY LUCK — it would have passed identically, and falsely, had it
  named `internal/mcp_client`. Commands and full numbers: M15. **Resolution:** XC1 rewritten
  to assert a trailing count > 0, with `internal/mcp_client` as a named
  present-but-unexecuted negative control (re-point, never delete); every criterion
  (XC1–XC5) now carries an explicit vacuity check ("what would this still pass under if the
  claim were false?"); the Examples section and Testing Strategy re-pointed at
  counts-not-presence, including the triage warning that a raw-profile grep proves nothing.
- **Objection 2 (gemini-3-1-pro): Conflict Surface improperly discharged.** The blanket "not
  applicable to infrastructure" hid a real overlap: `test-coverage-xpkg` needs the same
  package list as `test-coverage`, whose discovery pipeline lives inline at
  `make/coverage.mk:18` (V9). **Resolution:** Conflict Surface rewritten as three separate
  findings — (1) no language surface, (2) `ailang check` gate vacuously discharged, both kept
  as specific discharges; (3) the REAL machinery overlap, with an explicit decision: extract
  the pipeline to ONE shared Make variable consumed by both targets (RECOMMENDED); copy-paste
  REJECTED as a guard-the-helper-miss-the-call-site hazard whose drift is silent because both
  copies still exit 0. New failable criterion XC5 enforces it (pipeline signature appears
  exactly once; killing mutation = paste-then-edit divergence). Files-to-Modify updated to
  include the variable extraction.

**Round 2: BLOCKED (rc=3), metered $0.0805.** Both reviewers PRESENT again
(`absent_reviewers` empty). Round 1's two fixes were ACCEPTED — neither reappeared. Two NEW
objections, both again non-directional (an overclaimed capability and a mis-sourced premise),
and both carrying a concrete reviewer-authored `proposed_fix`.

**Resolved under Gate 2's NARROW-REFINEMENT CARVE-OUT, not by force-passing.** The bounded path
(one revision + one re-quorum) was spent, and both surviving objections satisfy the carve-out's
two conditions: each carries the reviewer's own `proposed_fix`, and neither disputes the design
DIRECTION — one narrows a capability claim, the other corrects which file a premise was read
from. The controller applied the reviewers' own text; no controller-invented resolution, and no
objection overridden. (Carve-out first use was ratified by Mark earlier in this mission —
charter line 739 — so no re-ratification is owed.)

- **Objection 3 (gpt5-6-sol): the Overview claimed the diagnostic answers "and by whom", which
  the design cannot support.** A merged `-coverpkg` profile carries only aggregate source-block
  counts; M15 and XC1 establish nonzero execution, never which test caused it. **Measured by the
  controller and CONFIRMED (V17):** the 599 MB profile has exactly **one** `mode:` line and
  field counts of only {2,3} — `file:range numStatements count`, no originating-test column.
  **Resolution — the reviewer's arm 1, verbatim:** the contract is narrowed everywhere to
  *"is this 0%-locality reading executed by any test elsewhere?"* and "and by whom" is removed,
  with an explicit paragraph stating the diagnostic answers *whether*, never *by whom*. The
  reviewer's arm 2 (redesign with bounded per-test-package probes plus a block→test-package
  index, and a criterion proving attribution) is NOT silently dropped: it is recorded in Future
  Work as the design that would be required to make the stronger claim. Choosing between two
  arms the reviewer itself offered is not overriding it; taking arm 2 would convert a cheap
  triage aid into a second sprint, which is out of scope for this doc's recommendation.
- **Objection 4 (gemini-3-1-pro): the 599 MB mitigation rested on an unverified premise** —
  the doc asserted `coverage/` is git-ignored and cited **V10**, which reads
  `sonar-project.properties`. Sonar exclusions do not govern Git, so the citation could not
  support the claim. The reviewer is right about the *evidence* and the doc turned out to be
  right about the *conclusion*: **measured (V16)**, `git check-ignore -v coverage/coverage-xpkg.out`
  → rc=0 via `.gitignore:19:*.out`, `coverage/coverage.out` → ignored via `.gitignore:20`, with
  the control `git check-ignore -v Makefile` → rc=1 proving rc=0 is a measurement rather than a
  default. **Resolution — the reviewer's fix, verbatim:** V16 added against `.gitignore` itself,
  Risks & Mitigations re-pointed from V10 to V16. Per the fix's own conditional ("*if* the
  directory is not currently ignored, add `.gitignore` to Files to Modify"), and because it
  **is** ignored, `.gitignore` correctly does NOT enter Files to Modify/Create. This is the
  "right for the wrong reason" class the doc is otherwise about, caught in the doc itself.

**Status after round 2:** direction unchanged and twice-unchallenged across two full rounds with
both reviewers present in each. The doc lands as **Planned**; no sprint runs on it until D-COV-1
is answered.

---

## Future Work

- If D-COV-1 = EXECUTION: the decomposed Option-A sprint above, starting with the Sonar-ingest
  measurement.
- Possible standing CI assertion generalizing XC2: "the gated profile's package set matches the
  own-package expectation" — guards against any future silent semantics drift, not just this flag.
- **Per-test provenance ("by whom"), explicitly out of scope here** — gpt5-6-sol's round-2 arm 2.
  A merged `-coverpkg` profile cannot attribute a block to an originating test (V17), so
  answering "which test exercises this?" needs a different design: bounded per-test-package
  coverage probes into SEPARATE profiles, plus an index from exercised source blocks back to
  originating test packages, plus a criterion demonstrating a known block is attributed to the
  expected test package rather than merely reported nonzero. Cost scales with the number of test
  packages (105 today) and the M5 artifact problem multiplies accordingly, which is why it is
  deferred rather than folded in. Raise it only if triage repeatedly needs provenance and the
  zero-versus-nonzero answer proves insufficient — i.e. on demand evidence, not speculatively.
