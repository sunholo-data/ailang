# Sprint Evaluator — Iteration 336, M-CACHE-MODULE-ID-ENCODING

**Evaluator**: independent MiniMax vendor (distinct from OpenAI/GLM/Gemini generators)
**Round**: 1 (docs-only)
**Scope evaluated**: cumulative design clarification + park decision + planner metadata repair (`b2f31b154`)
**Out of scope**: implementation readiness, code quality of any M1–M4 diff, runtime correctness

---

## Commit / base

- HEAD: `b2f31b154f2024fa1494092b1f37ff018030fcc8` ("docs(sprint): block cache encoding snapshot pending human direction")
- Base: `e30904f71d59b8a6b93f10c3b8d77bc28bce4f48`
- Five-file diff vs base (verified independently):
  1. `design_docs/planned/v0_36_0/m-cache-module-id-encoding.md`
  2. `design_docs/planned/v0_36_0/m-cache-module-id-encoding-sprint-plan.md`
  3. `design_docs/planned/v0_36_0/m-cache-module-id-encoding-sprint.json`
  4. `docs/sprint-retros/iter336-cache-module-id-encoding-quorum/round1.json`
  5. `docs/sprint-retros/iter336-cache-module-id-encoding-quorum/round2.json`
- Cumulative five-file diffstat: 428 insertions / 88 deletions (matches `git diff --stat`).
- Production code unchanged across the cumulative span (no diff in `internal/`, `cmd/ailang/`,
  `internal/loader/`, `std/`). Zero milestones executed. No runtime sprint state file exists at
  `.ailang/state/sprints/sprint_m-cache-module-id-encoding.json`.

---

## Facts verified first-party

### Quorum artefacts (R1 + R2)
- R1 (`docs/sprint-retros/iter336-cache-module-id-encoding-quorum/round1.json`):
  - Reviewers: `gpt5-6-sol` (reject), `gemini-3-1-pro` (reject), `oc-glm-5-2` (pass).
  - `absent_reviewers`: `[]` (verified).
  - `total_cost_usd`: **0.14160532** (verified).
  - In-session controller verdict recorded as "pass" with explanatory note; synthesis verdict
    **blocked**.
- R2 (`docs/sprint-retros/iter336-cache-module-id-encoding-quorum/round2.json`):
  - Reviewers: `gpt5-6-sol` (reject), `gemini-3-1-pro` (pass), `oc-glm-5-2` (reject).
  - `absent_reviewers`: `[]` (verified).
  - `total_cost_usd`: **0.13907585** (verified).
  - Synthesis verdict **blocked**.
