# Sprint Plan: M-NIGHTLY-UNMEASURED-CATEGORY-GATE

**GitHub issue**: [sunholo-data/ailang#551](https://github.com/sunholo-data/ailang/issues/551)
**Design doc**: none — bug fix inside `tools/nightly_classify.py`, the same lane as
[#524](m-nightly-run-validity-gate-sprint-plan.md) (iter-119) and `#548` (iter-127). No quorum.
**Sprint ID**: `M-NIGHTLY-UNMEASURED-CATEGORY-GATE`
**Branch**: `sprint/m-nightly-unmeasured-category-gate`
**Base**: `origin/dev` @ `c7fc3b954`
**Planner model**: `claude-opus-5`
**Planned**: 2026-08-01 (mission-control iteration 128)
**Estimate**: **~0.6 working days (5h)**, ~395 LOC + ~14 KB frozen fixtures. One bounded executor run.
**Risk**: low-medium. No Go behaviour change, no compiler surface, no GPU. Touches the live nightly
path (`tools/launchd/nightly-eval.sh`) and one shell-parsed output contract.

---

## 0. Premise re-verification (first-party, this worktree, at `c7fc3b954`)

The controller labelled every fact. I re-checked all of them. **Two of the three candidate fixes are
refuted on measured data, including the controller's own preferred one.** The controller's stated
facts are otherwise confirmed, with one numeric correction to the issue body.

### CONFIRMED

| # | Premise | How verified |
|---|---|---|
| C1 | `nightly_classify.py:21` is `INFRA_CATEGORIES = {"api_error", "timeout", "executor_error"}`; `non_agentic` absent | Read the file |
| C2 | `run_validity()` counts only `infra_tainted()` categories → 2026-08-01 scored `0/12`, VALID | Ran `run_validity(parse_results_dir('/tmp/nightly_eval_20260801_rag_on/agent'), 0.30)` → `(True, '', 0, 12)` |
| C3 | Live repro emits **no INVALID line** on 08-01; the 07-29 control fires | Re-ran both. 08-01 → `HEALTH` + 11 verdicts, **5 `SUSPECTED-FLAKE` + 6 `INSUFFICIENT-HISTORY`**, no INVALID. 07-29 → `INVALID infra_outage 42/42 0.167 0.643` |
| C4 | The controller's "5 SUSPECTED-FLAKE verdicts" | Exactly 5. The other 6 lines are `INSUFFICIENT-HISTORY` |
| C5 | `results/` live in the `agent/` **subdirectory**; the parent yields a misleading `INVALID zero_files 0/0` | Reproduced both |
| C6 | `non_agentic` 0/336 across 8 prior nights, then 12/12 | Recomputed from `~/.ailang/state/nightly-eval-history.jsonl` (read-only) |
| C7 | Containment is already applied; do not redo | History shows `2 invalid nights excluded` (07-29 + 08-01) |
| C8 | **`#524` deliberately rejected the pass-rate disjunct** — the controller flagged this as unverified/inherited. **It is true, and load-bearing.** | `m-nightly-run-validity-gate-sprint-plan.md:204-217`: *"I am implementing the first disjunct only, and rejecting the second… A pass-rate-collapse trigger cannot distinguish 'we failed to measure the subject' from 'the subject genuinely broke catastrophically'."* Registered as risk **R-3** at line 477: *"the pass-rate disjunct gets added back by a reviewer reading #524 literally."* Note issue #524's own text **does** propose it — reading the issue alone would reintroduce the bug |
| C9 | `validity_backstop_test.go:201` asserts `non_agentic` must not be quarantined | Read it — **but see R-A: it governs a different mechanism** |
| C10 | The tests exist and **run in CI** | `make/test.mk:20` → `test-nightly-classifier`; `.github/workflows/ci.yml:136-152` invokes it **plus an anti-vacuity floor** (`--- PASS:` count ≥ 70). Baseline measured: **74 tests, 74 PASS lines, 4.0 s** |

### REFUTED

**R-A. Candidate (c) — "all trials `duration_ms == 0`" — DOES NOT FIRE ON THE INCIDENT IT WAS
PROPOSED FOR, and the issue's supporting number is wrong.**

The issue states *"12/12 benchmarks (**24/24 trials**) failed"* and *"Every trial… with
`duration_ms: 0`"*. Measured over the real directory:

| night | trials | passes | `duration_ms == 0` | categories |
|---|---|---|---|---|
| 2026-08-01 | 24 | **1** | **22 (91.7%)** | `non_agentic` 22, `compile_error` 1, `none` 1 |

**Two trials ran** (one pass at 250 668 ms, one `compile_error`). The issue's own history line says
`passes=1/24`, which contradicts its "24/24 failed" claim. So a literal *all-trials* rule is
**false on 2026-08-01** and would have shipped a gate that does nothing.

Worse, zero-duration is **not rare on healthy nights** — it is how `thrash_aborted` is recorded:

| night | zero-duration trials | which categories |
|---|---|---|
| 2026-07-28 (good) | 18/84 (**21.4%**) | `thrash_aborted` 16, `api_error` 2 |
| 2026-07-30 (good) | 11/84 (13.1%) | `thrash_aborted` 9, `api_error` 2 |
| 2026-07-31 (good) | 14/84 (16.7%) | `thrash_aborted` 13, `api_error` 1 |

A *fractional* duration gate is therefore hostage to `thrash_aborted`, a legitimate token-budget
outcome that already reaches 21% of trials. **Candidate (c) is dropped.**

**R-B. Candidate (b) — the controller's preferred category-concentration gate — produces FALSE
POSITIVES on 2 of 3 good nights at a `#524`-style threshold, and its rescuing clause reintroduces
exactly the failure mode `#524` rejected.**

Bench-level concentration of the top failing category, measured on every surviving real night:

| night | benches | top category (ANY trial) | fraction |
|---|---|---|---|
| 2026-07-28 **(good)** | 42 | `thrash_aborted` 13 | **0.31** |
| 2026-07-30 **(good)** | 42 | `thrash_aborted` 8 | 0.19 |
| 2026-07-31 **(good)** | 42 | `thrash_aborted` 13 | **0.31** |
| 2026-07-29 (outage) | 42 | `api_error` 42 | 1.00 |
| 2026-08-01 (outage) | 12 | `non_agentic` 12 | 1.00 |

A category-**agnostic** concentration gate at 0.30 marks **07-28 and 07-31 INVALID**. Both were
good nights that produced real signal. The only thing that saves (b) is its *unprecedented* conjunct
— `thrash_aborted` appears every night, so it is excused. Which means **the entire gate is carried by
"this category is new"**, and that clause has a fatal case:

> A genuine AILANG regression that makes 40/42 benchmarks fail with `verify_error` or
> `constraint_violation` — neither of which has appeared in any of the 8 trailing nights — is marked
> INVALID and **files nothing**.

That is verbatim the failure mode `#524` §2 rejected (C8), reached through a different door. (b) also
forces history into `run_validity()`, which is currently pure and runs **before** history load
(`main()` lines 612 vs 618), and would need a defined behaviour when history is DEGRADED — a silent
weakening of a gate, which `CLAUDE.md` §2 forbids. **Candidate (b) is dropped.**

**R-C. The `validity_backstop_test.go` objection to candidate (a) is real but MIS-AIMED, and the
sharper objection is different.**

That test governs Go's **per-row** `applyValidityBackstop` (`validity.go:133`), which quarantines
`api_error` rows at bank time. It has no connection to the Python **run-level** gate. The genuine
objection to (a) is in Python: `INFRA_CATEGORIES` is used by **two** callers —

- `run_validity()` (run-level gate), and
- `infra_tainted()` → `persistent_failures()` (**per-bench** filing suppression).

So adding `non_agentic` to that one set would *permanently silence every individual non-agentic
benchmark*, on every night, forever — a genuine 0-shot model failure could never be filed again.
**That** is why (a) is wrong: not the Go test, but the fact that one set is doing two jobs.

### MATERIALLY CORRECTED

**R-D. The fixture inventory is smaller than stated, and it is decaying.** The controller said real
dirs exist "for both a good night (07-31), a known api_error outage (07-29), and this non_agentic
outage (08-01)". True — but `/tmp/nightly_eval_2026072{4,5,6,7}_rag_on/agent` **still exist as empty
directories**: their JSON has already been reaped. Only **07-28, 07-29, 07-30, 07-31, 08-01** have
data. `extract_history.py` already carries the warning *"Never regenerate the committed replay
fixture after the live banks expire."* Freezing these five nights is time-critical and is why M3
exists.

---

## 1. The fix

`INFRA_CATEGORIES` is one set doing two jobs. Split it.

```python
# Per-BENCH taint. Unchanged — suppresses filing for one benchmark.
INFRA_CATEGORIES = {"api_error", "timeout", "executor_error"}

# Run-level ONLY. A category here means "the harness did not obtain a measurement
# of the subject", so a suite-wide concentration of it means the night is unmeasured.
RUN_UNMEASURED_CATEGORIES = INFRA_CATEGORIES | {
    "non_agentic",       # 0 turns / 0 tool calls: at ROW level a real outcome
                         # (validity_backstop_test.go:201); at 100% of a suite it is
                         # the tool-delivery branch its own docs name
                         # (error_categorizer.go:5-9).
    "quota_exhausted",   # metrics.go:180 — "says nothing about the model's capability"
    "rate_limit",        # metrics.go:181 — transient 429
    "cost_killed",       # metrics.go:182 — eval-side budget stop, not a subject measurement
}
```

`run_validity()` counts `RUN_UNMEASURED_CATEGORIES`; `infra_tainted()` keeps counting
`INFRA_CATEGORIES`. **No new knob, no history dependency, no signature change.**

### The discriminator, stated plainly

`#524` blessed the infra fraction as safe to gate on because it is *a direct measurement of
unmeasurability — the harness told us it could not reach the model*. Every category added above meets
that same test, taken from the taxonomy's **own comments**. Every category NOT added is a measurement
**of the subject** and stays gateable-through: `compile_error`, `runtime_error`, `logic_error`,
`verify_error`, `constraint_violation`, `refused`, `step_exhausted`, `thrash_aborted`,
`resource_limit` (metrics.go:186 — *"A model failure (unbounded allocation)"*).

### Threshold: stays 0.30, unchanged

Measured `RUN_UNMEASURED_CATEGORIES` bench fraction under the proposal:

| night | benches | today | **proposed** | fraction |
|---|---|---|---|---|
| 2026-07-28 | 42 | VALID | VALID | 0.048 |
| 2026-07-29 | 42 | INVALID | INVALID | 1.000 |
| 2026-07-30 | 42 | VALID | VALID | 0.048 |
| 2026-07-31 | 42 | VALID | VALID | 0.024 |
| **2026-08-01** | 12 | **VALID (the bug)** | **INVALID** | **1.000** |

**Zero false positives on all three good nights; fires on both outages.** 0.30 sits **6.25×** above
the worst good night (0.048) and **3.3×** below both incidents — the identical margin `#524`
justified, so no threshold re-litigation is needed and no new flag is introduced.

**Deliberate non-change**: `--invalid-infra-fraction` keeps its name. `nightly-eval.sh:530-538` never
passes it, so a rename buys nothing operationally and costs either a back-compat alias or a silent
break for any doc/skill that does pass it. Its help text is updated instead. **Executor: do not
rename this flag.**

### Both-directions control (measured, not asserted)

42/42 benchmarks failing on each non-unmeasured category → **VALID, and all 42 still filable**:
`compile_error`, `logic_error`, `runtime_error`, `verify_error`, `constraint_violation`,
`thrash_aborted`, `step_exhausted`, `resource_limit`, `refused`. A catastrophic genuine regression is
still detected and still pages. A normal night with 2 `non_agentic` benches → **VALID**, and
`persistent_failures` still returns those 2 benches, preserving the `validity_backstop_test.go`
stance at row level.

---

## 2. Milestones

### M1 — Split the sets and gate the run (~40 impl / ~120 test LOC)

Implement §1 in `tools/nightly_classify.py`. `run_validity()` counts the union set; `infra_tainted()`
is untouched.

| # | Acceptance criterion | Runnable check |
|---|---|---|
| M1.1 | A 12-bench all-`non_agentic` run is INVALID with reason `infra_outage` and `12/12` | `--- PASS: test_Gate_non_agentic_wipeout_is_invalid` |
| M1.2 | Every member of `RUN_UNMEASURED_CATEGORIES` fires **alone**, one `subTest` each, paired with `logic_error` (mirrors the existing `test_Taint_timeout_and_executor_error_also_taint` shape so no member is special-cased) | `--- PASS: test_Gate_every_unmeasured_category_fires` |
| M1.3 | All nine non-unmeasured categories at 42/42 stay **VALID** and yield 42 filable benches | `--- PASS: test_Gate_catastrophic_clean_regression_stays_valid` |
| M1.4 | A normal night (38 pass / 2 `non_agentic` / 2 `compile_error`) is VALID **and** `persistent_failures` returns both `non_agentic` benches | `--- PASS: test_Gate_non_agentic_still_files_on_a_normal_night` |
| M1.5 | `INFRA_CATEGORIES == {"api_error","timeout","executor_error"}` exactly, and `INFRA_CATEGORIES & RUN_UNMEASURED_CATEGORIES == INFRA_CATEGORIES` with the two sets **not equal** | `--- PASS: test_Gate_infra_taint_set_is_unchanged` |
| M1.6 | 0.30 boundary is inclusive for the union set on both sides (mirrors `test_Validity_boundary_is_inclusive`) | `--- PASS: test_Gate_union_boundary_is_inclusive` |

**Anti-vacuity — what each test still passes under if the fix is reverted:**
M1.1 **FAILS** on `c7fc3b954` (returns VALID) — this is the required fail-on-main test.
M1.2 **FAILS** for the four added members.
M1.3 / M1.4 / M1.5 / M1.6 **PASS on main** — they are deliberately *pins*, not detectors. M1.3 is what
fails if someone adopts candidate (b); M1.4 and M1.5 are what fail if someone adopts candidate (a).
Their value is failing on the **wrong fix**, and the executor must state this in the docstrings.

### M2 — Name the category on the INVALID line (~15 impl / ~50 test LOC)

Today the operator sees `INVALID infra_outage 12/12 0.042 0.643` and cannot tell an API outage
(action: check the provider) from an agent-mode misconfiguration (action: fix `opencode`). The banner
also *asserts* `infra-tainted 12/12` (`nightly-eval.sh:552`), which on 08-01 would be **false** — 0
of 12 were infra.

Append a 6th tab field: the dominant unmeasured category, e.g.
`INVALID\tinfra_outage\t12/12\t0.042\t0.643\t[non_agentic]`. Update `nightly-eval.sh:542` to read `$6`
and reword the banner from "infra-tainted" to "unmeasured … [category]".

| # | Acceptance criterion | Runnable check |
|---|---|---|
| M2.1 | CLI INVALID line carries `[non_agentic]` as field 6 on the 08-01 shape | `--- PASS: test_Gate_invalid_line_names_dominant_category` |
| M2.2 | Shell banner names the category and no longer claims "infra-tainted" for a non-infra cause | `--- PASS: test_Routing_invalid_banner_names_unmeasured_category` |
| M2.3 | A legacy **5-field** INVALID line still routes: exactly 1 `messages send`, 1 `--type note`, zero `--type bug`/`--github`/`public-feedback` | `--- PASS: test_Routing_invalid_run_files_nothing` (existing test, unmodified input) |
| M2.4 | `test_Vocabulary_every_python_reason_exists_in_validity_go` still passes — **no new reason string**, so `validity.go` is not touched | existing test |

**Anti-vacuity:** M2.1/M2.2 **FAIL on main** (no 6th field exists). M2.3 **passes on main** — it is the
back-compat guard and must be left byte-identical, so it fails only if field order is changed rather
than appended.

### M3 — Freeze the real nightly corpora as in-repo fixtures (~60 impl / ~70 test LOC + ~14 KB data)

`/tmp` is being reaped (R-D: four nights already gone) and CI cannot see it, so the committed tests
need in-repo data. Mirror the existing `extract_history.py` precedent with
`tools/testdata/nightly_classify/extract_trials.py`, emitting one JSONL per night distilled to the
fields the parser reads plus `duration_ms`. Freeze **07-28, 07-29, 07-30, 07-31, 08-01**. Test-side
helper materialises a manifest into a tmpdir using the real slot-name convention
(`<bench>[_trial2]_ailang_<model>_<ts>.json`) so the real CLI runs against real-shaped input.

| # | Acceptance criterion | Runnable check |
|---|---|---|
| M3.1 | The frozen 08-01 corpus, through the **real CLI**, emits exactly one `INVALID` line and **zero** verdict lines | `--- PASS: test_Fixture_real_0801_incident_is_invalid_and_files_nothing` |
| M3.2 | The frozen 07-28, 07-30 and 07-31 corpora stay **VALID**, emit no `INVALID` line, and still emit ≥1 verdict | `--- PASS: test_Fixture_real_good_nights_stay_valid` |
| M3.3 | 07-29 stays INVALID (positive control retained through the refactor) | `--- PASS: test_Fixture_real_0729_outage_still_invalid` |
| M3.4 | Per-night totals pinned to the measured values below, so a corrupted fixture is loud | `--- PASS: test_Fixture_manifests_match_recorded_totals` |

Pinned values for M3.4 (measured first-party in this worktree):

| night | benches | trials | passes | unmeasured-bench fraction | top failing category (benches) |
|---|---|---|---|---|---|
| 2026-07-28 | 42 | 84 | 54 | 0.048 | `thrash_aborted` 13 |
| 2026-07-29 | 42 | 84 | 14 | 1.000 | `api_error` 42 |
| 2026-07-30 | 42 | 84 | 65 | 0.048 | `thrash_aborted` 8 |
| 2026-07-31 | 42 | 84 | 56 | 0.024 | `thrash_aborted` 13 |
| 2026-08-01 | 12 | 24 | 1 | 1.000 | `non_agentic` 12 |

**Anti-vacuity:** M3.1 **FAILS on main** (today: 5 `SUSPECTED-FLAKE` + 6 `INSUFFICIENT-HISTORY`, no
INVALID). M3.2 **passes on main** — it is the false-positive guard, and it is precisely what fails if
a future planner lowers the threshold or adopts category-agnostic concentration (`thrash_aborted`
= 0.31 on two of those nights). M3.3 passes on main and guards against regressing `#524`.

**Do NOT** append these rows to `replay_2026-07.jsonl`: `ReplayTests` and `TaintTests` iterate that
fixture and pin aggregates over it. New nights go in **new files**.

### M4 — Close the class: cross-language taxonomy-drift guard (~35 test LOC, ~2 CI LOC)

This is the **second** filing of the same gap (`#524` → `#551`): the Go producer's category taxonomy
and the Python consumer's classification sets drift silently. Per `CLAUDE.md` §3, fix the pattern.
Precedent already exists in-file — `test_Vocabulary_every_python_reason_exists_in_validity_go` reads
`validity.go` from Python.

Harvest `ErrorCategory[A-Za-z0-9_]* = "…"` from `internal/eval_harness/metrics.go` and
`error_categorizer.go`; assert every value is classified in **exactly one** of
`RUN_UNMEASURED_CATEGORIES`, a new explicit `MODEL_OUTCOME_CATEGORIES`, or `{"none"}`.

Verified runnable before planning: **16 Go categories harvested, all 16 classify, 0 overlap, exactly
one Python-only extra (`executor_error`)**. Note `executor_error` is a **phantom** — no Go code emits
it anywhere in the repo; it exists only in Python and design docs. Pin it in an explicit
`KNOWN_PYTHON_ONLY = {"executor_error"}` allowlist with that comment; **do not remove it** (behaviour
change with no evidence).

| # | Acceptance criterion | Runnable check |
|---|---|---|
| M4.1 | Every Go `ErrorCategory*` value is classified exactly once; the two sets are disjoint | `--- PASS: test_Taxonomy_every_go_category_is_classified` |
| M4.2 | Known-negative mutation: injecting a synthetic Go constant into the harvested set makes the check fail (mirrors the mutation control in `test_Vocabulary_every_label_is_routed`) | same test |
| M4.3 | Python-only extras equal `{"executor_error"}` exactly, with the phantom documented | `--- PASS: test_Taxonomy_python_only_extras_are_pinned` |
| M4.4 | CI anti-vacuity floor raised from 70 to `<new total> - 4`, matching the four-test rename buffer the comment prescribes | `.github/workflows/ci.yml:148`; `make test-nightly-classifier` PASS count ≥ floor |

**Anti-vacuity:** M4.1/M4.3 **pass on main** (new machinery). M4.2 is the anti-vacuity control and is
mandatory — without it M4.1 would pass under an empty harvest (the classic vacuous-set bug).

**Baseline for M4.4**: 74 tests / 74 PASS lines today, floor 70. The executor must **count**, not
guess, and the CI comment's instruction (*"Raise it whenever you add tests"*) is a hard requirement.

---

## 3. Definition of done

- [x] M1 — split per-benchmark infra taint from run-level unmeasured categories.
- [x] M2 — name the dominant category in INVALID output and shell routing.
- [x] M3 — freeze and replay the five surviving real nightly corpora.
- [x] M4 — enforce cross-language taxonomy classification and raise the CI floor.

- `make test-nightly-classifier` green; PASS count ≥ the new floor; runtime still < 15 s.
- `python3 tools/nightly_classify.py --tonight /tmp/nightly_eval_20260801_rag_on/agent --arm rag_on`
  prints an `INVALID` line naming `[non_agentic]` and **no** verdict lines. *(Opportunistic: `/tmp`
  may be reaped by execution time — M3.1 is the durable form of this check.)*
- The same command on `/tmp/nightly_eval_20260731_rag_on/agent` prints **no** INVALID line.
- `make ci` green. `CHANGELOG.md` updated. `git grep -n "non_agentic" tools/` shows the new set only.
- Commits use `refs #551`; the final commit uses `Fixes #551`.
- **Do not** re-mark 2026-08-01 — containment is already applied (C7).

## 4. Risks

| # | Risk | Mitigation |
|---|---|---|
| R-1 | Executor takes the simple route and adds `non_agentic` to `INFRA_CATEGORIES` (candidate (a)) | M1.4 + M1.5 fail loudly; §0 R-C states why |
| R-2 | A later planner reads `#551` literally and implements category-agnostic concentration (b) | M1.3 + M3.2 fail; §0 R-B records the measured 2-of-3 false-positive rate |
| R-3 | The pass-rate disjunct returns via `#524`'s issue text | Nothing in this sprint touches pass rate; `#524` R-3 still stands |
| R-4 | Appending field 6 breaks shell parsing | `awk` extracts `$2..$5` positionally, so appending is safe; M2.3 pins the legacy line |
| R-5 | Fixtures balloon the repo | ~14 KB across 5 JSONL files vs the 33 KB `replay_2026-07.jsonl` already committed |
| R-6 | `/tmp` reaped before M3 runs | **Highest-urgency item.** Four nights are already gone (R-D). Executor should run M3's extraction **first**, before any code change, and commit the fixtures on their own |

## 5. Explicitly out of scope

- **Why `opencode` stopped running agentically.** Needs a human with the GPU (`opencode` v1.15.7).
  This sprint makes the nightly *report* the outage correctly; it does not fix it. Until it is fixed
  every nightly will be correctly marked INVALID — which is the intended behaviour, not a new bug.
- Renaming `--invalid-infra-fraction` (§1).
- Removing the phantom `executor_error` (M4).
- Any change to Go `validity.go` / `applyValidityBackstop`.

## 6. Acceptance criteria I could NOT make runnable

1. **"The next real nightly is gated correctly."** Only observable at 05:00 the following day.
   Recorded as a post-landing observation for the mission log, not a gate.
2. **Fixtures for 2026-07-24 … 07-27.** Their `/tmp` JSON is already reaped (R-D); the *history*
   rows survive in `replay_2026-07.jsonl` but trial-level directories cannot be reconstructed. The
   good-night false-positive control therefore rests on three nights, not seven.
