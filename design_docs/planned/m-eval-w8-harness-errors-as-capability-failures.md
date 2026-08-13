# M-EVAL-W8: harness errors must not be scored as capability failures

**Status**: Planned — **split out of [m-eval-validity-discipline](m-eval-validity-discipline.md) on 2026-08-13 (Mark) so it can route independently of the disputed W9.**
**Target**: v0.33.1
**Priority**: P0 — ailang#619. A published OS-board figure was a ~4× understatement of real capability.
**Estimated**: 1-2 days
**Dependencies**: Pairs with [m-eval-failure-attribution](m-eval-failure-attribution.md) — see "Relationship to failure-attribution". Sequence that one first for standard-mode data; this one stands alone for agent/rotation data.
**Provenance**: Authored as W8 of m-eval-validity-discipline (2026-08-07). Reality-checked in-session at iteration 178 (2026-08-11, base `5f471b2b7`). Quorum R1/R2 objections on AC-W8.3 adopted VERBATIM. Content below is carried over unchanged except where marked.

---

## Why this is its own document

The parent doc reached **BLOCKED** at quorum on 2026-08-11 across two rounds. The surviving objection disputes the design *direction of W9* (whether the coverage gate should compare benchmark-set identity rather than counts). The parent records that **W8 is untouched by either objection**, and that the open human decision was whether W8 may route on its own, in its own scoped doc.

**Mark took that decision on 2026-08-13: yes.** W8 is P0, fully reality-checked, has complete acceptance criteria, and is blocked only by being stapled to a disputed sibling. This doc is that split. W9 and the rest of the umbrella stay in the parent.

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No execution-semantics change; a filter over already-banked rows |
| A2: Replayability | 0 | No trace/replay impact |
| A3: Effect Legibility | 0 | Not touched |
| A4: Explicit Authority | 0 | Not touched |
| A5: Bounded Verification | +1 | A published rate becomes checkable against a stated denominator |
| A6: Safe Concurrency | 0 | Not touched |
| A7: Machines First | +1 | The OS board and its bucket sync are machine-consumed; a 4× wrong rate misroutes real decisions |
| A8: Minimal Syntax | 0 | Eval tooling only |
| A9: Cost Visibility | +1 | Stops a poisoned bank from freezing combos out of re-runs |
| A10: Composability | 0 | Not touched |
| A11: Structured Failure | +1 | A no-measurement becomes representable (`null`) instead of a fabricated `0.0` |
| A12: System Boundary | +1 | Separates apparatus failure from subject failure at the aggregation boundary |

**Net Score: +5** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism
- [x] A3 (Effects): no hidden side effects
- [x] A4 (Authority): no ambient access
- [x] A7 (Machines First): improves machine-analyzable output

---

## Problem Statement

**The OS leaderboard publisher counts harness errors as capability failures.** Same class as the fable "60% Python = 16 API refusals counted as capability fails" face in the parent doc, but in the *publisher* rather than the display.

The harness already writes `validity: {valid: false, reason: "harness_error"}` on exactly these rows — and the aggregation never reads it.

