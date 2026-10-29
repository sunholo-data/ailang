# Sprint Plan: M-EFFECT-ROW-OPEN-CLOSED-SOUNDNESS (#1958)

**Status:** IMPLEMENTATION COMPLETE / INDEPENDENT EVALUATION PENDING — Mark authorized “Execute sprint #1958” on 2026-10-29.
**Design:** `design_docs/planned/v0_55_0/m-effect-row-open-closed-soundness.md`
**Intended repository plan path:** `design_docs/planned/v0_55_0/m-effect-row-open-closed-soundness-sprint-plan.md`
**Intended progress path:** `.ailang/state/sprints/sprint_M-EFFECT-ROW-OPEN-CLOSED-SOUNDNESS.json`
**Measured base:** `origin/dev` `c04a44780` (2026-10-29).
**Duration:** 2.5 engineering days, four resumable milestones of approximately 60–90 minutes of agent implementation/validation each, plus corpus verification and review allowance.
**Release target:** v0.56.0, the next minor after v0.55.0 was tagged during execution. No patch release, version bump, or deployment is authorized by this plan.
**Priority:** P0. **Risk:** high (row-unification semantics); limited by approved migration and discriminator fixtures.
**AILANG prompt version loaded:** v0.16.8 (`ailang prompt` loaded from the archived-source binary; active version confirmed).

## Summary

Reject an annotated effectful callback when it is passed through a closed pure slot, so an open row cannot lose its own labels. First make both std/ai streaming wrappers and their builtins truthfully propagate callback effects; then enforce the symmetric open/closed row side condition. Width subsumption and the nested-callback fix are outside this sprint.

## Authorization and human decisions

The design's F1, F2, and F3 are all explicitly **RULED 2026-10-26 by Mark, attended**:

- F1: streaming callbacks become row-polymorphic (`StreamChunk -> () ! {e}`; caller row `{AI, e}`).
- F2: reject the currently accepted width-subsumption class for this release; rewrite the two ceiling subtests into strict-compatible spellings while retaining their authority assertion.
- F3: breaking soundness posture, minor release, explicit migration note, no opt-out.

**There are no OPEN design-freeze human-decision gates. Do not reopen these rulings.** The design records that quorum trigger 1 fired and the requester handled review; no quorum artifact was produced there. Unlike the SMT/runtime-parity docs, this doc does not state an additional mandatory before-planning quorum step. This draft preserves the session's required plan approval and explicit execution gate: **do not implement until the user approves this concrete plan and says to execute the sprint.**

If a newly discovered compatibility change needs a genuinely new ruling, cite the new decision and **STOP: commit current work, report `BLOCKED: <decision>`, and end the run**. A ruled F1/F2/F3 is never such a blocker.

## Current implementation and velocity

On the pinned base, `internal/types/row_unification.go:97–121` binds an open tail without rejecting open-side labels absent from the closed side. A scratch build accepted `pure viaMapPure(g: ... ! {IO}) = map(g, xs)` and then printed `leak` at runtime. The two callback builtins in `internal/builtins/ai_step.go:463,607` still close the callback row, while `std/ai.ail:302,318` declares IO callbacks under outer AI only.

Recent relevant work: the numeric-literal soundness fix `4eeecfe29` landed 2026-10-28 with 476 insertions/23 deletions across 18 files; this includes 56 pipeline, 64 elaborator and 89 CLI parity test lines. It is one observation, not a robust LOC/day estimate. The velocity helper exited 141 after its piped history listing and did not produce usable averages. This plan therefore uses the design's 2–3 day estimate, 620 authored lines total (mostly tests), approximately 248 lines/day, with separate corpus/review allowance. Do not inflate delivery claims from the helper's historical changelog matches.

## Registry reuse audit

Read-only searches: `ailang pkg search effect` (8 results) and `ailang pkg search streaming` (3). Inspected `pkg info` and `pkg docs` for `sunholo/ollama_stream@0.1.0` and `sunholo/gemini_live@0.5.1`. The former is a Stream-effect NDJSON client; the latter pure protocol helpers. Neither can change the compiler's row equation or the std/ai builtin signature.