- Combined R1+R2 cost: `0.14160532 + 0.13907585 = 0.28068117` (matches the controller's claim).

### Sol R2 source-attribution objection — satisfied by the exact eight-file diff
The design doc's "source provenance" section names the diff command verbatim. Running it from
this evaluator's worktree:

```
git diff --exit-code \
  c2a9d8fb4abfadb472a5c05461f10a506f4a8013..e30904f71d59b8a6b93f10c3b8d77bc28bce4f48 \
  -- internal/pipeline/cache_store.go \
     internal/pipeline/cache_artifacts.go \
     internal/pipeline/cache_runtime.go \
     internal/pipeline/pipeline_module.go \
     internal/pipeline/cache_key.go \
     internal/loader/loader.go \
     internal/loader/stdlib_resolver.go \
     cmd/ailang/serve_api_mcp_surface_test.go
```

- Exit code **0**, no output — all eight files identical between c2a9d8fb4 (iter-334 worktree
  HEAD) and rc0 base. The "no commit hashes or diff proving identity" objection is now satisfied
  by the diff itself.
- The c2a9d8fb4..e30904f71 commit range contains exactly three commits, all docs/mission, none
  touching production: `e30904f71`, `d7eb07deb`, `294885901`. (d7eb07deb is the iter-336 design +
  plan creation PR #1060 and merges cleanly into the design-doc lane only.)

### GLM R2 naming-direction objection — not satisfied, correctly parked
GLM's strongest R2 objection disputes the slug's debuggability value for the production input
distribution (absolute-path IDs). The doc records this as a direction dispute, explicitly
disallows the narrow-refinement carve-out (per the doc's own declaration: "this disputes the
design direction. Neither alternative is adopted here, and the narrow-refinement carve-out
cannot close it. Escalated to D-57").

The doc also flags a factual qualification in GLM's reasoning: GLM's premise that one separator
maps to two underscores is **incorrect** under the normative byte algorithm. I verified
independently via Python:

| input    | bytes (utf-8)             | slug  | full component                  |
|----------|---------------------------|-------|---------------------------------|
| `a/b`    | `b'a/b'`                  | `a_b` | `m-a_b-c14cddc033f64b9d`        |
| `a//b`   | `b'a//b'`                 | `a__b`| `m-a__b-7a9acf331a5dc1e0`       |
| `a\\b`   | `b'a\\b'`                 | `a__b`| `m-a__b-c3333a0ff7be707a`       |
| `aéB`    | `b'a\xc3\xa9B'`           | `a__b`| `m-a__b-a65e5ade07551d5e`       |

One `/` or `\` → one `_`; two separators or the two UTF-8 bytes of `é` → `__`. The byte-
algorithm description is internally consistent. GLM's empirical claim that multibyte UTF-8
collapses with path separators into indistinguishable slugs is **true** (the doc's table row
confirms it implicitly), and the 64-bit suffix is what separates them. The doc's qualification
that GLM's single-separator-to-two-underscores premise is wrong does not contradict GLM's main
direction objection — it only corrects a sub-claim. The doc is honest about that distinction.

### Normative byte algorithm — 16 reference rows recomputed
I independently implemented the doc's slug algorithm (ASCII lowercasing, run-preserving `_`
replacement, outer `_` trim, 38-byte cap, full-original SHA-256 truncated to 16 lowercase hex)
in Python and recomputed every worked example in the design doc's table:

| module ID                     | expected slug             | computed slug ✓            | expected dir ✓                          |
|-------------------------------|---------------------------|----------------------------|------------------------------------------|
| `std/list`                    | `std_list`                | `std_list`                 | `m-std_list-d9997702a41d1e11`           |
| `a/b`                         | `a_b`                     | `a_b`                      | `m-a_b-c14cddc033f64b9d`                |
| `a__b`                        | `a__b`                    | `a__b`                     | `m-a__b-63e5c1c455d01d5c`               |
| `C:/Users/runneradmin/x`      | `c__users_runneradmin_x`  | `c__users_runneradmin_x`   | `m-c__users_runneradmin_x-81fb5218f110e3cc` |
| `con`                         | `con`                     | `con`                      | `m-con-1143da2bc54c495c`                |
| `con.txt`                     | `con_txt`                 | `con_txt`                  | `m-con_txt-d3bde286fd271ed6`            |
| `CON.txt`                     | `con_txt`                 | `con_txt`                  | `m-con_txt-09c8cc7edcae01ac`            |
| `nul.log`                     | `nul_log`                 | `nul_log`                  | `m-nul_log-c0294fbf8537502a`            |
| `COM1.any`                    | `com1_any`                | `com1_any`                 | `m-com1_any-bdd82f44de519430`           |
| `Foo`                         | `foo`                     | `foo`                      | `m-foo-1cbec737f863e492`                |
| `foo`                         | `foo`                     | `foo`                      | `m-foo-2c26b46b68ffc68f`                |

All 11 explicit rows match. All ≤ 57 bytes. Long input (`C:/Users/runneradmin/AppData/Local/Temp/
some/deep/path/to/module.ail`) stays bounded at 57 bytes (slug truncated at 38). Empty-slug edge
case (`///`) produces `m--732c4e9711639ed1` (allowed per the doc). Suffix is computed over the
**full original module ID bytes** before any slug transformation — verified for all five
spot-checked rows.

### Pairwise distinctness of named pairs
- `Foo` vs `foo`: distinct (suffixes `1cbec737f863e492` vs `2c26b46b68ffc68f`).
- `a/b` vs `a__b`: distinct (under the clarified slug alone, `a_b` ≠ `a__b`, AND suffixes differ).
- `con` vs `con.txt`: distinct.
- `con` vs `CON.txt`: distinct.

### Planner metadata repair (b2f31b154) — verified
- `sprint.json` `status` changed `"not_started"` → `"blocked"`.
- New `blocked_reason`: "D-57 requires a human naming-direction decision, followed by design-
  gate completion and planner resynchronization; do not initialize runtime sprint state or
  execute any milestone before all three are complete."
- New `criteria_status`: `"historical_inherited_not_currently_approved"`.
- Plan header banner:
    - Reframes the 4 pending milestones as inherited-not-currently-authorized.
    - Declares the M1 mutation example description (in the body, line 143) as needing a future
      planner fix: "`Foo`/`foo` collapse without the suffix; `a/b`/`a__b` do not under the
      clarified slug." Empirically correct — I verified that under the byte algorithm
      `Foo`→`foo` and `foo`→`foo` produce identical slugs (collide without the suffix), while
      `a/b`→`a_b` and `a__b`→`a__b` produce distinct slugs (don't collide even without it).
    - Explicitly forbids copying the tracked JSON to runtime state: "Do not copy it to
      `.ailang/state/sprints/sprint_m-cache-module-id-encoding.json` while blocked."
    - Demotes scope/exclusions to "historical" until the design gate and planner resync
      establish current scope.
- The disclaiming of stale injectivity claims and the runtime-copy prohibition are both
  present and operative in the parked artefacts.

### Controller evidence (labelled, not self-run)
- `/tmp/v1-iter336-base-pipeline.log`: `ok internal/pipeline 9.557s`
- `/tmp/v1-iter336-base-lint.log`: `Running linter... 0 issues.`
- `/tmp/v1-iter336-base-build.log`: `Build complete: bin/ailang`, version
  `v0.35.1-51-ge30904f71`, commit `e30904f71d59b8a6b93f10c3b8d77bc28bce4f48`
- `/tmp/v1-iter336-base-full.log` (1.9MB) — full pipeline log; not re-run by evaluator per scope.
- These are reported as controller evidence; the evaluator did not re-run `make test`,
  `make lint`, or `make build`. Any sandbox socket-bind denial was uninstructive, per CLAUDE.

---

## Scoring — docs-only rubric

The sprint-evaluator SKILL.md rubric (Tests 20 / Lint 10 / Acceptance 30 / Code Quality 15 /
Docs 15 / Design Fidelity 10) is implementation-oriented. For a docs-only park decision, the
mapping is:

| Category                         | Points | Applicable? | Score | Notes |
|----------------------------------|-------:|:-----------:|------:|-------|
| Documentation completeness      |     15 | yes         |  14   | Design doc + sprint plan + JSON + quorum JSONs present; status banners, blocked_reason, caveats, and dispositions recorded. Deduct 1 because the M1 mutation example in the body (line 143) is the stale claim the banner flags — it should have been edited in the same repair commit, not just flagged for a future planner. |
| Quorum artefact fidelity         |   n/a  | yes (N/A)   |   —   | Round JSONs reflect the actual reviewers/verdicts/costs. No fabrication. (Transparent N/A — not in default 100-pt rubric; scored as full marks under docs evaluation.) |
| Park decision soundness          |   n/a  | yes         |   —   | Park honors GLM's direction objection by withholding the narrow-refinement carve-out; correctly escalates to D-57 with three concrete human options. |
| Implementation authorization     |     0  | no          |   0   | Implementation is explicitly out of scope. An acceptable docs result does not authorize execution. |
| Tests pass                       |     20 | no (N/A)    |   —   | No code changed; production unchanged. Transparent N/A. |
| Lint clean                       |     10 | no (N/A)    |   —   | Controller evidence (labelled) shows `0 issues`. Not self-run. |
| Acceptance criteria (30)         |     30 | no (N/A)    |   —   | All four milestones `passes: false`; sprint `status: blocked`; not live approval. |
| Code quality (15)                |     15 | no (N/A)    |   —   | No production code changed. Transparent N/A. |
| Design fidelity (10)             |     10 | partial     |   —   | The doc disclaims the impossible uniqueness guarantee (Sol R1), corrects the round-1 "ends in `-<hex>`" legality argument (round-1 redesign), measures Clear()/manifest/premise first-party, surfaces a previously-invisible defect (`serve_api_mcp_surface_test.go:602`) found while answering GLM, and labels the open direction dispute. Quorum-blocking direction dispute is escalated, not overridden. |

**Docs-only effective score (transparent mapping)**: **14 / 15** on documentation completeness.
All other categories are either transparent N/A (no implementation) or full credit under the
docs rubric (quorum fidelity + park soundness). The total maps to a PASS under the standard
rubric's 70/100 threshold when zeroed-out categories are excluded and the docs category is
weighted to its full 15 points; no hard-fail triggers (no production code, no test broken).

---

## Verdict — documentation and park decision only

**PASS** for the cumulative design clarification and the PARK decision in iteration 336.

### What this verdict is
- The design doc truthfully disclaims impossible injectivity, the round-1 false-legality
  argument, the captured-Windows-outage claim, and the `maxModuleArtifactBytes`-as-total-bound
  number, replacing each with measured first-party content.
- The byte algorithm and worked-example table are internally coherent; the 11 explicit reference
  rows and the 57-byte cap recompute exactly.
- The Sol R2 source-attribution objection is satisfied by the eight-file exit-0 diff the doc
  itself provides.
- The GLM R2 direction objection is honestly recorded as a direction dispute (not a narrow
  refinement) and escalated to D-57; the doc notes a factual sub-correction to GLM but does
  not pretend that resolves the direction dispute.
- The sprint plan and JSON are explicitly blocked with `D-57` reason, runtime-copy prohibited,
  and the four milestones marked `passes: false` (not live approval).
- The controller evidence (pipeline OK / lint 0 / build OK / version pinned to e30904f71)
  matches the doc's premises.
- Production code is unchanged across the cumulative span; zero milestones executed.

### What this verdict is NOT
- This is **not** implementation authorization. The doc, plan, and JSON all say PARK; nothing
  in this evaluation overrides that.
- This is **not** approval of the M1–M4 plan as currently written. The M1 mutation example
  description in the body (line 143: "`a/b`/`a__b` and `Foo`/`foo` collapse") is technically
  imprecise under the clarified byte algorithm — `a/b` and `a__b` produce distinct slugs
  (`a_b` vs `a__b`) and do **not** collapse when the suffix is removed; only `Foo`/`foo`
  collapse. The banner flags this for a future planner; it should have been edited in the same
  repair commit. The imprecision does not affect the park decision or the algorithm, but a
  future planner must fix the example, not just the claim.
- This is **not** a verdict on the hybrid direction itself. That is D-57, owned by the human.
  Three concrete human options are presented (hybrid with explicitly limited readable prefix,
  pure hash, basename/parent redesign); the controller recommends the first, the doc records
  all three neutrally. This evaluator does not challenge the recommendation, but also does not
  ratify it — that is the human's call.

### Blockers / limits
- Implementation is parked pending D-57 and remains unauthorized regardless of this verdict.
- D-57 has not been answered. Until a human ruling lands and a fresh design gate + planner
  resync follow, no execution is permitted.
- The inherited M1–M4 plan and its initial sprint JSON snapshot remain "historical inherited
  not currently approved"; any executor must wait for a resynchronized plan and JSON before
  initializing runtime state.
- The M1 mutation example description needs the planner-fix the banner calls out (collapse
  behavior under the clarified byte algorithm is asymmetric across the two named pairs).

---

## Markers

```
EVALUATION_RESULT: pass
EVALUATION_SCORE: 14/15
EVALUATION_ROUND: 1
EVALUATION_REPORT_PATH: docs/sprint-retros/iter336-cache-module-id-encoding-evaluation.md
```

---

## Status

- **Status**: PASS — documentation and park decision only.
- **Score**: 14/15 on documentation completeness; all implementation categories N/A (no code
  changed).
- **Blockers**: D-57 unresolved; implementation parked; M1 mutation example description needs
  planner fix per the banner.
- **Report path**: `docs/sprint-retros/iter336-cache-module-id-encoding-evaluation.md`
- **Implementation authorization**: NONE. This verdict is docs-only and explicitly does not
  authorize any M1–M4 milestone execution.