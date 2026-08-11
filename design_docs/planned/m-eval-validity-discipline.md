# M-EVAL-VALIDITY-DISCIPLINE: like-for-like, coverage-gated eval comparisons everywhere

**Status**: IN PROGRESS — coverage gating + per-model coverage landed 2026-07-11 (ratings block + ELO leaderboard). Remaining: uplift/delta like-for-like, cross-mode/harness labelling, tests, **W8 (P0, added 2026-08-07): harness errors scored as capability failures**, and **W9 (added 2026-08-11): the coverage gate compares counts, not benchmark-set identity**.
**Quorum**: 2 rounds run 2026-08-11 (iter-178), artifacts `m-eval-validity-discipline-2026-08-11T17-46-03Z.json` and `…T17-49-21Z.json`. Both rounds **BLOCKED**, both reviewers **present** (`absent_reviewers: []` — no N−1 hole), metered **$0.0955** total. Every objection from both rounds was measured rather than forwarded and its `proposed_fix` adopted VERBATIM (R1→W9 ACs, R2→AC-W8.3 + the Conflict Surface). **The doc is NOT cleared to route**: round 2's surviving R1 objection disputes the design *direction* of **W9**, so per the mission-control carve-out this parks `needs-human-review` rather than taking a controller-authored third round. **W8 is untouched by either objection** — whether W8 may route on its own, in its own scoped doc, is the human decision on the bookkeeping issue.
**Target**: v0.30.x (eval infrastructure + dashboard)
**Priority**: P1 — every benchmark run has surfaced a *new manifestation of the same class of bug* (invalid cross-cohort comparison). This is the fix that stops the cycle.
**Author**: Claude Opus 4.8 (requested by Mark, 2026-07-11 — "every benchmark run we need to fix all this data and reliability")

---

## The recurring bug (one class, many faces)
The eval **display** has no validity discipline, so each run surfaces a new invalid comparison:
- **Sparse local models posted a bogus #1** — qwen ran 6 benchmarks, cloud ran 55; the merged ELO ranked qwen top because ELO across *disjoint benchmark sets* isn't comparable.
- **"Agent uplift" was mostly not like-for-like** — standard included the 2 impossible reimplements (0%) that agent excluded; and several "uplifts" compared `or-deepseek` (standard API) vs `opencode-or-deepseek` (agent CLI) — different harness/ID; haiku had no standard baseline at all.
- **fable's 60% Python** was 16 genuine-but-spurious API refusals counted as capability fails.
- **glm-5.2's low score** was reasoning-token accounting + un-logged code (see [m-eval-reasoning-model-fairness](m-eval-reasoning-model-fairness.md)).

None were random — they're all *comparing things that aren't comparable*, then publishing the result as a headline.

## Principle
**A number is only comparable to another number if it was measured the same way, on the same benchmarks.** The display must enforce this — not the reader, and not a per-run manual audit.

## Rules (enforced in code, not convention)
1. **Coverage is first-class.** Every model in the ratings block carries `benchmarks` (distinct count) + the block carries `maxCoverage`. ✅ done.
2. **Coverage annotation, not hiding.** Every model stays **visible** in the leaderboard (the local models are the point of the server). Models below the coverage threshold are marked **provisional** — dimmed, italic, no medal, with a coverage badge — so a 6-benchmark ELO is visible but can't be misread as beating a 55-benchmark one. Full ranking (medal, undimmed) is earned as coverage fills in. ✅ done (ELO leaderboard). **Corrected 2026-08-11 (iter-178, measured — this rule read "currently 50%" and that was stale):** there are **two** thresholds, deliberately, in `docs/src/components/BenchmarkDashboard/coverageGate.js` — `RATE_COVERAGE_FRACTION = 0.5` and `ELO_COVERAGE_FRACTION = 0.9`. A pass *rate* degrades gracefully with coverage; a *rating* does not, so ELO demands near-identical sets. Anyone quoting "50%" against the ELO board is quoting this doc, not the code.
3. **Like-for-like deltas.** Any *cross-mode* (standard→agent) or *cross-run* delta is computed over the **intersection** of benchmarks the two cohorts both ran, and only across a **matching (model, harness) identity**. `or-X` (API-standard) vs `opencode-or-X` (agent-CLI) is a *harness* comparison and must be labelled as such, not presented as "uplift". ⏳ remaining.
4. **Label the axis.** Standard vs agent, API vs CLI-harness, per-language vs blended — every comparison states what it holds constant.
5. **Tests are the guardrail.** Distribution/validity invariants are unit-tested so a re-tier or a new cohort can't silently break them (the v0.29.2 re-tier already broke two drift detectors — caught late).