All four milestones use `action: none`: this is a compiler/stdlib soundness invariant, not a package capability to depend on or reimplement. No registry dependency, package publication, or downstream agent dispatch is required.

## Dependencies and coordination

- **Internal order:** M1 before M2; M2 before M3; M3 before M4. M1 can be reviewed/committed independently; the guard must never land before the std/ai migration.
- **PR #1790 (pure rows / iface purity):** not a semantic prerequisite. It touches all `.stdlib-golden` files, including `ai.json`; whichever lands second regenerates from the combined tree, preserving its purity/hash decisions. Prefer #1790 first when both are ready, so this sprint validates the combined final interface. Do not implement its remaining M4 here.
- **PR #1963 (#1718/#1826):** independent production files. Either order works; this sprint must rerun the corpus on the combined tree if it lands second. #1963 owns nested function-value charging and compatibility-tail escape; do not edit its walker/application changes.
- **Unowned check:** inventory shows no #1958 implementation PR or coordinator task. Recheck `gh pr list` and live task titles immediately before starting; stop assigning the same files to another agent if ownership appears.
- **Release:** design targets the next minor. Current checkout may have advanced to v0.55.0 during the sweep; parent/release owner must assign the actual release name without turning this sprint into a release operation.

## Milestones

### M1: Propagate std/ai streaming callback effects (~180 LOC)

**Goal:** Make the two std wrappers and builtin types carry the callback row accurately before enabling the stricter unifier.
**Estimate:** 35 implementation/docs/example + 145 test lines; 0.5 day, one 60–90 minute executor run.
**Dependencies:** none.

**Files:** `internal/builtins/ai_step.go`, `std/ai.ail`, `internal/pipeline/testdata/builtin_types.golden`, `.stdlib-golden/ai.{json,sha256}`; companion tests in `internal/builtins/ai_stream_effect_rows_test.go` and `internal/pipeline/effect_row_open_closed_test.go`; new example `examples/runnable/ai_streaming_effect_rows.ail` and manifest entry.

- [x] Give both callback function types `RowTail("e")`; give the enclosing builtin function the same tail plus AI, following `_terminal_withTerminal`'s established builder pattern.
- [x] Both wrappers accept `on_chunk: StreamChunk -> () ! {e}` and return under `! {AI, e}`; update misleading callback-effect comments.
- [x] Pure callback in an AI-only caller checks; IO callback in an AI-only caller is rejected; the same callback in AI+IO checks. Test both recorded and non-recorded APIs.
- [x] Refresh builtin/stdlib goldens using existing commands, never hand-edit hashes or resurrect old purity metadata.
- [x] Type-check example and run it with the existing deterministic AI stub where supported; no live paid provider call is needed to prove row propagation.
- [x] Focused builtin/pipeline tests and `make test-changed` pass; commit M1 before M2.

**Example syntax checklist:**

| Module | contracts | effects | inline tests |
|---|---|---|---|
| `ai_streaming_effect_rows.ail` | skip: provider-produced chunks/results are not an SMT-verifiable pure computation | include: pure `quiet(StreamChunk)->()` callback, exported AI-only call; explicit AI+IO renderer example | skip: effectful streaming entry is covered by deterministic Go compile/stub integration tests, not pure inline rows |

### M2: Enforce the symmetric open/closed row guard (~230 LOC)

**Goal:** An open row's own extra labels cannot vanish when unifying against a closed row.
**Estimate:** 30 production + 200 unit/integration test lines; 0.75 day, one 60–90 minute executor run.
**Dependencies:** M1.

**Files:** `internal/types/row_unification.go`, companion `internal/types/row_unification_open_closed_test.go`, `internal/pipeline/effect_row_open_closed_test.go`, and `internal/pipeline/effect_ceiling_reachable_test.go`.

