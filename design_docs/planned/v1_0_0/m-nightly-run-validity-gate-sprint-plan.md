# Sprint Plan: M-NIGHTLY-RUN-VALIDITY-GATE

**GitHub issue**: [sunholo-data/ailang#524](https://github.com/sunholo-data/ailang/issues/524)
**Design doc**: none — mission-classified as a bug fix inside `tools/nightly_classify.py`, same shape as
[m-nightly-flake-guard](../../implemented/v1_0_0/m-nightly-flake-guard.md) which produced the file.
**Sprint ID**: `M-NIGHTLY-RUN-VALIDITY-GATE`
**Branch**: `sprint/m-nightly-run-validity-gate`
**Worktree**: `/Users/voightkampff/dev/sunholo-data/ailang/.claude/worktrees/sprint-m-nightly-validity`
**Base**: `origin/dev` @ `bf6353703`
**Planner model**: `claude-opus-4-8`
**Planned**: 2026-07-29 (mission-control iteration 119)
**Estimate**: **~1.1 working days (9h)** — up from the mission's ~0.5d. The delta is justified in §6, not absorbed.
**Risk**: medium. No compiler surface, no Go behaviour change, no GPU. But this sprint mutates the
**live** history file the nightly writes at 05:00 daily, and one of its four milestones can silently
*delete* 42 records if implemented naively (§4, R-3).

---

## 0. Premise re-verification (first-party, this worktree, at `bf6353703`)

The controller labelled every handed-down fact. I re-checked all of them and found **one flatly
false**, **one under-specified**, and **one unverified claim that is true and much larger than
suspected**. Per the mission's iteration-112 standard, refutations are stated loudly.

### CONFIRMED

| # | Premise | How verified |
|---|---|---|
| C1 | **Defect 1 is real.** `persistent_failures` (`tools/nightly_classify.py:92-104`) excludes a bench only when `set(cats) - INFRA_CATEGORIES` is empty | Ran the function directly. `[(F,'api_error'),(F,'api_error')]` → suppressed; `[(F,'api_error'),(F,'compile_error')]` → **filed**; `[(F,'timeout'),(F,'logic_error')]` → **filed** |
| C2 | **Defect 2 is real.** No run-level gate exists | Read all 541 lines. `[n for n in dir(nc) if 'valid' in n.lower()]` → `[]`. No record field, no aggregate inspection in `main()` |
| C3 | The measured incident, exactly as tabulated | Recomputed from the live `~/.ailang/state/nightly-eval-history.jsonl` (252 records, read-only). See §1 |
| C4 | The invalid run is already IN the history file, unflagged | 42 records dated `2026-07-29`, schema keys `[arm,bench,cats,class,date,model,passes,trials]` — no validity field |
| C5 | `--type note` never creates a GitHub issue, so "notify once, file nothing" is achievable | `cmd/ailang/messages_send.go:99-103`: `knownGitHubTypes = {bug, feature}`; `syncToGitHub := *github \|\| knownGitHubTypes[category]` |
| C6 | `M-EVAL-MEASUREMENT-CONTRACT` (`970d90e29`) vocabulary is reusable | Read the commit + `internal/eval_harness/validity.go`. Shape `{"valid":bool,"reason":str}`, **absent ⇒ VALID** (load-bearing), reasons `canary_failed / zero_files / zero_pass_all / config_mismatch / harness_error` |

### REFUTED

**R-A. DEFECT 3 IS FALSE. `tools/test_nightly_classify.py` DOES run in CI, and it already has an
anti-vacuity guard.** The controller's greps were too narrow on both sides:

- The root `Makefile` has no python — but it does `include make/test.mk` (line 73), and
  `make/test.mk:19-20` is:
  ```make
  test-nightly-classifier: ## Run nightly variance-guard contract and replay tests
  	@python3 tools/test_nightly_classify.py -v
  ```
- `.github/workflows/ci.yml` never matches `pytest|python3 -m|tools/test_` because it invokes the
  **make target**, at `ci.yml:133-136`, in the `test` job — followed by `ci.yml:138-144`:
  ```yaml
  - name: Assert nightly classifier tests ran (no silent skips)
    run: |
      PASSES=$(grep -c -- '--- PASS:' nightly_classifier.log || true)
      if [ "$PASSES" -lt 20 ]; then
        echo "::error::nightly classifier emitted only $PASSES PASS lines; ..."
  ```
- It is also a dependency of `make ci` (`make/ci.mk:11`).

I ran it: `Ran 40 tests ... OK`, 40 `--- PASS:` lines. **A regression test added by this sprint WILL
execute in CI on every PR.** The sibling sprint's R-D anticipated exactly this trap and fixed it
(M4.6). Defect 3 was a re-derivation of an already-solved problem.

**Consequence for scope**: the planned "make the tests reachable" milestone is **deleted**. What
survives is a genuine, much smaller weakness found while refuting it — see F-1.

### MATERIALLY CORRECTED

**R-B. The "polluted trailing window" claim is TRUE, and I measured it: 12 of 42 benchmarks would
get the wrong verdict tomorrow.** The controller flagged this as unverified and asked me to confirm
the consumer and window before scoping remediation. The consumer is `nightly_classify.py` itself
(`select_window`, `consecutive_failures`), fed by `nightly-eval.sh:357-366`. I replayed the real
history for a hypothetical `2026-07-30` in which each bench fails both trials, with and without the
07-29 records:

| clean verdict → polluted verdict | count | direction |
|---|---|---|
| `REGRESSION` → `SUSPECTED-FLAKE` | 9 | **false negative — a real regression goes unpaged** |
| `REGRESSION` → `GAP` | 1 | **false negative** |
| `SUSPECTED-FLAKE` → `REGRESSION` | 2 | **false positive — a new noise issue gets filed** |
| unchanged | 30 | — |

The false negatives are the dangerous half and they are *silent*. Five of them
(`canonical_normalization`, `immutable_data_structures`, `inline_tests`, `intent_annotated_solver`,
`record_update`/`records_book`) had a **perfectly green 10/10 trailing window** which the outage
drags to 8/10 or 9/10 — i.e. exactly the "was solid, broke today" signal the whole detector exists
to catch is the signal the pollution destroys. Reproduce:

```
python3 - <<'PY'
import sys, json; sys.path.insert(0,'tools'); import nightly_classify as nc
recs=[json.loads(l) for l in open('/Users/voightkampff/.ailang/state/nightly-eval-history.jsonl')]
clean=[r for r in recs if r['date']!='2026-07-29']
for b in sorted({r['bench'] for r in recs}):
    p=nc.classify_bench(b,['compile_error'],recs,'opencode-qwen3-5-35b-a3b-mxfp8','rag_on','2026-07-30')
    c=nc.classify_bench(b,['compile_error'],clean,'opencode-qwen3-5-35b-a3b-mxfp8','rag_on','2026-07-30')
    if p.label!=c.label: print(b,c.label,'->',p.label)
PY
```

**Blast radius, confirmed by reading `select_window`**: it takes the 5 most recent *distinct dates*
strictly before tonight, so the 07-29 night sits in the window for **2026-07-30 … 2026-08-03
inclusive — 5 nights** — then falls out on its own. `consecutive_failures` is **not** windowed: it
walks back until a night with `passes != 0`, so 07-29 also inflates the escalation counter by +1 for
any bench whose failing streak spans it (this is the 2 false positives above). That contamination
ends the first time the bench passes.

**R-C. "~0.5d" is wrong for the work as scoped.** My bottom-up estimate is 9h. See §6 for the
attribution and for the cut line that *does* fit 0.5d.

### Additional first-party findings (not in the controller's set)

**F-1. The CI anti-skip floor is stale and now weak.** `ci.yml:141` fails only below **20** PASS
lines; the suite emits **40**. Half the tests could be deleted or silently skipped and CI would stay
green. This is the same class as the vacuity the guard was written to prevent — the guard just
drifted. Cheap fix, folded into M4.

**F-2. The any-infra fix ALONE would have prevented all four false issues.** Replaying the taint
rule over the real history, per night, over the benchmarks the detector currently files:

| date | filed today | suppressed by any-infra | survivors |
|---|---|---|---|
| 07-24 | 6 | 1 | 5 |
| 07-25 | 7 | 1 | 6 |
| 07-26 | 4 | 0 | 4 |
| 07-27 | 5 | 2 | 3 |
| 07-28 | 8 | 0 | 8 |
| **07-29** | **7** | **7** | **0** |

So M1 and M2 are genuine **defence in depth**, not redundancy — but their *unique* contributions
must be stated honestly: M1 stops the filing, and **only M2/M3 stop the history pollution**, because
`records_for_dir` (`:124-144`) banks every benchmark regardless of what `persistent_failures`
returned. Fixing defect 1 alone leaves all 12 verdict flips in R-B intact.

**F-3. The any-infra fix breaks no existing test, and its one real cost is a one-night delay, not a
lost alarm.** Only one test touches infra categories (`test_all_infra_cats_only_is_ignored`,
`tools/test_nightly_classify.py:119-122`) and it is an all-infra case that still passes. The
`ReplayTests` call `classify_bench` directly with hardcoded `['compile_error']`, so they never route
through `persistent_failures`. **However**: the flake-guard sprint's showcase non-vacuity result —
`csv_to_json_converter` escalating on 07-27 — used a row whose real categories are
`['api_error','compile_error']`, i.e. **infra-tainted**. Under the new rule that bench is suppressed
on 07-27. I checked whether the escalation is lost: it is **not**, it moves to 07-28
(`consec=4`, still `>= K`, no prior `class=regression` banked), because the sibling sprint shipped
`consec >= K` instead of the design doc's literal `consec == K`. That refinement is now load-bearing
for a second reason nobody anticipated. M1's AC pins it.

**F-4. `make check-file-sizes` does not cover this file.** `make/code-health.mk:125` iterates
`find internal cmd -name "*.go"`. `tools/nightly_classify.py` will grow 541 → ~730 lines; that is
*not* a CI gate here, but it is past the "acceptable" band in `coding-standards.md`. Recommendation,
not a requirement: if M3 pushes it past ~750, split the validity logic into
`tools/nightly_validity.py`. Do **not** spend sprint time on a speculative split.

---

## 1. The data that decides the threshold

Recomputed first-party from the live history file (252 records, 6 nights, one model
`opencode-qwen3-5-35b-a3b-mxfp8`, arm `rag_on`, 42 benchmarks × 2 trials per night):

| date | benches | passes/trials | pass rate | **benches with ANY infra trial** | **infra-taint fraction** | benches all-infra |
|---|---|---|---|---|---|---|
| 2026-07-24 | 42 | 52/84 | 0.619 | 2 | **0.048** | 1 |
| 2026-07-25 | 42 | 54/84 | 0.643 | 1 | **0.024** | 0 |
| 2026-07-26 | 42 | 61/84 | 0.726 | 2 | **0.048** | 2 |
| 2026-07-27 | 42 | 65/84 | 0.774 | 2 | **0.048** | 0 |
| 2026-07-28 | 42 | 54/84 | 0.643 | 2 | **0.048** | 2 |
| **2026-07-29** | 42 | **14/84** | **0.167** | **42** | **1.000** | 35 |

The separation is not marginal: **0.048 max baseline vs 1.000**, a factor of 21.

---

## 2. Decision 1 — the predicate and its threshold

### The rule

```
infra_taint_fraction = |{bench : any trial's error_category ∈ INFRA_CATEGORIES}| / |benches parsed|

INVALID  ⟺  benches_parsed == 0                        → reason "zero_files"     (existing vocabulary)
         ∨  infra_taint_fraction >= 0.30                → reason "infra_outage"   (one new constant)
```

Threshold exposed as `--invalid-infra-fraction` (default `0.30`) so tests can drive both sides.

### What it does on all six measured nights — stated explicitly, as required

| date | fraction | verdict | consequence |
|---|---|---|---|
| 2026-07-24 | 0.048 | VALID | detector runs normally (5 issues-worth of signal preserved) |
| 2026-07-25 | 0.024 | VALID | normal |
| 2026-07-26 | 0.048 | VALID | normal |
| 2026-07-27 | 0.048 | VALID | normal |
| 2026-07-28 | 0.048 | VALID | normal |
| **2026-07-29** | **1.000** | **INVALID** | **zero issues filed; one `--type note`; records banked flagged invalid** |

**Zero false positives across the five good nights; fires on the one bad night.** The threshold sits
**6.25×** above the highest observed baseline and **3.3×** below the incident — the widest margin
available. 0.30 is not curve-fitted to a single incident: it is independently defensible as *"nearly
a third of the suite could not be measured"*, and any value in `(0.05, 1.0]` separates this corpus,
so the choice is robust to the exact number.

### DELIBERATE DEPARTURE FROM ISSUE #524 — flagged for the controller

Issue #524 proposes `infra fraction exceeds a threshold` **OR** `the pass total falls outside the
trailing window`. **I am implementing the first disjunct only, and rejecting the second.**

A pass-rate-collapse trigger cannot distinguish *"we failed to measure the subject"* from *"the
subject genuinely broke catastrophically"*. A real compiler regression that breaks 40 of 42
benchmarks produces the same pass-rate signature as this outage — and under the OR rule the detector
would declare the run INVALID and **file nothing on the single most important night in the
project's history.** That is a silent fallback on data that drives decisions, which
`CLAUDE.md` §2 forbids outright.

The infra-taint fraction is a *direct* measurement of unmeasurability (the harness told us it could
not reach the model), so it is safe to gate on. Pass-rate deviation is retained as a **reported
corroborating statistic** in the INVALID notification (`pass rate 0.167 vs trailing median 0.643`)
so a human sees both numbers — but it never triggers the gate.

Controller: if you want the OR rule anyway, say so before M2 — but then the failure mode above must
be written into the issue as an accepted risk.

### Secondary design points

- **Gate order.** The gate runs *before* per-benchmark classification. On INVALID, **no**
  `REGRESSION` / `SUSPECTED-FLAKE` / `GAP` / `INSUFFICIENT-HISTORY` lines are emitted at all.
- **No minimum-suite knob.** A 2-bench run with 1 tainted really is 50% unmeasurable. Adding
  `--min-benches` would be an untested knob guarding a case that has never occurred. The `0` case is
  handled by `zero_files`.
- **Exit code stays 0.** An invalid run is a loud *classification*, not a crash — same posture the
  flake-guard sprint chose for DEGRADED history (M2.4 there). rc≠0 would make launchd log an error
  and change operator behaviour for a condition the tool handled correctly.

---

## 3. Decision 2 — the mixed-trial fix, and what it costs

**Rule**: a benchmark is infra-tainted if **any** trial's category is in `INFRA_CATEGORIES`;
tainted benchmarks are never filed.

```python
def infra_tainted(trials) -> bool:
    return any(cat in INFRA_CATEGORIES for _, cat in trials)
```

**The consequence the controller asked me to weigh**: a benchmark whose only clean signal is one
trial has n=1. **Accepted, and it is already the house position** — `classify_bench`'s
`min_trials=4` across the window exists precisely because the sibling sprint concluded a single
trial cannot certify anything (`test_single_trial_prior_night_cannot_certify_solid`, R-C4 of that
plan: *"`was_solid_in_prev` certifies solid from ONE trial"* was the headline defect). Filing a
GitHub issue off one surviving trial while the other trial proves the infrastructure was broken is
the same error in a new place. Suppression is the consistent choice.

**Measured cost** (F-2 table): ~0.8 suppressed persistent-failures per normal night, 4 across five
nights (13%). **Named cost** (F-3): the flake-guard sprint's own showcase escalation
(`csv_to_json_converter`, 07-27) is delayed by one night to 07-28. Not lost — delayed — and only
because `consec >= K` shipped instead of `consec == K`.

**Existing tests broken: none** (F-3, verified). A *new* test is therefore mandatory or the change is
untested.

---

## 4. Decision 3 — the already-polluted history

**Chosen: adopt the `M-EVAL-MEASUREMENT-CONTRACT` vocabulary verbatim, flag the 42 rows in place,
and filter at read time. Do not delete, do not rewrite the numbers.**

| Option | Verdict |
|---|---|
| Leave it, accept ~5 polluted nights | **Rejected.** R-B measured 12/42 wrong verdicts, 10 of them silent false negatives, on a detector whose entire purpose is catching the solid→broken case. |
| Delete the 42 rows | **Rejected.** Violates `M-EVAL-MEASUREMENT-CONTRACT` D4 (*never delete data — the invalid row is evidence of the bug*). |
| **Validity field + read-time filter + one-shot backfill** | **Chosen.** Same shape, same semantics, same default direction as `970d90e29`. |

### The contract, transcribed to the JSONL schema

- Field: `"validity": {"valid": false, "reason": "infra_outage"}`, written **only** on invalid rows.
- **Absent ⇒ VALID.** This is load-bearing exactly as it is in Go: 210 of the 252 live records
  predate this sprint. Treating absent as invalid erases the entire history from every window.
  (`validity.go:65-75` — *"a plain struct would make absent and explicitly-invalid
  indistinguishable"*.)
- Reasons: reuse `zero_files` as-is; add exactly **one** new constant, `infra_outage`, and add it to
  `internal/eval_harness/validity.go` so there remains **one** vocabulary, not two. The Go const is
  a doc-comment + one line; no Go behaviour changes and no Go test is required — enforcement lives in
  a Python test that reads the `.go` file as text (M4.2), so no Go toolchain is needed to run it.
- Loader split mirrors the Go API exactly:
  `load_history` **filters invalid by default**; `load_history_including_invalid` opts back in.

### THE TRAP THAT MUST NOT BE FALLEN INTO (top risk, R-3)

`update_history` (`:398-424`) rebuilds the whole file from `load_history(path)` and then
`atomic_write_history`. **If `load_history` starts filtering, the very next nightly at 05:00 silently
deletes all 42 invalid rows** — turning a data-quality fix into permanent data loss, and violating
D4. `update_history`, `history_health` and `bootstrap_history` must call
`load_history_including_invalid`. This has a dedicated, named acceptance test (M3.4).

### The backfill

New one-shot subcommand:

```
python3 tools/nightly_classify.py --mark-invalid 2026-07-29 --reason infra_outage \
    --note "total serving failure of opencode-qwen3-5-35b-a3b-mxfp8; 42/42 benchmarks api_error; issues #520-#523 filed in error (all closed)" \
    --history ~/.ailang/state/nightly-eval-history.jsonl
```

It goes through the existing `HistoryLock` + `atomic_write_history` — no new concurrency surface. I
verified the live file is already in `atomic_write_history`'s canonical form (sorted by
`record_key`, `sort_keys=True`, `separators=(',',':')`): re-serialising all 252 records reproduces
the 40,056-byte file **byte-for-byte**. So "the other 210 records are untouched" is a checkable
assertion, not a hope.

**The sprint does not run the backfill against live state.** M3 implements and tests it against a
copy; running it on the rig is a deployment step (§7) with a pre-flight `cp` backup.

---

## 5. Milestones

Four milestones, dependency-ordered, each independently committable. Every acceptance criterion names
an exact command and an exact observable, and every one can fail.

**Mutation hygiene (applies to every "must fail on today's code" criterion below)**: state which
mutation form you ran; assert the edit matched **exactly once** (`grep -c` before/after); print the
diff into the commit message or sprint-JSON `notes`; confirm the mutated file still *imports*
(`python3 -c 'import nightly_classify'`) so a syntax error cannot be mistaken for a caught mutant;
revert and re-run green.

---

### M1 — Per-benchmark infra taint: ANY trial, not ALL (~1h, ~15 impl LOC + ~70 test LOC)

Add `infra_tainted(trials)`; use it in `persistent_failures` in place of `set(cats) - INFRA_CATEGORIES`.
Keep `legacy_classify` calling the same `persistent_failures` (it must not diverge; it is the golden
comparator).

**Acceptance criteria:**

| # | Command | Exact observable |
|---|---|---|
| M1.1 | `python3 tools/test_nightly_classify.py -v -k Taint` | `--- PASS:` for `test_Taint_mixed_infra_and_real_category_is_suppressed` — fixture `[(False,'api_error'),(False,'compile_error')]` yields **no** output line. **This test must FAIL on `bf6353703`**; record the pre-fix failure text in the sprint JSON `notes`. |
| M1.2 | same run | `--- PASS:` for `test_Taint_clean_failure_still_files` — `[(False,'compile_error'),(False,'compile_error')]` still produces exactly one line. A rule that suppresses everything is as broken as no rule. |
| M1.3 | same run | `--- PASS:` for `test_Taint_timeout_and_executor_error_also_taint` — one case per member of `INFRA_CATEGORIES`, each paired with `logic_error`. Prevents an implementation that only special-cases `api_error`. |
| M1.4 | `python3 tools/test_nightly_classify.py -v` | rc=0, **≥ 40** `--- PASS:` lines. No pre-existing test regresses (F-3 predicts none will; if one does, stop and report — the prediction was wrong). |
| M1.5 | Real-corpus replay, asserted in a test using the committed fixture `tools/testdata/nightly_classify/replay_2026-07.jsonl` | `test_Taint_replay_costs_are_pinned`: over 07-24…07-28 the taint rule suppresses exactly **4** of **30** persistent failures (per-night 1,1,0,2,0). Hard-coded numbers so any future widening of `INFRA_CATEGORIES` shows up as a failing test, not as silent extra suppression. |
| M1.6 | same run | `test_Taint_csv_to_json_escalation_moves_to_0728_not_lost` — with 07-27 suppressed and no `class=regression` banked, `csv_to_json_converter` still escalates to `REGRESSION` on 07-28 with `consecutive == 4`. **Pins F-3**: proves `consec >= K` is what saves the alarm, so a future revert to `consec == K` fails loudly here. |
| M1.7 | `bash -n tools/launchd/nightly-eval.sh` | rc=0 (no shell change expected in M1; this is a no-drift check) |

---

### M2 — The run-level validity gate (~3h, ~90 impl LOC + ~40 shell LOC + ~130 test LOC)

Add `run_validity(results, threshold) -> (valid: bool, reason: str, tainted: int, total: int)` and an
`INVALID` stdout line emitted **immediately after `HEALTH`**, before any per-benchmark verdict:

```
INVALID	infra_outage	42/42	0.167	<trailing-median-pass-rate>
```

On INVALID: emit no verdict lines; still compute and bank tonight's records (M3 flags them). Route in
`tools/launchd/nightly-eval.sh` **before** the `REGRESSIONS`/`SUSPECTED`/`GAPS` branches: one
`ailang messages send controlplane --type note` (C5: never reaches GitHub), no Discord ping, log line,
and the summary body leads with the invalid banner.

**Acceptance criteria:**

| # | Command | Exact observable |
|---|---|---|
| M2.1 | `python3 tools/test_nightly_classify.py -v -k Validity` | `--- PASS:` for `test_Validity_six_night_table` — a single table-driven test asserting the §2 verdict for **all six** real nights (0.048/0.024/0.048/0.048/0.048 → VALID; 1.000 → INVALID). This is the threshold-justification test; the numbers are the sprint's evidence. |
| M2.2 | same run | `--- PASS:` for `test_Validity_boundary_is_inclusive` — fraction exactly `0.30` → INVALID; `0.2999` → VALID. Off-by-one on the comparator is the likeliest silent bug. |
| M2.3 | same run | `--- PASS:` for `test_Validity_zero_benches_is_zero_files_not_zerodivision` — empty results dir → `INVALID zero_files`, rc=0, **no traceback on stderr**. |
| M2.4 | End-to-end CLI: `run_cli('--tonight', <07-29-shaped fixture>, '--history', <copy>)` | stdout contains exactly one `INVALID\t` line and **zero** lines beginning `REGRESSION\t`, `SUSPECTED-FLAKE\t`, `GAP\t`, `INSUFFICIENT-HISTORY\t`. rc=0. |
| M2.5 | Same fixture at `bf6353703` behaviour, asserted in the same test | The identical fixture under the *pre-gate* code path emits ≥1 `REGRESSION` line. Assert both, so the test proves a **behaviour change**, not a tautology. (Sibling precedent: `test_single_trial_prior_night_cannot_certify_solid`.) |
| M2.6 | `python3 tools/test_nightly_classify.py -v -k Routing` | `--- PASS:` for `test_Routing_invalid_run_files_nothing`: stubbed `ailang` on `PATH` logging argv; on an INVALID fixture the captured argv contains **exactly one** `messages send`, with `--type note`, and **zero** occurrences of `--type bug`, `--type feature`, `--github`, and **zero** `public-feedback` (Discord) sends. Any `--type bug` fails M2. |
| M2.7 | `grep -nE -- '--type "?bug"?' tools/launchd/nightly-eval.sh` | Every match is inside the `REGRESSION` branch, and that branch is unreachable when `INVALID` is set (assert the guard is an early `if [[ -n "$INVALID" ]]; then ... else`, not a per-branch condition repeated 4×). |
| M2.8 | `bash -n tools/launchd/nightly-eval.sh` and `shellcheck tools/launchd/nightly-eval.sh` | rc=0, no new warnings vs the baseline captured in the commit message. **`shellcheck` is local-only, NOT CI-enforced** (sibling R-E) — do not claim it as CI coverage. |
| M2.9 | Mutation probe | Raise the default threshold to `1.01` in a scratch edit → `make test-nightly-classifier` rc≠0 naming `test_Validity_six_night_table`. Revert. Follow the mutation-hygiene rules above. |

---

### M3 — Validity on history records + the backfill (~3.5h, ~90 impl LOC + ~150 test LOC)

- Write `"validity"` onto tonight's records when the run is INVALID (`records_for_dir` gains the
  marker; valid runs write **no** field, preserving absent⇒valid and keeping the file diff-quiet).
- `load_history` filters invalid **by default**; `load_history_including_invalid` is the escape hatch.
- `update_history` / `history_health` / `bootstrap_history` switch to the including-invalid loader.
- `--mark-invalid DATE --reason R [--note TEXT]` one-shot backfill, through `HistoryLock` +
  `atomic_write_history`.
- `HEALTH` line gains `, N invalid nights excluded`.

**Acceptance criteria:**

| # | Command | Exact observable |
|---|---|---|
| M3.1 | `python3 tools/test_nightly_classify.py -v -k Validity` | `--- PASS:` for `test_Validity_absent_field_means_valid` — a record with no `validity` key survives `load_history`. Assert on a **210-record** slice of the committed replay fixture so the failure mode ("entire history vanishes") is visible as a count, not a boolean. |
| M3.2 | same | `--- PASS:` for `test_Validity_invalid_rows_never_enter_a_window` — the exact R-B experiment as a test: seed real 07-24…07-29 records, flag 07-29 invalid, assert the 2026-07-30 verdicts for **all 42 benches** equal the 07-29-removed verdicts, and assert that **without** the flag ≥10 of them differ. The second half is what makes it non-vacuous. |
| M3.3 | same | `--- PASS:` for `test_Validity_consecutive_failures_skips_invalid_nights` — `consecutive_failures` must also skip invalid rows, or the 2 false-positive escalations in R-B survive the fix. (Easy to miss: it is a separate walker from `select_window`.) |
| M3.4 | **THE DATA-LOSS GUARD**: `-k Validity` | `--- PASS:` for `test_Validity_nightly_update_does_not_delete_invalid_rows` — take a 252-record history with 42 flagged invalid, run a full `--update-history` cycle for a *new* date, assert the file still contains **252 + new** records and all 42 still carry `valid:false`. **This test must FAIL if `update_history` is left calling the filtering loader** — verify by mutation (M3.7). |
| M3.5 | `-k Backfill` | `--- PASS:` for `test_Backfill_marks_only_the_named_date`: on a byte-identical copy of the live 252-record file, `--mark-invalid 2026-07-29` yields exactly 252 records; exactly 42 carry `validity.valid == false` with `reason == "infra_outage"` and the `note`; and the **other 210 records are byte-identical** to their pre-run serialisation (assert via SHA-256 of the sorted re-serialisation of just those 210). |
| M3.6 | `-k Backfill` | `--- PASS:` for `test_Backfill_is_idempotent` (second run → byte-identical file) and `test_Backfill_unknown_date_is_a_loud_error` (rc≠0, no file written, message names the date — never a silent no-op). |
| M3.7 | Mutation probe ×2 | (a) revert `update_history` to the filtering loader → M3.4 fails; (b) make absent-validity decode to invalid → M3.1 fails with a record-count in the message. Both reverted. Record both failure texts in the sprint JSON `notes`. |

---

### M4 — Vocabulary single-sourcing, CI floor, docs (~1.5h, ~10 impl LOC + ~40 test LOC + docs)

The originally-planned "make the tests reachable" milestone is **deleted** (R-A). What replaces it:

- Add `ReasonInfraOutage = "infra_outage"` to `internal/eval_harness/validity.go` with a doc comment
  naming the nightly as the writer. **No other Go change.**
- Raise `ci.yml`'s anti-skip floor from 20 to a value tracking the real suite (F-1).
- `CHANGELOG.md` `[Unreleased]`; migration runbook (§7) recorded in the PR body.

**Acceptance criteria:**

| # | Command | Exact observable |
|---|---|---|
| M4.1 | `git diff internal/eval_harness/validity.go` | Exactly one new const + doc comment. `grep -c 'infra_outage' internal/eval_harness/validity.go` == 1. No function bodies changed. |
| M4.2 | `-k Vocabulary` | `--- PASS:` for `test_Vocabulary_every_python_reason_exists_in_validity_go` — the test **reads `internal/eval_harness/validity.go` as text**, regex-extracts the `Reason* = "..."` literals, and asserts every reason `nightly_classify.py` can write is in that set. Fails loudly if the file is missing (no skip, no `try/except: pass`). **No Go toolchain required** — this is deliberate, so a sandbox without a warm module cache can still run it. |
| M4.3 | `git diff .github/workflows/ci.yml` | The `PASSES -lt 20` floor becomes `-lt 45` (40 existing + ≥5 added). Verify locally first: `make test-nightly-classifier 2>&1 \| grep -c -- '--- PASS:'` must be ≥ 45, else lower the floor to the real number **and say so** — never set a floor the suite cannot meet. |
| M4.4 | `make test-nightly-classifier` | rc=0; final line `OK`; PASS count ≥ 45. |
| M4.5 | `make ci` is **not** required and **not** claimed | Enforcement is `ci.yml:133-144` (already wired, R-A). State this in the PR body so no future reader repeats defect 3. |
| M4.6 | `CHANGELOG.md` `[Unreleased]` | Entry names: the `INVALID` label, the `--invalid-infra-fraction` default 0.30, the any-trial taint rule, the `validity` field with **absent⇒valid**, `--mark-invalid`, and the history path `~/.ailang/state/nightly-eval-history.jsonl`. `git diff CHANGELOG.md` non-empty. |
| M4.7 | Issue #524 | PR body states the **deliberate departure** in §2 (pass-rate disjunct rejected) and the **refutation of defect 3** (R-A), so both survive review rather than being quietly dropped. |

---

## 6. Estimate, and the cut line that fits 0.5d

| Milestone | Impl | Test | Hours |
|---|---|---|---|
| M1 per-bench taint | 15 | 70 | 1.0 |
| M2 run-level gate | 90 + 40 shell | 130 | 3.0 |
| M3 history validity + backfill | 90 | 150 | 3.5 |
| M4 vocabulary + CI floor + docs | 10 | 40 | 1.5 |
| **Total** | **~245** | **~390** | **9.0 (≈1.1d)** |

**Why this is 2.2× the mission's ~0.5d, stated in the open rather than absorbed:** the 0.5d estimate
sized "a bug fix inside `tools/nightly_classify.py`" — which is M1 (1h) plus most of M2. It did not
price **M3 at all**, because the history-remediation half of the item only becomes visible once you
measure R-B and discover that (a) 12/42 verdicts are already wrong for the next 5 nights, and (b) the
naive read-time filter *deletes* 42 records on the next nightly. M3 is 3.5h of the 9h and it is where
the real risk lives.

**If the controller wants a 0.5d box:** ship **M1 + M2 only** (4h). That stops all new false filings
(F-2 proves M1 alone would have prevented #520–#523) and makes every future outage self-flagging.
It leaves the 12 already-broken verdicts in place for 5 nights, expiring 2026-08-03. M3+M4 then
become a follow-up item. **Do not cut M3 partially** — a filtering `load_history` without M3.4's
guard is strictly worse than no filtering at all.

**Cut order inside the full sprint, if it overruns:** (1) M4.3's CI floor bump → separate one-line PR;
(2) M3.6's idempotency test → manual check recorded in the commit message. **Never cut**: M3.4 (the
data-loss guard), M2.5 (the behaviour-change proof), M1.1 (the fails-on-today's-code proof).

---

## 7. Migration / deployment runbook

1. Merge to `dev`; the rig pulls.
2. **Back up first**: `cp ~/.ailang/state/nightly-eval-history.jsonl{,.bak-20260729}`.
3. Run the backfill **once**, manually, **before** the next 05:00 nightly:
   `python3 tools/nightly_classify.py --mark-invalid 2026-07-29 --reason infra_outage --note "..."`.
4. Verify: `wc -l` still `252`; `grep -c infra_outage` == `42`; run the classifier read-only and
   confirm the `HEALTH` line reports `1 invalid night excluded`.
5. First nightly after: confirm the file still has all 42 invalid rows (M3.4's real-world echo) and
   that verdicts match the 07-29-removed baseline.

**Rollback**: `git revert` the M3 commit; the `validity` field becomes inert (an unknown key that
`load_history` ignores), and the `.bak` restores the file. Nothing else reads this JSONL.

---

## 8. Risks (most-likely-to-be-silently-wrong first)

| # | Risk | Why likely | Mitigation |
|---|---|---|---|
| **R-1** | **`update_history` keeps the filtering loader and the next 05:00 nightly deletes 42 records.** A data-quality fix becomes permanent data loss, contradicting D4 | It is a one-word change in a function nobody re-reads, and *every test passes* — the deletion only manifests on the next real run | M3.4 is a dedicated named test **plus** M3.7(a) mutation-proves it fails without the fix. Runbook step 5 checks it in production |
| **R-2** | **The gate is tuned to one incident and fires on a merely-bad night.** A 0.35 fraction on some future night suppresses real signal for 24h | Only 6 nights of data exist; 5 of them are one model | Threshold sits 6.25× above observed max; M2.1 pins all six nights as a table so any retune must restate the evidence. Cost is bounded at one night and is *loud* (a controlplane note), never silent |
| **R-3** | **The pass-rate disjunct gets added back** by a reviewer reading #524 literally, and a real catastrophic regression is silenced | The issue text asks for it | §2 states the rejection and its failure mode; M4.7 requires it in the PR body |
| **R-4** | **The any-infra rule is implemented as `cats[0] in INFRA` or only for `api_error`**, so `timeout`/`executor_error` still leak | Path of least resistance from the issue's `api_error`-heavy evidence | M1.3 has one case per `INFRA_CATEGORIES` member |
| **R-5** | **`test_Validity_invalid_rows_never_enter_a_window` is written only in the positive direction** and passes on today's code | Asserting "verdicts equal the clean baseline" is trivially true if nothing changed | M3.2 requires the negative half too: ≥10 verdicts must differ *without* the flag |
| **R-6** | The Go const is added but nothing enforces the two vocabularies stay in sync | It is an unused exported const; a future rename in Go breaks nothing visibly | M4.2 reads the `.go` file as text from the Python suite that already runs in CI |
| **R-7** | The executor runs the backfill against **live** `~/.ailang/state/` during the sprint and races the 05:00 nightly | The command is right there in the plan | §4 and §7 both state: sprint tests use a **copy**; the live run is a deployment step, after a `cp` backup. `HistoryLock` makes the race safe even so |
| **R-8** | `tools/nightly_classify.py` grows to ~730 lines | +~185 LOC on a 541-line file | Not a CI gate (F-4). Note it; split only if it crosses ~750 |

---

## 9. Gate commands (exact, for the executor)

Run from the worktree root. **None of these bind a socket** — the whole suite is stdlib `unittest`
with `subprocess` child processes and temp dirs, no servers, no ports. The sandbox's loopback-bind
denial is irrelevant here; if any new test needs one, that is a design error — use a temp dir instead.

```bash
# per-milestone (fast, ~2s)
python3 tools/test_nightly_classify.py -v -k Taint        # M1
python3 tools/test_nightly_classify.py -v -k Validity     # M2, M3
python3 tools/test_nightly_classify.py -v -k Routing      # M2.6
python3 tools/test_nightly_classify.py -v -k Backfill     # M3.5, M3.6
python3 tools/test_nightly_classify.py -v -k Vocabulary   # M4.2

# the sprint gate — this is what CI runs
make test-nightly-classifier
#   PASS: final line is "OK", rc=0, and:
make test-nightly-classifier 2>&1 | grep -c -- '--- PASS:'    # must be >= 45

# shell (local only, NOT CI-enforced)
bash -n tools/launchd/nightly-eval.sh
shellcheck tools/launchd/nightly-eval.sh
```

**`make test` (Go) is NOT a gate for this sprint.** The only Go edit is a single new string constant
with no callers; it cannot change behaviour. If the executor wants reassurance, `gofmt -l
internal/eval_harness/validity.go` (must print nothing) is sufficient and needs no build. **Do not
run `make test`**: a sandboxed Go test failure here would be uninformative and would cost the
controller a re-run to disprove.

**`rig.lock` is NOT acquired.** No GPU, no eval, no model.

---

## 10. Deliverables

- `tools/nightly_classify.py` (modified: `infra_tainted`, `run_validity`, `INVALID` line, validity
  read/write split, `--mark-invalid`; 541 → ~730 lines)
- `tools/test_nightly_classify.py` (modified: +~390 test LOC, 40 → ~48 tests)
- `tools/launchd/nightly-eval.sh` (modified: early INVALID branch + summary banner, ~+40 lines)
- `internal/eval_harness/validity.go` (modified: **one** new const + doc comment)
- `.github/workflows/ci.yml` (modified: anti-skip floor 20 → 45)
- `CHANGELOG.md` (`[Unreleased]`)
- **No compiler surface. No new Go logic. `make check-boundaries` unaffected.**

> **Executor gotcha**: `.gitignore:77` ignores `.ailang/`, yet every sibling sprint JSON is tracked
> (force-added by convention). `git add .` will **silently skip** the sprint JSON. Use
> `git add -f .ailang/state/sprints/sprint_M-NIGHTLY-RUN-VALIDITY-GATE.json`.

---

**SPRINT_PLAN_PATH**: `design_docs/planned/v1_0_0/m-nightly-run-validity-gate-sprint-plan.md`
**SPRINT_JSON_PATH**: `.ailang/state/sprints/sprint_M-NIGHTLY-RUN-VALIDITY-GATE.json`