## Work items
- **W1 (done)** — `benchmarks` + `maxCoverage` in `ratings_export.go`; ELO leaderboard marks under-covered models provisional (dimmed, no medal, coverage badge).
- **W2 (CLI ✅ · dashboards mostly ✅)** — `ailang eval-elo` tracks per-model AILANG coverage + flags `provisional` below 50% of `max_coverage` (Cov column + `--json`; 7abbb3218). **Frontend:** a shared `coverageGate.js` helper (`buildCoverage` → `id → {benchmarks, provisional}` + `maxCoverage`, reads the W1 ratings block) is threaded through both dashboards. Gated + **CI-docs-build-verified** (tip 9b4f4f890): `ModelChart` (dimmed bars, ⚠, sorted last — kills the live bogus-#1 where the 3 local qwen at 9/56 ranked as headline), `ModelComparisonTable` (true-coverage badge replaces the run-count heuristic; dimmed rows), `ValueScoreTable` (provisional models get **no medal**, sort last, dimmed), `QualityScatter` (provisional excluded from the Pareto frontier), `ModelDeltaTrend` (provisional models filtered from the delta trend + provider average). `DollarsPerPassTable` is **dead code** (not mounted on any page) — no gating needed. `ComparisonTable`/`BenchmarkChampionsTable`/`OSLocalLeaderboard` were already safe. **All live dashboard rank/compare views are now coverage-gated.** **Verification note:** local full build is blocked by a pre-existing `sidebars.js` quirk (Node-26 rig), so verify via per-file babel compile + coverage-logic-on-real-data + the CI docs build (never trust a background exit code — read the log / CI check).
- **W3 (done ✅)** — `ComputeUplift` (`internal/eval_analysis/uplift.go`): shared-benchmark, matching-identity standard→agent delta; wired into the ratings block as `ratings.uplift`, and **surfaced on the site** via `AgentUpliftTable` on the Agent Harness Explorer page. Verified on v0.29.2 (haiku +51.8%, sonnet +25.4%, luna −32.6%). Mismatched identity (`or-X` vs `opencode-or-X`) is excluded as a harness comparison, not uplift. (5 unit tests.) Gotcha fixed: the unified-publish script preferred a stale repo `bin/ailang` over PATH, silently dropping the uplift block — removed it.
- **W4 (partial ✅)** — done: `ComputeUplift` tests (shared-benchmark/identity/macro-avg/lang-scope) + `eval-elo` coverage-gating test (full/sparse/at-threshold). Remaining: a "no ranked model below threshold" invariant on the dashboard board once W2-frontend lands; keep the tier-distribution detectors in sync with the corpus.
- **W5** — apply the same discipline to the merged local↔cloud board ([m-eval-local-cloud-unify](m-eval-local-cloud-unify.md)) — local only enters full (non-provisional) ranking once its AILANG coverage matches. The rig is now **AILANG-first**: `os-rotation-filler.sh` fills every core+stretch+frontier AILANG benchmark for the current version *first* (default), then auto-hands-off to the cross-language pass; a new release resets coverage so AILANG-first resumes. Completeness = every full-tier bench banked for every local model, OR one full AILANG lap (deadlock-safe against benchmarks a weak model can't pass).

- **W8 (NEW 2026-08-07, P0 — ailang#619)** — **the OS leaderboard publisher counts harness errors as capability failures.** Same class as the fable "60% Python = 16 API refusals counted as capability fails" face above, but in the *publisher* rather than the display. `cmd/ailang/eval_publish.go` computes `PassRate = Passed / Trials` over **every banked row** and never reads `validity` — even though the harness already writes `validity: {valid: false, reason: "harness_error"}` on exactly these rows (`internal/eval_harness/validity_backstop*`). Concretely on 2026-08-07: 30 `api_error` rows from the ollama 300s-timeout cascade ([m-ollama-v1-streaming-idle-timeout](m-ollama-v1-streaming-idle-timeout.md), ailang#618) put motoko-local's published v0.33.0 **frontier at exactly `3/22 = 0.13636363636363635`** — bit-for-bit the published value — where **17 of the 22 were harness timeouts**. True figure ≈ 60% (n=5): a ~4× understatement, live on the dashboard and synced to the bucket. Frozen wrong, too, because `--skip-existing` treats a banked `api_error` as done, so those combos never re-run for that version.
  - **Fix:** exclude `validity.valid == false` rows from BOTH numerator and denominator, and surface the excluded count (`n=5 (17 invalid excluded)`) rather than silently shrinking `n` — a silent drop trades one invisible bug for another. Per Critical Principle 2, a harness error must never be scored as a capability failure.
  - **Also:** `--skip-existing` should not treat an invalid row as satisfying a combo (otherwise every harness outage permanently poisons that version's bank). Deleting the rows is the current manual workaround — done 2026-08-07, 30 rows removed after backup.
  - **Tests:** a banked `validity.valid=false` row must not move a published pass rate; the excluded-count must appear.

  ### W8 — controller reality-check, iteration 178 (2026-08-11), base `5f471b2b7`

  Every row below was re-derived by command in-session; a `0` is paired with a known-positive
  control in the same call, so an empty result is a measurement and not a broken instrument
  (Gate 2 rule 3a). **Two of the three claims above are wrong about WHERE, and one is already
  fixed** — the corrections are load-bearing for scoping, not editorial.

  | # | Claim under test | Command | Result | Verdict |
  |---|---|---|---|---|
  | V1 | `cmd/ailang/eval_publish.go` never reads `validity` | `grep -ci validity cmd/ailang/eval_publish.go` / control `grep -c PassRate` | **0** / control **5** | TRUE |
  | V2 | …and it is therefore the fix site | `grep -n 'LoadResults\|eval_analysis\.' cmd/ailang/eval_publish.go` | **0 hits** — the publisher reads rotation `summary.json`, it does not aggregate raw rows | **FALSE — wrong site** |
  | V3 | The real aggregation point | `internal/eval_harness/rotation_summary.go:246` | `PassRate: float64(passed) / float64(len(g.Trials))`; the loop at `:222` counts `passed` off `CompileOk && RuntimeOk && StdoutOk` with **zero** `IsValid()` reads (`grep -ci 'validity\|IsValid'` = **0**, control `PassRate` = **8**) | **`SummarizeRotation` is the defect site** |
  | V4 | The rollup is unguarded too | `rotation_summary.go:294-307` | `ModelRollupStats.PassAt1 = passTrials / trials`, same unfiltered `BenchmarkSummary` sums | TRUE — second numerator |
  | V5 | The harness marks these rows | `internal/eval_harness/validity.go:51,89` | `ReasonHarnessError = "harness_error"`; `func (m *RunMetrics) IsValid() bool { return m.Validity == nil \|\| m.Validity.Valid }` | TRUE |
  | V6 | `--skip-existing` still treats an invalid row as done | `cmd/ailang/eval_skip_existing.go` | `hasValidBankedResult` already gates on `row.IsValid()`; landed `f3189541a` (2026-07-29), ancestor of `origin/dev` | **ALREADY FIXED — drop from scope** |
  | V7 | A filter helper already exists and can be reused | `internal/eval_analysis/validity_filter.go`; direction measured with `go list -deps` (authoritative, not grep) | `FilterValidResults` + `CountInvalid` exist, called from `loader.go:54` only. But `eval_analysis -> eval_harness` = **2**, `eval_harness -> eval_analysis` = **0** (control: `eval_harness` has **25** internal deps) — importing them into `SummarizeRotation` is an import **cycle** | Helper exists, **reuse NOT available at the defect site**; `eval_harness` needs its own guard off the `RunMetrics.IsValid()` that already lives there (V5) |
  | V8 | There is an in-repo idiom for surfacing a shrunken sample | `rotation_summary.go:56-59` | `TokensCacheUnaccounted` — *"a shrunken sample stated out loud rather than a silent one"* (2026-08-11) | TRUE — **follow this shape** |
  | V9 | The published board can carry the excluded count today | `jq '[.rows[0]\|paths(scalars)]' docs/static/benchmarks/os/latest.json` | every leaf is a bare rate (`lang.*`, `tiers.*.*`); **no `n`, no denominator, no exclusion field** | FALSE — the JSON schema + dashboard need the field |
  | V10 | The defect is live, not historical | `find eval_results -name '*.json' ! -name summary.json \| head -4000 \| xargs grep -l '"valid":[[:space:]]*false' \| wc -l` | **160** invalid rows; control (rows carrying any `validity` block) = **160** | TRUE — live in the bank |
  | V11 | The specific `3/22` instance | `jq .rows` on the rig-synced `latest.json` (v0.33.0, generated 2026-08-11) | motoko-local frontier now `0.25`; the 2026-08-07 manual row deletion cleared *that* instance | Instance cleared, **defect stands** |

  **Scope consequence.** The fix is ONE guard at ONE aggregation point (`SummarizeRotation`),
  plus surfacing. `eval_publish.go` changes only to carry the count through to the board; and the
  `--skip-existing` bullet is already closed by `f3189541a` and is struck from the sprint.
  Re-publishing an already-banked rotation needs `--summarize` (`eval_publish.go:89`), because a
  `summary.json` written before the fix has the wrong `passed`/`trials` baked in — that is an
  operational note for the rollout, not repo work.

  ### W8 acceptance criteria (scoped; the umbrella ACs below do not cover W8)

  - **AC-W8.1** — `SummarizeRotation` excludes `!row.IsValid()` rows from BOTH `Passed` and
    `Trials` in every `BenchmarkSummary`, and from `ModelRollupStats.PassAt1`/`Trials`.
  - **AC-W8.2** — the exclusion is COUNTED, never silent: `BenchmarkSummary` and
    `ModelRollupStats` each carry an `invalid_excluded` count, following the
    `TokensCacheUnaccounted` idiom (V8).
  - **AC-W8.3** — a group whose trials are ALL invalid must not publish `NaN` (`0/0`) or a
    fabricated `0.0`; it is a measurement of nothing and must be representable as such.
    **Schema migration is part of this AC** (quorum R1, `gemini-3-1-pro`, fix adopted VERBATIM):
    *"migrating `PassRate` and `PassAt1` in `BenchmarkSummary` and `ModelRollupStats` from
    `float64` to `*float64`. This allows a 0-valid-trial result to be set to `nil`, serializing
    cleanly to `null` in JSON instead of triggering an unsupported value error on NaN."*
    The objection is correct and its consequence is a crash, not a cosmetic one: `encoding/json`
    **errors** on `NaN`, and V3/V9 measured both fields as bare `float64`, so leaving the structs
    unchanged turns an all-invalid cohort into a failed publish.

  #### Conflict Surface — `summary.json` consumers of `pass_rate` / `pass_at_1`

  Added at quorum R2 (`gemini-3-1-pro`, round 2), whose objection was that AC-W8.3's first draft
  *"waves this off ('call that out in the milestone') instead of mapping the conflict surface"*.
  **The objection is correct AND understated** — measured at base `5f471b2b7`, the migration is
  not confined to `rotation_summary.go`:

  | Consumer | Evidence | Hazard |
  |---|---|---|
  | `internal/eval_harness/rotation_summary.go:33,69` | `PassRate float64` / `PassAt1 float64` — the producer | the fields being migrated |
  | `cmd/ailang/eval_trend.go:70` | **its own** `PassRate float64 \`json:"pass_rate"\`` (12 `PassRate` refs, 1 `RotationSummary` ref) | a `null` unmarshals to the zero value **silently** — an all-invalid cohort reads as a real `0.0` trend point |
  | `tools/build-snapshot/main.go:588` | **its own** `PassRate float64 \`json:"pass_rate"\`` | same silent `0.0`, in the snapshot the site ships |
  | `internal/eval_analysis/sweet_spot.go:54` | `PassRate float64 \`json:"pass_rate"\`` | same class; confirm whether its input is a rotation summary before migrating |
  | `cmd/ailang/eval_publish.go` | 5 `PassRate` refs, sums `BenchmarkSummary` | must carry the count through (AC-W8.4) |
  | 14 shell/JS consumers (`tools/os-release-snapshot.sh`, the `eval-analyzer` / `eval-gap-finder` / `post-release` skill scripts, ×2 for the `.claude`/`.agents` skill copies) | `git grep -l 'summary.json'` over `*.sh *.py *.js` | `jq` arithmetic on a `null` yields `null`, not an error — a silently empty column |

  Control: `git grep -l 'summary.json'` returns **78** files repo-wide, so the 11-file Go subset
  above is a filtered result and not an empty instrument.

  **Reviewer's `proposed_fix`, adopted VERBATIM as AC-W8.3's remaining half:** *"Add a Conflict
  Surface section that specifically identifies all downstream consumers of `summary.json` (e.g.,
  measured via `git grep -l 'summary.json'`). Require that all consuming structs be updated to
  `*float64` in the same commit, and add a validation step in the consumers that hard-fails or
  explicitly skips when `PassRate == nil`, preventing silent 0.0 defaults."* Per Critical
  Principle 2 a `nil` rate is a no-measurement and must never be rendered as `0.0`; the milestone
  that migrates the producer migrates **every** struct in the table above in the same commit, and
  each consumer gets an explicit nil branch (skip-and-count, never a zero).
  - **AC-W8.4** — `eval-publish` surfaces the excluded count on the OS board JSON and the
    generated page; a bare rate with a silently shrunken denominator is the bug, not the fix.
  - **AC-W8.5** — tests: a fixture rotation containing `validity.valid=false` rows publishes the
    SAME pass rate as the fixture with those rows removed, and a DIFFERENT one from the fixture
    with them counted. Each new assertion names the mutation it kills, and the mutation is run
    per-row with only that test selected (skill rule 3i).
  - **AC-W8.6** — no gate is vacuous at base: every acceptance command is baselined on unmodified
    `dev` and recorded (rule 3e).

- **W9 (NEW 2026-08-11, iter-178 — raised by quorum R1, `gpt5-6-sol`, and MEASURED before adoption)** — **the coverage gate compares COUNTS, never benchmark-set IDENTITY, so two models on disjoint sets of equal size are ranked as comparable.** This is the doc's own headline defect surviving inside the fix for it: Rule 1 makes "coverage" a scalar, and a scalar cannot express "measured on the same benchmarks". Measured at base `5f471b2b7`:

  | # | Claim | Command | Result |
  |---|---|---|---|
  | V12 | The gate is count-based | `coverageGate.js` | `buildCoverage` builds `id → n` from `m.benchmarks` (an integer); `isProvisional`/`isProvisionalForElo` compare that integer to a fraction of `maxCoverage`. **No benchmark ID is read anywhere in the file**: `grep -ci 'benchmark_id\|benchmarkId\|benchIds'` = **0**, controls `benchmarks` = **6** and `maxCoverage` = **13** firing in the same call |
  | V13 | The producer HAS the set but exports only its size | `internal/eval_analysis/ratings_export.go:107-117` | `maxCoverage = len(bs)`; `"benchmarks": len(modelBenches[id])` — `modelBenches[id]` **is** the set, and only `len` crosses the boundary |
  | V14 | The reviewer's threshold detail was quoting this doc, not the code | `coverageGate.js:26-27` | `RATE_COVERAGE_FRACTION = 0.5`, `ELO_COVERAGE_FRACTION = 0.9` — so "promotes at 50% into the full ELO ranking" is **FALSE** for ELO and TRUE for rates. The objection's *core* (count ≠ set) survives both thresholds |

  **W9 acceptance criteria — the round-1 sketch was REPLACED at round 2 by `gpt5-6-sol`'s own
  text, adopted VERBATIM.** The reviewer's round-2 objection is sound and is the reason the first
  draft could not stand: a *pairwise* shared-intersection rating *"can produce different samples
  and ratings for every model pair, so it cannot support one deterministic, transitive
  leaderboard"* — i.e. the round-1 sketch would have traded an invalid comparison for a
  non-transitive ordering, which is the same class of bug one layer down. Adopted:

  > *"Each ratings export includes sorted completed benchmark IDs and a deterministic required-set
  > ID computed from the benchmark corpus/version and tier policy. Full ELO ranking is partitioned
  > by exact required-set ID, and a model is medal-eligible only when its completed set equals
  > that cohort's required set. Models with missing, extra, or differently composed sets remain
  > visible and provisional but are not ordered against the full cohort. Count thresholds are
  > annotations only. Optional intersection ratings are separate, explicitly labelled artifacts
  > for one fixed, named intersection shared by every model shown; pair-specific intersections
  > never contribute to a global leaderboard. Corpus or re-tier changes create a new required-set
  > ID and cannot silently reuse the prior cohort."*

  Plus, per the same objection: verification rows locating the existing ELO cohort/grouping code
  and an inventory of whether it can enforce exact-set cohorts; and tests for equal-count disjoint
  sets, same-count different-composition sets, subsets, extras, release/re-tier changes, stable
  set-ID generation, and the absence of cross-cohort ordering. **Explicit no-data behaviour is
  required rather than silently retaining the count-based gate.**

  ⚠ **W9 IS PARKED FOR HUMAN RATIFICATION, NOT ROUTED.** The controller adopted the reviewer's
  text but has NOT verified the ELO cohort/grouping inventory it calls for, and partitioning the
  public leaderboard by required-set ID is a visible product change — see the DECISIONS ask on the
  bookkeeping issue.

  **Scope:** NOT part of the W8 sprint (W8 is one guard at one aggregation point in
  `eval_harness`; W9 is a schema + display change in `eval_analysis` + the dashboard). Queued
  separately so W8's P0 is not held behind it. **AC1 below is consequently NOT satisfied today**
  — see the note there.

## Acceptance criteria
1. No dashboard/leaderboard/CLI ranks a model against others on a materially different benchmark set without a coverage annotation + gate. ⚠ **NOT SATISFIED as of 2026-08-11 (iter-178)** — the shipped gate annotates on benchmark *count*, so "materially different benchmark set" is not actually tested; equal-count disjoint sets pass it. Closing this AC is **W9**, not W1/W2. Recorded rather than quietly claimed, because a doc that marks its own headline AC done is how the next reader stops checking.
2. Every "uplift"/delta on the site is over shared benchmarks + matching identity, or is explicitly labelled as a harness/mode comparison.
3. Unit tests enforce the gating + distribution invariants; they fail loudly on a new cohort or a re-tier.

## Out of scope
- Changing the ELO math itself (it's fine within a cohort; the issue is *cross-cohort* display).
- The per-model correctness fixes tracked separately ([m-eval-reasoning-model-fairness](m-eval-reasoning-model-fairness.md)).