- [x] With r1 open/r2 closed, error on nonempty only1 before binding; symmetrically error on only2 with r1 closed/r2 open. Diagnostic names the disallowed labels and slot row.
- [x] Matrix covers both orientations and EffectRow/RecordRow kinds, mixed explicit row variables, permitted extra closed labels, ordinary closed/closed, unchanged open/open, and previously valid stdlib regression cases.
- [x] R1–R4 reject annotated callback parameter, record field, filter/foldl/any, and mixed `{IO,e}` rows through pure slots.
- [x] Re-pin exactly the two approved width-class ceiling subtests using a named hook with explicit slot effects or an annotated lambda; preserve `max=[Stream]` and the assertion that own stored/callback types do not reach Process.
- [x] Ordinary pure map and effectful mapE row-polymorphic callers remain accepted; directly invoking IO from pure remains rejected.
- [x] Focused types/pipeline tests plus `make test-changed` pass; commit M2.

### M3: Pin two-engine rejection and mutation evidence (~140 LOC)

**Goal:** Prevent a static regression from being disguised by a runtime capability backstop.
**Estimate:** 140 test lines; 0.5 day, one 60–90 minute executor run.
**Dependencies:** M2.

**Files:** companion `cmd/ailang/effect_row_soundness_test.go` or existing command fixture harness (reuse, do not duplicate build helpers), pipeline fixtures, and validation evidence next to this plan.

- [x] On evaluator and `--bytecode`, each R1–R5 negative program fails before its side-effect marker is printed; matching declared-effect controls run and print exactly once.
- [x] Check both `_ai_step_with_stream` and `_ai_step_with_stream_recorded` wrapper acceptance via deterministic stub/compile tests, including pure renderer and AI+IO renderer.
- [x] In a scratch copy only, remove both guards with M1 retained: new leak and unit matrix tests turn red. Restore guards and remove M1: std/ai migration test turns red. Record exact SHAs and commands.
- [x] Existing direct-call and pure callback controls remain green; no new silent fallback is introduced.
- [x] Focused command/pipeline/type suites and `make test-core` pass; commit M3.

### M4: Validate the combined corpus, goldens and migration note (~70 LOC)

**Goal:** Land a documented soundness correction with measured compatibility, preserving the existing PRs' work.
**Estimate:** 70 docs/evidence lines; 0.75 day, one 60–90 minute executor run plus bounded corpus verification. Split/resume evidence work if external corpora exceed this window; bank completed work first.
**Dependencies:** M3; refresh merge base and check intervening #1790/#1963 changes.

**Files:** `changelogs/unreleased/2026-10-29-effect-row-open-closed-soundness.md` (or actual execution date), validation doc beside plan, std/ai comments/example documentation, sprint JSON.

- [x] Base-versus-fixed checks use fresh archived source, `AILANG_NO_CACHE=1`, matching stdlib, and pinned external SHAs. Record 0 status flips and 0 first-diagnostic changes for the current examples/std corpus and available ailang-packages, ailang-parse, motoko/world roots. Never claim unavailable external results as passed.
- [x] Recheck #1790 and #1963 merge status; regenerate goldens and rerun affected tests on the actual combined tree if either landed. Do not copy old `.stdlib-golden/ai.json` over newer metadata.
- [x] Changelog fragment opens `### ` and clearly states Breaking — soundness, mapE/row-polymorphic migration, stricter width-class limitation, and AI callback row behavior.
- [x] Run `make test-changed`, `make lint`, `make check-file-sizes`, `make check-boundaries`, `make freeze-stdlib`, `make check-changelog`, `make verify-examples`, and `tools/verify-stdlib.sh` as required by the design; all commands run unpiped and results banked. CI runs the full suite; do not run `make test` in the executor.
- [ ] Mark only measured milestones passed; invoke sprint-evaluator after implementation and report any remaining corpus/release limits.

## File-size and scope controls

Pinned base sizes: `row_unification.go` 309, `ai_step.go` 631, `std/ai.ail` 425. Add tests to new companion files rather than growing already large implementation files. No parser, iface schema, cache-key, effect budget, runtime authority, std/list purity, or unifier consolidation change is authorized by this plan. Do not touch the shared checkout's parked benchmark JSON edits.

