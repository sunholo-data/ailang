# Sprint Plan: M-NIGHTLY-SUSTAINED-FAILURE-LABEL

**GitHub issue**: [sunholo-data/ailang#538](https://github.com/sunholo-data/ailang/issues/538)
(siblings: #537 closed triage verdict, #540 honest capability-gap filing)
**Design doc**: none — mission-classified as a bug fix inside `tools/nightly_classify.py` +
`tools/launchd/nightly-eval.sh`, the same shape and size as
[m-nightly-run-validity-gate](m-nightly-run-validity-gate-sprint-plan.md) (#524, iteration 119) and
[m-nightly-flake-guard](../../implemented/v1_0_0/m-nightly-flake-guard.md) (iteration 113), both of
which shipped doc-less. Confirmed appropriate.
**Sprint ID**: `M-NIGHTLY-SUSTAINED-FAILURE-LABEL`
**Branch**: `sprint/m-nightly-sustained-failure-label`
**Worktree**: `/tmp/wt-iter122-nightly-label`
**Base**: `origin/dev` @ `7ce83f299`
**Planner model**: `claude-opus-4-8`
**Planned**: 2026-07-30 (mission-control iteration 122)
**Estimate**: **~0.9 working days (7.5h)** — inside the mission's 0.5–1d box.
**Risk**: low-medium. No compiler surface, no Go change, no GPU, no live-state mutation (unlike
#524, this sprint writes nothing to `~/.ailang/state/`). The one real hazard is a *silent* one and
it is measured, not theorised: a naive version of this fix pages a chronic benchmark **every night
forever** (§2, R-1) or makes it **vanish entirely** (§2, R-2). Both are proven below and both are
gated by named tests.

---

## 0. Premise audit (first-party, this worktree, at `7ce83f299`)

The controller labelled every fact and asked for refutations. **All five VERIFIED premises are
UPHELD, reproduced exactly.** The refutations are in the *reasoning around* them: one framing in
issue #538 is wrong, the controller's preferred body wording is wrong for a case it did not
consider, and one of the two mechanisms the controller assumed guards re-paging is not the one
doing the work in the live instance.

### UPHELD

| # | Premise | Settled by |
|---|---|---|
| P1 | The incoherent boundary, exact table | `uv run --no-project python` probe importing `tools/nightly_classify.py`; output: `0/10 → GAP`, `1/10 → REGRESSION esc_from=SUSPECTED-FLAKE`, `5/10 → SUSPECTED-FLAKE`, `10/10 → REGRESSION esc_from=-`. Byte-for-byte the controller's table. |
| P2 | The code | `tools/nightly_classify.py:328-357` (`classify_bench`, ladder + K=3 backstop at :347-354), `:291-315` (`consecutive_failures`, `already_regressed` at **:314** = `str(record.get("class","")).lower() == "regression"`) |
| P3 | Only `REGRESSION` files a bug | `grep -n '--type' tools/launchd/nightly-eval.sh` → **exactly 4 hits**: `:528` `bug` (REGRESSION branch), `:506`/`:555`/`:569` `note` (INVALID / SUSPECTED-FLAKE / GAP). INSUFFICIENT-HISTORY has no branch of its own — it is folded into the final summary at `:578-586`. INVALID short-circuits via the outer `else` at `:505-511`. |
| P4 | The shell already knows and still lies | `:513-517` sets `RULE="escalated: 3rd consecutive all-fail night (from ${ESCALATED})"` vs `RULE="solid trailing window"`; **both** then hit `:518-528` — same `--title "Nightly regression: ${BENCH} (${DATE})"`, same `--type "bug"`, same body line *"Investigate this solid-window break or sustained-failure escalation."* `ESCALATED` is `Verdict.tsv()` field **7** (`:38-49`), read as the 6th awk output field at `:492`. |
| P5 | CI wiring is real, floor is 55, baseline is 60 | `make test-nightly-classifier` → **rc=0, exactly 60 `--- PASS:` lines**, `REPLAY SUMMARY: filed=5 guarded_regressions=2 suppressed=4`. `make/test.mk:19-20` → `python3 tools/test_nightly_classify.py -v`. `.github/workflows/ci.yml:136-151`: runs the target, then fails below **55** PASS lines. Also a `make ci` dependency (`make/ci.mk:11`). |
| P6 | The live instance | `~/.ailang/state/nightly-eval-history.jsonl` (294 rows, read-only): `config_file_parser` / `opencode-qwen3-5-35b-a3b-mxfp8` / `rag_on` = `07-24 0/2, 07-25 0/2, 07-26 1/2, 07-27 0/2, 07-28 0/2, 07-29 0/2 (validity.valid=false, infra_outage), 07-30 0/2`. **1 pass in 14 trials.** Exactly as stated. |
| P7 | Open-design-question part (b) is real **and consequential** | Not just theoretically. Simulated the naive rename (escalation emits `SUSTAINED-FAILURE`, banked as `class="sustained-failure"`, guard unchanged) over a stream whose lucky pass stays inside the 5-night window. **Today: 1 page. Naive rename: 3 consecutive pages.** Full output in §2 R-1. |
| P8 | `tools/nightly_classify.py` is 689 lines | `wc -l` → 689. Headroom to the 800 target: 111. Fix adds ~40. **Also: `make check-file-sizes` does not cover it** (`make/code-health.mk:125` iterates `find internal cmd -name "*.go"`) — same as #524's F-4. Not a gate; still stays under. |

### REFUTED / MATERIALLY CORRECTED — loudly

**R-A. Issue #538's headline framing is wrong, and the fix must not "remove the asymmetry".**
#538 says: *"a strictly worse benchmark gets the quieter label"* and calls the boundary
*"incoherent"*. The **ordering is deliberate, documented, and defensible** — it is not ordered by
badness, it is ordered by **evidence of achievability**.
`design_docs/implemented/v1_0_0/m-nightly-flake-guard.md:263-270`:

> …only a `p̂ == 0` window (GAP, never-passed/already-failing) is excluded — **that is gap-finder
> territory**, same as today. This makes the worst case provable: **no benchmark that has ever
> passed goes unpaged past its 3rd consecutive failing night.**

A 0/10 benchmark has **never been observed green on this model**, so there is no evidence it is
achievable at all → that is a benchmark-design / capability question, and it *does* get a channel
(the GAP `--type note` at `:557-570`, explicitly "Gap-finder candidates"). A 1/10 benchmark **has**
been observed green, so a sustained all-fail run is worth a triage slot. Ordering by achievability
evidence is coherent; #538 conflates it with ordering by badness.

**Consequence for scope: `GAP` stays exempt from escalation, and the sprint adds a test that
LOCKS that exemption** (M1.6) so a future reader who re-reads #538's framing cannot silently widen
it. The real defect in #538 is narrower than its own title: **the escalated verdict asserts
something false**, nothing more. That narrowing is what keeps this at 0.9d.

*(This also independently vindicates the controller's rejection of the pass-rate floor — see R-D.)*

**R-B. The controller's proposed body wording — *"never green in the window"* — is FALSE in a case
neither the controller nor #538 considered, i.e. it commits the same class of error it is fixing.**
Escalation is **label-agnostic** across `SUSPECTED-FLAKE` **and `INSUFFICIENT-HISTORY`**
(`:349`, `label not in {"REGRESSION","GAP"}`; flake-guard doc :265-267 revised it that way in
quorum round 1 deliberately). A benchmark with **one** prior night at 2/2 is `INSUFFICIENT-HISTORY`
(`min_nights=2`), and if it then all-fails 3 nights it escalates — **that benchmark WAS green.**
`tools/test_nightly_classify.py:733-747` (`test_new_benchmark_timeline`) is exactly this shape.

The wording must be the statement that is true in **every** escalation case:

> failed all trials for N consecutive nights; the prior window is **not solid** (X/Y passes), so
> this is **not** a certified fresh break — triage as sustained failure / capability gap.

No "never green". No "solid-window break". M2 pins this with a body-text test.

**R-C. In the LIVE instance, `already_regressed` is not what stops the re-page — the window drain
is. The controller's (b) is still real, but for a different stream shape.** Replaying the
1-pass-then-dead shape forward with the class banked each night:

```
07-26: REGRESSION   esc=SUSPECTED-FLAKE   ← escalates
07-27: GAP  07-28: GAP  07-29: GAP  07-30: GAP  07-31: GAP  ← lucky pass fell out of W=5
```

The label becomes `GAP` (exempt) before `already_regressed` is ever consulted. Confirmed live:
`classify_bench(config_file_parser, …, tonight="2026-07-31")` → `SUSPECTED-FLAKE`, consec=4, **not**
paging, because `07-30`'s banked `class="regression"` suppresses it. So **both** mechanisms exist and
**both** are load-bearing on different stream shapes (flake-guard doc :264-265 names both:
"nights 4+ do not re-fire — the window then drains toward GAP"). A fix that only tests the
window-drain shape would pass while breaking the guard. M1.2 tests the *guard* shape specifically,
because that is the one the naive rename breaks (R-1).

**R-D. The controller is right to reject the pass-rate floor, and I can give the anchor it was
missing.** Not merely "it silences chronic failures": the pass-rate floor **contradicts a written,
shipped invariant** — flake-guard D2's *"no benchmark that has ever passed goes unpaged past its
3rd consecutive failing night"*. A rate floor deletes that guarantee outright. And #524's
`decision_departures` rejected a pass-rate disjunct for the adjacent reason (a rate signal cannot
distinguish "we failed to measure" / "it genuinely broke"). **Rejection UPHELD with a stronger
reason than the one given.**

### NEW FINDINGS (not in the handoff)

**F-1 (the sharpest one — an unrecognised label is SILENTLY DROPPED, not mis-filed).** I extracted
the routing block from `nightly-eval.sh` (the same harness `RoutingContractTests` uses), stubbed
`ailang`, and fed it one `SUSTAINED-FAILURE` line. Result: **exactly one `messages send` — the
end-of-run summary — with `Regressions: none| Suspected flakes: none| Non-regression failures: none`.
The benchmark disappears completely.** Every extractor at `:489-494` is an exact-match awk
(`$1=="REGRESSION"` …); there is no catch-all and no `else`. So a **classifier-only** fix would
make chronic failures *invisible* — precisely the outcome the K=3 backstop was added to prevent.
This is the decisive argument for §3's answer and the reason M2 is not optional.

**F-2 (iteration 113's own showcase example is an instance of this bug).** Property-scanned the
committed fixture `tools/testdata/nightly_classify/replay_2026-07.jsonl` (211 records) for
`label == "REGRESSION" and passes != trials`: **2 violations, both `csv_to_json_converter`** —
`2026-07-27` at **1/6** and `2026-07-28` at **1/8**. The 07-27 one is the flake-guard sprint's
headline non-vacuity result (`test_replay_csv_to_json_escalates_once`, :1032-1043) and #524's F-3
follow-up (`test_Taint_csv_to_json_escalation_moves_to_0728_not_lost`, :296-313). So the invariant
test in M3.1 is **non-vacuous against committed data**, not just against a synthetic fixture.

**F-3 (blast radius is exactly 7 existing assertions).** Tests that assert the escalated verdict is
`REGRESSION` and must be updated (not deleted — their *shape* is still the contract):
`:312` `test_Taint_csv_to_json_escalation_moves_to_0728_not_lost`,
`:745` `test_new_benchmark_timeline`,
`:758-759` `test_Escalation_fires_exactly_once`,
`:771` `test_Escalation_missed_third_night_fires_on_fourth`,
`:1035`+`:1043` `test_replay_csv_to_json_escalates_once`,
`:1054-1057` `test_Replay_aggregate_is_five_today_vs_two_guarded`,
plus `:1146-1148` `test_routing_smoke_aggregates_suppressed_labels` (`--type bug` / `--type note`
counts). Per `.claude/rules/coding-standards.md`, out-of-date expectations get rewritten, no
back-compat shim.

**F-4 (pre-existing cosmetic bug found in passing, NOT in scope).** `:578` runs
`awk` over `$INSUFFICIENT` unconditionally, so when there are no insufficient-history benchmarks
every nightly summary still prints a garbage line:
`insufficient history:  ( over  nights, failing /3 toward escalation)\n`. Reproduced in F-1's
harness. Two-line `[[ -n ]]` guard. **Deliberately left out** — see §6.

**F-5.** `record["class"] = verdict.label.lower()` is at **`:674`**, inside
`if args.update_history and not unavailable`. That single line is why the classifier side must own
the label: it is what puts the assertion into the durable trend input.

**F-6.** No consumer outside these three files. `grep -rln 'SUSPECTED-FLAKE|suspected-flake|nightly-eval-history'` (excluding `.git`, `node_modules`, `__pycache__`) → `tools/nightly_classify.py`, `tools/launchd/nightly-eval.sh`, `tools/test_nightly_classify.py`, plus docs/changelog/sprint-JSON. **No Go, no dashboard, no UI.** `legacy_classify` (`:360-371`) emits only REGRESSION/GAP and is untouched (it is the golden comparator).

---

## 1. The defect, stated precisely

`classify_bench` (`:347-354`) promotes a non-solid window to the literal string `REGRESSION`:

```python
if (label not in {"REGRESSION", "GAP"}
        and consecutive >= escalate_after
        and not already_regressed):
    escalated_from = label
    label = "REGRESSION"
```

`REGRESSION` is the only label that reaches `--type bug` (P3), and `--type bug` is the only thing
that creates a GitHub issue (`cmd/ailang/messages_send.go` `knownGitHubTypes = {bug, feature}`, per
#524's C5). So the escalation produces a GitHub issue titled **"Nightly regression: `<bench>`"**
whose body says *"Investigate this solid-window break"* — while the same body prints
`prior window 1/10 over 5 nights`, contradicting itself.

**The invariant being violated:** *no verdict may be labelled `REGRESSION` unless the trailing
window is solid (`passes == trials`).* Violated twice in the committed fixture (F-2).

---

## 2. Top risks (both measured, both gated)

**R-1 — THE RE-PAGE TRAP.** Renaming the escalated label without teaching
`consecutive_failures` about the new banked class turns "pages once" into "pages every night".
Measured (stream: solid 07-24, 1/2 on 07-25, all-fail thereafter — lucky pass stays inside W=5 at
the escalation boundary):

| night | today | naive rename |
|---|---|---|
| 07-26 | SUSPECTED-FLAKE | SUSPECTED-FLAKE |
| 07-27 | SUSPECTED-FLAKE | SUSPECTED-FLAKE |
| 07-28 | **REGRESSION** ← pages | **SUSTAINED-FAILURE** ← pages |
| 07-29 | SUSPECTED-FLAKE (suppressed) | **SUSTAINED-FAILURE** ← **pages again** |
| 07-30 | SUSPECTED-FLAKE (suppressed) | **SUSTAINED-FAILURE** ← **pages again** |
| 07-31 | GAP | GAP |

**3 pages instead of 1.** Gated by M1.2. Every existing test passes under the naive rename except
the ones F-3 lists, so this must be a *named new* test, not a hoped-for regression catch.

**R-2 — THE VANISHING TRAP.** F-1: a classifier-only fix makes the benchmark emit no message at
all. Gated by M2.1 **and** systemically by M3.2 (label-vocabulary lockstep).

**R-3 — SUPPRESSING EVERYTHING.** A fix that quiets the genuine solid-window break is a regression,
not a fix. Gated by M1.4 (synthetic 10/10 → break, same night) and M1.5 (the real
`higher_order_functions` 07-26 control from #483 / iteration 113) — **neither may change**.

**R-4 — MIGRATION.** Live history already contains `class="regression"` rows written by
*escalations* (`config_file_parser` 07-29 and 07-30, verified in P6). If the guard only recognises
the new string, those rows stop suppressing and the live instance re-pages tomorrow. The guard must
accept **both**. Gated by M1.3.

---

## 3. Decision on the OPEN DESIGN QUESTION

**BOTH sides own the fix, in ONE PR. The classifier owns the label; the shell owns the message; the
suppression guard is widened to a set of paging classes and becomes an explicitly tested
invariant.**

*Why not shell-only (the controller's leaning):* `:674` banks `verdict.label.lower()` into
`~/.ailang/state/nightly-eval-history.jsonl`, which is the **sole input to every future window and
escalation count**. A shell-only fix leaves the machine-readable contract asserting `regression`
for a benchmark that was never solid — the exact "polluted trend input" failure #524 spent a
milestone remediating. CLAUDE.md §2 forbids data whose value is knowingly wrong.

*Why not classifier-only:* **F-1, measured.** The shell drops unrecognised labels silently. Chronic
failures would go from mislabelled-but-visible to invisible, defeating iteration 113's backstop.

*The suppression invariant, stated so it can be tested:*

```python
PAGING_CLASSES = {"regression", "sustained-failure"}   # "regression" retained for pre-fix rows
```

`consecutive_failures` returns `already_paged` (renamed from `already_regressed`) = any valid night
in the current all-fail run whose banked `class` is in `PAGING_CLASSES`. **A benchmark pages at most
once per all-fail run**, for both label kinds and across the pre/post-fix boundary.

*Label vocabulary after the fix (5 labels, unchanged count — `REGRESSION` narrows, one is added):*

| label | means | channel | Discord |
|---|---|---|---|
| `REGRESSION` | window solid (`passes == trials`) → all-fail tonight | `--type bug` | 1 ping |
| **`SUSTAINED-FAILURE`** *(new)* | K=3 all-fail run, window **not** solid | `--type bug` | **no ping** |
| `SUSPECTED-FLAKE` | all-fail + mixed window, not yet escalated | `--type note` | no |
| `GAP` | window all-fail (never green) — **still exempt from escalation** (R-A) | `--type note` | no |
| `INSUFFICIENT-HISTORY` | < 2 nights / < 4 trials | summary line | no |

**No Discord ping for `SUSTAINED-FAILURE`** — deliberate and tested (M2.3). The ping's text is
*"benchmarks that previously passed now fail"*; a sustained failure is the *absence* of a change.
The ping means "something broke tonight"; the bug means "this deserves a triage slot". Keeping them
separate is the whole point of the sprint. Side benefit: once escalations leave the REGRESSION
branch, that ping's wording becomes true again for the first time.

**Where `already_regressed` was, `escalated_from` stays**: `SUSTAINED-FAILURE` carries
`escalated_from ∈ {SUSPECTED-FLAKE, INSUFFICIENT-HISTORY}` in TSV field 7, and `REGRESSION` now
*always* carries `-` there — so the dead `if [[ "$ESCALATED" != "-" ]]` branch at `:515-517` is
removed rather than left to rot.

---

## 4. Milestones

### M1 — Classifier: the `SUSTAINED-FAILURE` terminal label + the paging-suppression invariant (2.5h, ~150 LOC)

`tools/nightly_classify.py`: add `PAGING_CLASSES`; `consecutive_failures` (`:291-315`) returns
`already_paged` computed against that set (accepting the legacy `"regression"` string);
`classify_bench` (`:347-354`) sets `label = "SUSTAINED-FAILURE"` on escalation. `escalate_after`,
the window rule, the `GAP` exemption and `consec >= K` (not `== K`) all unchanged.

Acceptance criteria:
- **M1.1 (MUST FAIL AT `7ce83f299`)** `test_Sustained_escalation_is_not_labelled_regression`: 5 prior
  nights ×2 trials with one pass, tonight all-fail → label `SUSTAINED-FAILURE`,
  `escalated_from == "SUSPECTED-FLAKE"`, `passes/trials == 1/10`. Record the pre-fix failure text
  (`AssertionError: 'REGRESSION' != 'SUSTAINED-FAILURE'`) in the JSON notes.
- **M1.2 (MUST FAIL under the naive rename — R-1)**
  `test_Sustained_pages_at_most_once_per_run_when_pass_stays_in_window`: the §2 R-1 stream, 6 nights
  forward, banking `class = label.lower()` each night → the sequence contains **exactly one**
  `SUSTAINED-FAILURE`. Prove non-vacuity by reverting *only* the `PAGING_CLASSES` line and showing
  the count is 3.
- **M1.3 (MIGRATION — R-4)** `test_Sustained_legacy_regression_class_still_suppresses`: a prior
  night banked with the **old** `class="regression"` (as the live 07-30 row is) suppresses tonight's
  escalation. Then the live shape: `classify_bench("config_file_parser", …, tonight="2026-07-31")`
  against a copy of the real 294-row history → **not** a paging label.
- **M1.4 (POSITIVE CONTROL — R-3)** `test_Rule_solid_window_regresses` and the synthetic
  5×(2/2)→all-fail case still yield `REGRESSION`, same night, `escalated_from == ""`.
- **M1.5 (REAL POSITIVE CONTROL — R-3)** `test_Replay_real_genuine_regression_pages_same_night`
  unchanged: `higher_order_functions` @ `2026-07-26` → `("REGRESSION", 4, 4)`.
- **M1.6 (LOCKS R-A)** `test_Rule_never_passed_bench_does_not_escalate` extended: 0/10 with
  `consecutive >= 3` → `GAP`, and assert the label is **not** `SUSTAINED-FAILURE`. Comment cites
  flake-guard D2 :263-270 so the next reader does not "fix" #538's framing.
- **M1.7** F-3's 7 assertions updated; `make test-nightly-classifier` rc=0 with **no** unexpected
  failures. If a test outside F-3's list moves, STOP and report — the blast-radius map was wrong.
- **M1.8** `uv run --no-project python -c "import tools.nightly_classify"`-equivalent import check
  passes after every mutation experiment (a syntax error must not be mistaken for a caught mutant).

### M2 — Shell: route the new label, and stop asserting a break (2.5h, ~130 LOC)

`tools/launchd/nightly-eval.sh`: new `SUSTAINED` extractor beside `:489-494` (fields 2–7); a new
branch after the REGRESSION branch using `--type "bug"`, title
`Nightly sustained failure: ${BENCH} (${DATE})`, and R-B's wording; **no** `public-feedback` send.
Delete the dead `ESCALATED` branch at `:515-517` and rewrite the REGRESSION body to say
solid-window break only. Add `Sustained failures: ${SUSTAINED_NAMES}|` to the summary at `:582`.

Acceptance criteria:
- **M2.1 (MUST FAIL AT `7ce83f299` — F-1/R-2)** `test_Routing_sustained_failure_files_one_bug`: feed
  the extracted routing block one `SUSTAINED-FAILURE` line → exactly **1** `--type bug`, and the
  benchmark name appears in the summary. At HEAD this asserts 1 and gets 0.
- **M2.2 (BODY HONESTY — R-B)** `test_Routing_sustained_body_claims_no_break`: the filed body
  contains the window fraction, the consecutive count and `escalated_from`, and contains **none** of
  `solid-window break`, `regression`, `never green`. Title matches
  `^Nightly sustained failure: .* \(\d{4}-\d{2}-\d{2}\)$` (preserving the dated-title dedup contract
  flake-guard D1.2/V25 relies on).
- **M2.3 (NO DISCORD)** same harness: `messages send public-feedback` count **0** for a
  sustained-failure-only night, and still **1** for a REGRESSION-only night.
- **M2.4** `test_routing_smoke_aggregates_suppressed_labels` extended to all **5** labels in one
  `$CLASSIFIED` → `--type bug` == 2, `--type note` == 2, `public-feedback` == 1, and the summary
  names each bench.
- **M2.5** `bash -n tools/launchd/nightly-eval.sh` rc=0; `shellcheck` locally clean for the edited
  region (not CI-enforced).
- **M2.6** `test_Routing_invalid_run_files_nothing` unchanged — INVALID still short-circuits
  everything (#524's guarantee must not be reopened).

### M3 — Systemic: the label-vocabulary lockstep guard + the REGRESSION invariant (1.5h, ~70 LOC)

The reason this bug existed is that **the emitter and the router had no shared vocabulary**. Fix the
class, not just the case (CLAUDE.md §3).

Acceptance criteria:
- **M3.1 (NON-VACUOUS ON COMMITTED DATA — F-2)** `test_Invariant_regression_implies_solid_window`:
  sweep every `(bench, model, "rag_on", date)` in `tools/testdata/nightly_classify/replay_2026-07.jsonl`
  and assert `label == "REGRESSION" ⇒ passes == trials`. At HEAD this fails with **2** violations
  (`csv_to_json_converter` 07-27 @1/6, 07-28 @1/8); after the fix, 0. Print the violation list on
  failure.
- **M3.2 (PREVENTS THE NEXT SILENT DROP — F-1)** `test_Vocabulary_every_label_is_routed`: derive the
  emittable label set from `tools/nightly_classify.py` source and the routed set from
  `tools/launchd/nightly-eval.sh` (`$1=="…"` awk patterns), assert **equality**. Prove non-vacuity by
  deleting one awk line in a temp copy and showing the test fails. Mirrors the existing
  `VocabularyTests` python↔Go pattern at `:672-690`.
- **M3.3** `REPLAY SUMMARY` line extended to
  `filed=5 guarded_regressions=N sustained=M suppressed=K` so the aggregate is **shown moving** from
  the current `filed=5 guarded_regressions=2 suppressed=4`. The expected post-fix numbers must be
  asserted, not merely printed — and the count of benchmarks that reach `--type bug` must be
  **unchanged** (`2`), because this sprint re-labels, it does not re-scope.

### M4 — CI floor, changelog, mission record (1h, ~65 LOC)

Acceptance criteria:
- **M4.1** Measure `make test-nightly-classifier 2>&1 | grep -c -- '--- PASS:'`. Require **≥ 71**
  (60 baseline + ≥ 11 new). Raise `.github/workflows/ci.yml:151` from `55` to **`measured - 4`**, and
  update the comment above it with the measured number and the date — the drift #524's F-1 caught
  happened because the comment went stale. State the new floor in the PR body.
- **M4.2** `make test-nightly-classifier` rc=0; final line `OK`.
- **M4.3** `CHANGELOG.md` entry under Fixed: the label split, the paging-once invariant, the
  no-Discord decision, `refs #538`, and one line noting #537 was a false alarm of this class.
- **M4.4** `wc -l tools/nightly_classify.py` < 800 (expected ~730). Record the number.
- **M4.5** This plan's §3 label table appended as an amendment note to
  `design_docs/implemented/v1_0_0/m-nightly-flake-guard.md`'s D4 table region — **a dated
  "amended by #538" footnote only**, not a rewrite of the implemented doc.
- **M4.6** Final commit message uses `Fixes #538`.

---

## 5. Gates

```bash
make test-nightly-classifier                      # THE sprint gate: rc=0, final line OK
make test-nightly-classifier 2>&1 | grep -c -- '--- PASS:'   # must be >= 71
python3 tools/test_nightly_classify.py -v -k Escalation      # per-milestone, ~2s
python3 tools/test_nightly_classify.py -v -k Routing
python3 tools/test_nightly_classify.py -v -k Vocabulary
bash -n tools/launchd/nightly-eval.sh
wc -l tools/nightly_classify.py                   # < 800
```

**No sockets anywhere.** The suite is stdlib `unittest` + `subprocess` + temp dirs; the sandbox's
loopback-bind denial is irrelevant. If a new test appears to need a socket, that is a design error —
use a temp dir. **`make test` (Go) is NOT a gate**: this sprint touches no Go. **`rig.lock` is NOT
acquired**; no GPU, no eval run, no write to `~/.ailang/state/` (the live history is read-only input
to M1.3, always via a temp copy).

**Committing the sprint JSON**: `.ailang/` is gitignored (`.gitignore:77`) but every sibling sprint
JSON is tracked, so the progress file needs
`git add -f .ailang/state/sprints/sprint_M-NIGHTLY-SUSTAINED-FAILURE-LABEL.json`.

**Mutation hygiene** (per #524's standard): for every "must fail before" criterion, state the
mutation form, assert the edit matched exactly once (`grep -c` before/after), paste the pre-fix
failure text into the JSON notes, confirm the mutated file still imports, then revert and re-run
green.

---

## 6. Explicitly OUT of scope

1. **Widening `GAP` to escalate** — R-A. Measured cost of *not* doing it: on the live 294-row
   history, replaying a hypothetical all-fail 07-31 for all 42 streams yields **0 GAP streams with
   `consecutive >= 3`**, so widening would page nothing today anyway; and it would delete
   flake-guard D2's written guarantee's rationale. M1.6 locks the exemption instead.
2. **The pass-rate floor** — rejected, R-D, with a stronger reason than the handoff's.
3. **F-4's empty-`INSUFFICIENT` garbage summary line.** Real (every nightly prints it), two lines to
   fix, in a file this sprint edits — but it is a *different* bug in a *different* branch with no
   relation to #538's assertion defect. Bundling it dilutes the PR's single claim. **File it as its
   own issue during M4** and let the mission route it; do not fix it here.
4. Any change to `INFRA_CATEGORIES`, `escalate_after`, `window_nights`, `min_nights`, `min_trials`.
5. Any change to the eval harness, benchmarks, prompts, or the AILANG compiler. #540 owns the actual
   `config_file_parser` capability gap; this sprint owns only what the detector *says*.
6. Multi-model / `rag_off` history. Backfilling `class` on existing rows (the guard reads both
   strings — no migration needed, R-4/M1.3).
7. Closing #537 (already closed) or #540 (a real gap, stays open).

## 7. Cut order if overrunning

1. **M4.5** (the flake-guard doc footnote) → fold into the PR body.
2. **M3.3** (replay-summary arithmetic) → keep the print, drop the added assertion.
3. **NEVER CUT**: M1.2 (the re-page invariant — the one thing that silently rots), M2.1 (the
   vanishing trap), M1.4+M1.5 (the positive controls — without them this PR cannot be distinguished
   from "suppress everything"), M3.2 (the lockstep guard — the systemic fix), M4.1 (the floor; a
   floor that is not raised makes every test above vacuous).
4. **If a 0.5d box is forced**: M1+M2 only (5h). That fixes the falsehood and routes the label
   correctly. Do **not** ship M1 without M2 — F-1 proves that combination makes chronic failures
   disappear, which is strictly worse than today.