Concretely on 2026-08-07: 30 `api_error` rows from the ollama 300s-timeout cascade ([m-ollama-v1-streaming-idle-timeout](m-ollama-v1-streaming-idle-timeout.md), ailang#618) put motoko-local's published v0.33.0 **frontier at exactly `3/22 = 0.13636363636363635`** — bit-for-bit the published value — where **17 of the 22 were harness timeouts**. True figure ≈ 60% (n=5): a **~4× understatement**, live on the dashboard and synced to the bucket.

**Fix:** exclude `validity.valid == false` rows from BOTH numerator and denominator, and surface the excluded count (`n=5 (17 invalid excluded)`) rather than silently shrinking `n` — a silent drop trades one invisible bug for another. Per CLAUDE.md Critical Principle 2, a harness error must never be scored as a capability failure.

---

## Controller reality-check, iteration 178 (2026-08-11), base `5f471b2b7`

Every row re-derived by command in-session; a `0` is paired with a known-positive control in the same call, so an empty result is a measurement and not a broken instrument (Gate 2 rule 3a). **Two of the three original claims are wrong about WHERE, and one is already fixed** — the corrections are load-bearing for scoping, not editorial.

| # | Claim under test | Command | Result | Verdict |
|---|---|---|---|---|
| V1 | `cmd/ailang/eval_publish.go` never reads `validity` | `grep -ci validity cmd/ailang/eval_publish.go` / control `grep -c PassRate` | **0** / control **5** | TRUE |
| V2 | …and it is therefore the fix site | `grep -n 'LoadResults\|eval_analysis\.' cmd/ailang/eval_publish.go` | **0 hits** — the publisher reads rotation `summary.json`, it does not aggregate raw rows | **FALSE — wrong site** |
| V3 | The real aggregation point | `internal/eval_harness/rotation_summary.go:246` | `PassRate: float64(passed) / float64(len(g.Trials))`; the loop at `:222` counts `passed` off `CompileOk && RuntimeOk && StdoutOk` with **zero** `IsValid()` reads (`grep -ci 'validity\|IsValid'` = **0**, control `PassRate` = **8**) | **`SummarizeRotation` is the defect site** |
| V4 | The rollup is unguarded too | `rotation_summary.go:294-307` | `ModelRollupStats.PassAt1 = passTrials / trials`, same unfiltered `BenchmarkSummary` sums | TRUE — second numerator |
| V5 | The harness marks these rows | `internal/eval_harness/validity.go:51,89` | `ReasonHarnessError = "harness_error"`; `func (m *RunMetrics) IsValid() bool { return m.Validity == nil \|\| m.Validity.Valid }` | TRUE |
| V6 | `--skip-existing` still treats an invalid row as done | `cmd/ailang/eval_skip_existing.go` | `hasValidBankedResult` already gates on `row.IsValid()`; landed `f3189541a` (2026-07-29), ancestor of `origin/dev` | **ALREADY FIXED — dropped from scope** |
| V7 | A filter helper already exists and can be reused | `internal/eval_analysis/validity_filter.go`; direction measured with `go list -deps` (authoritative, not grep) | `FilterValidResults` + `CountInvalid` exist, called from `loader.go:54` only. But `eval_analysis -> eval_harness` = **2**, `eval_harness -> eval_analysis` = **0** (control: `eval_harness` has **25** internal deps) — importing them into `SummarizeRotation` is an import **cycle** | Helper exists, **reuse NOT available at the defect site**; `eval_harness` needs its own guard off the `RunMetrics.IsValid()` that already lives there (V5) |
| V8 | There is an in-repo idiom for surfacing a shrunken sample | `rotation_summary.go:56-59` | `TokensCacheUnaccounted` — *"a shrunken sample stated out loud rather than a silent one"* | TRUE — **follow this shape** |
| V9 | The published board can carry the excluded count today | `jq '[.rows[0]\|paths(scalars)]' docs/static/benchmarks/os/latest.json` | every leaf is a bare rate (`lang.*`, `tiers.*.*`); **no `n`, no denominator, no exclusion field** | FALSE — the JSON schema + dashboard need the field |
| V10 | The defect is live, not historical | `find eval_results -name '*.json' ! -name summary.json \| xargs grep -l '"valid":[[:space:]]*false' \| wc -l` | **160** invalid rows; control (rows carrying any `validity` block) = **160** | TRUE — live in the bank |
| V11 | The specific `3/22` instance | `jq .rows` on the rig-synced `latest.json` (v0.33.0, generated 2026-08-11) | motoko-local frontier now `0.25`; the 2026-08-07 manual row deletion cleared *that* instance | Instance cleared, **defect stands** |
| **V12** | **(added 2026-08-13 at split) Where W8's target rows actually live** | `xargs grep -l '"valid":[[:space:]]*false'` over `eval_results/**`, grouped by top dir; control = rows carrying any `validity` block | **147** invalid of **160** with a block; `motoko_full_core_matrix` **80**, `motoko_profile_matrix` **10**, `ab_conv_docx_*` **20** | TRUE — **W8's value is concentrated in agent/motoko rotation data**, which is precisely the OS-board population it publishes |

**Scope consequence.** The fix is ONE guard at ONE aggregation point (`SummarizeRotation`), plus surfacing. `eval_publish.go` changes only to carry the count through to the board; the `--skip-existing` bullet is already closed by `f3189541a` and is struck from the sprint. Re-publishing an already-banked rotation needs `--summarize` (`eval_publish.go:89`), because a `summary.json` written before the fix has the wrong `passed`/`trials` baked in — an operational note for the rollout, not repo work.

---

## Acceptance criteria

- **AC-W8.1** — `SummarizeRotation` excludes `!row.IsValid()` rows from BOTH `Passed` and `Trials` in every `BenchmarkSummary`, and from `ModelRollupStats.PassAt1`/`Trials`.
- **AC-W8.2** — the exclusion is COUNTED, never silent: `BenchmarkSummary` and `ModelRollupStats` each carry an `invalid_excluded` count, following the `TokensCacheUnaccounted` idiom (V8).
- **AC-W8.3** — a group whose trials are ALL invalid must not publish `NaN` (`0/0`) or a fabricated `0.0`; it is a measurement of nothing and must be representable as such. **Schema migration is part of this AC** (quorum R1, `gemini-3-1-pro`, fix adopted VERBATIM):
  > *"migrating `PassRate` and `PassAt1` in `BenchmarkSummary` and `ModelRollupStats` from `float64` to `*float64`. This allows a 0-valid-trial result to be set to `nil`, serializing cleanly to `null` in JSON instead of triggering an unsupported value error on NaN."*

  The objection is correct and its consequence is a crash, not a cosmetic one: `encoding/json` **errors** on `NaN`, and V3/V9 measured both fields as bare `float64`, so leaving the structs unchanged turns an all-invalid cohort into a failed publish.
- **AC-W8.4** — `eval-publish` surfaces the excluded count on the OS board JSON and the generated page; a bare rate with a silently shrunken denominator is the bug, not the fix.
- **AC-W8.5** — tests: a fixture rotation containing `validity.valid=false` rows publishes the SAME pass rate as the fixture with those rows removed, and a DIFFERENT one from the fixture with them counted. Each new assertion names the mutation it kills, and the mutation is run per-row with only that test selected (skill rule 3i).
- **AC-W8.6** — no gate is vacuous at base: every acceptance command is baselined on unmodified `dev` and recorded (rule 3e).

### Conflict Surface — `summary.json` consumers of `pass_rate` / `pass_at_1`

Added at quorum R2 (`gemini-3-1-pro`, round 2), whose objection was that AC-W8.3's first draft *"waves this off ('call that out in the milestone') instead of mapping the conflict surface"*. **The objection is correct AND understated** — measured at base `5f471b2b7`, the migration is not confined to `rotation_summary.go`:

| Consumer | Evidence | Hazard |
|---|---|---|
| `internal/eval_harness/rotation_summary.go:33,69` | `PassRate float64` / `PassAt1 float64` — the producer | the fields being migrated |
| `cmd/ailang/eval_trend.go:70` | **its own** `PassRate float64 \`json:"pass_rate"\`` (12 `PassRate` refs, 1 `RotationSummary` ref) | a `null` unmarshals to the zero value **silently** — an all-invalid cohort reads as a real `0.0` trend point |
| `tools/build-snapshot/main.go:588` | **its own** `PassRate float64 \`json:"pass_rate"\`` | same silent `0.0`, in the snapshot the site ships |
| `internal/eval_analysis/sweet_spot.go:54` | `PassRate float64 \`json:"pass_rate"\`` | same class; confirm whether its input is a rotation summary before migrating |
| `cmd/ailang/eval_publish.go` | 5 `PassRate` refs, sums `BenchmarkSummary` | must carry the count through (AC-W8.4) |
| 14 shell/JS consumers (`tools/os-release-snapshot.sh`, the `eval-analyzer` / `eval-gap-finder` / `post-release` skill scripts, ×2 for the `.claude`/`.agents` skill copies) | `git grep -l 'summary.json'` over `*.sh *.py *.js` | `jq` arithmetic on a `null` yields `null`, not an error — a silently empty column |

Control: `git grep -l 'summary.json'` returns **78** files repo-wide, so the 11-file Go subset above is a filtered result and not an empty instrument.

**Reviewer's `proposed_fix`, adopted VERBATIM as AC-W8.3's remaining half:** *"Add a Conflict Surface section that specifically identifies all downstream consumers of `summary.json` (e.g., measured via `git grep -l 'summary.json'`). Require that all consuming structs be updated to `*float64` in the same commit, and add a validation step in the consumers that hard-fails or explicitly skips when `PassRate == nil`, preventing silent 0.0 defaults."*

Per Critical Principle 2 a `nil` rate is a no-measurement and must never be rendered as `0.0`; the milestone that migrates the producer migrates **every** struct in the table above in the same commit, and each consumer gets an explicit nil branch (skip-and-count, never a zero).

---

## Relationship to failure-attribution

[m-eval-failure-attribution](m-eval-failure-attribution.md) is the **producer** half: it decides *whose fault* a failure was and sets validity accordingly. This doc is the **consumer** half: it stops aggregation from counting rows already marked invalid.

They cover disjoint populations, split by mode (measured 2026-08-13):

- **Agent / motoko rotation data** — 147 invalid rows tree-wide, 80 in `motoko_full_core_matrix` alone (V12). These are mostly `api_error` cascades, which today's backstop **does** catch. **W8 delivers here on its own** and does not need the producer doc.
- **Standard-mode release baselines** — only 4 of 877 rows in the v0.32.0 baseline carry `validity.valid=false`, because standard-mode contamination wears `runtime_error`/`logic_error`, which the backstop never inspects. **W8 needs the producer doc to have anything to filter here.**

Neither supersedes the other. If both are scheduled, run producer-first so the standard-mode population is populated before W8 filters it; if only one is scheduled, W8 still fixes the published OS board.

---

## Success Criteria

- [ ] AC-W8.1 through AC-W8.6 all satisfied
- [ ] Every struct in the Conflict Surface table migrated to `*float64` **in the same commit**
- [ ] Each consumer has an explicit nil branch that skips-and-counts, never renders `0.0`
- [ ] Re-publishing the v0.33.0 rotation with `--summarize` produces a frontier rate with a stated excluded count
- [ ] `make test` green, `make lint` clean, `make check-boundaries` green

## Non-Goals

- **W9 and the rest of the umbrella.** They stay in [m-eval-validity-discipline](m-eval-validity-discipline.md); W9's direction is disputed and parked `needs-human-review`.
- **`--skip-existing` handling.** Already fixed by `f3189541a` (V6).
- **Producer-side classification.** That is [m-eval-failure-attribution](m-eval-failure-attribution.md).

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| A partial `*float64` migration leaves a consumer silently reading `0.0` | High — reintroduces the bug in a new place | Same-commit migration of every struct in the Conflict Surface table; explicit nil branch per consumer, each with a test |
| Re-publishing stale `summary.json` files carries the pre-fix numbers forward | Med | Rollout requires `--summarize`; recorded as an operational note |
| Splitting from the parent loses the quorum history | Low | Full R1/R2 provenance and the verbatim `proposed_fix` text are carried over above |

## Related Documents

- [m-eval-validity-discipline.md](m-eval-validity-discipline.md) — parent; W1-W7, W9 remain there
- [m-eval-failure-attribution.md](m-eval-failure-attribution.md) — producer half
- [m-ollama-v1-streaming-idle-timeout.md](m-ollama-v1-streaming-idle-timeout.md) — ailang#618, source of the 30-row cascade

## References

- ailang#619
- CLAUDE.md Critical Principle 2 — no silent fallbacks

---

**Document created**: 2026-08-13 (split from m-eval-validity-discipline, authored 2026-08-07)
**Last updated**: 2026-08-13