## Day-by-day schedule

Day 1: M1, then M2 with matrix and positive controls. Day 2: M3 two-engine regressions and scratch mutations; start corpus comparisons. Day 3 (half-day equivalent): finish M4 combined-tree checks, documentation, and evaluator handoff. The 2.5-day estimate includes review/corpus allowance; run-level commits let the attended session use shorter remaining windows safely.

## Success metrics

R1–R5 fail before printing on both engines; positive controls still work. No unapproved human decision remains open. Exactly the two approved ceiling spellings are intentionally changed. The current corpus has no unexplained acceptance or diagnostic flips. Builtin and stdlib goldens are generated from the final tree. Focused checks and architectural boundaries pass, with an explicit record of any external corpus unavailable to this machine.

## Draft-generation notes

The standard JSON generator imports/reads agent messages as a side effect; this read-only subtask was explicitly forbidden to send/ack messages. Therefore this review-only scratch JSON was populated directly using its documented schema, with real milestones and registry decisions. No repository plan/JSON, progress checkpoint, coordinator message or execution was created. Move/recreate these artifacts in an isolated approved branch only after the parent obtains the session gate.

## Execution amendment approved 2026-10-29: motoko width-class migration

M1–M3 are implemented and pass local gates. The original M4 external compatibility requirement caught five previously accepted motoko core entries (60/60 base, 55/60 fixed), all the same already-approved F2 width class through `play_chunks`'s IO+Trace callback slot. The exact one-line migration in the companion patch narrows only that slot to IO and preserves the helper's outer IO+Trace row and Ports ABI. Fresh migrated checks restore 60/60 on both arms, zero flips. See the companion validation report for the original failure evidence, pinned SHA, delivery branch, and duplicate-work check.

**Amendment approved by Mark, attended:** deliver the exact one-line consumer annotation migration on `sunholo-voight-kampff/motoko_agent`, base `sunholo/main-dst-20261027` at `835c738b8ca50ed38f36c117058847c816db32b8`, then update the AILANG Docker source pin and provenance. Preserve the live rig checkout. Do not mark M4 complete until actual fork delivery, source pin, and combined-tree gates are recorded, then hand off for independent evaluation. Do not weaken the compiler guard, implement width subsumption, or discard the original 60→55 measurement.

## Historical integration checkpoint: M4 required a new decision

After #1963 merged, actual integration 970a46694 passed the focused regressions,
MCP OAuth, unchanged 49 interface goldens, lint/structure and corrected combined
manifest gate. The actual migrated motoko core is 55/60 through fixture_rt's nested
hand-off, independently reproduced on #1963-only. The exact attempted four pure
`! {}` field annotations fail unchanged. This follows the ratified conservative
parameter-wide architecture; changing it or adding explicit-empty callback contract
semantics is outside this plan's existing approval. See the new focused amendment
at `design_docs/planned/v0_56_0/m-effect-explicit-empty-callback-contract.md`.
At that checkpoint all its human decisions remained OPEN. M4 was intentionally
unpassed; no new compiler or consumer semantics had been implemented. Only three manifest aggregate fields were
reconciled for the two independently added examples, with verify-examples green.


## Final approved amendment and delivery checkpoint

Mark subsequently approved option A and explicitly executed its four-milestone
sprint. Closed-empty conversion, persistence/cache identity, both-engine controls,
new prompt 11 and the runnable example are implemented with measured gates.
The exact conditional four-field consumer migration restores fresh no-cache 60/60
and actual 60 core / 9 extension checks, then was independently reviewed and normally
merged as fork PR #5 at `083ef43726981099a0bad4fedda18fc72105d6fa`. Docker and MOTOKO §15
pin that exact tested tree. Full changed/core and structural/interface/example
checks pass. Original failure stages remain preserved in the companion reports.
M4 implementation is complete; independent evaluator acceptance remains pending.
