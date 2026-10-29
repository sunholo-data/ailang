# M-EFFECT-ROW-OPEN-CLOSED-SOUNDNESS validation

**Archive status:** Accepted, implemented and **unreleased**, target v0.56.0. Root PASS 97/100 on `8de735cd9adfd4dcfe879686e1d7ac80d7a6f1af`; merged `80793dbcf42423027be4bab1a0088da54e2323d8`. [Independent report](m-effect-row-open-closed-soundness-evaluation.md).

**Historical record:** The chronological execution text below is preserved, including its failed 55/60 measurements, failed four-field hypothesis and then-pending gates. Those statements describe their measured checkpoints, not the current accepted state.

Execution authorized by Mark (“Execute sprint #1958”), 2026-10-29. Independent evaluation and merge remain pending.

## Pinned implementation and controls

- Pre-sprint source: `c04a44780f32b6af98905a46191c104b1d5b32ce`, archived with `git archive`, built separately as `1958-base`.
- M1 `1f46a92ec`: row-polymorphic callbacks and matching enclosing rows for both streaming APIs; generated builtin and stdlib goldens.
- M2 `9bdf1ddf9`: symmetric open/closed guards; approved explicit-lambda spellings for the two ceiling precision controls.
- M3 `034ffa2e0`: before-effects rejection with evaluator and `--bytecode`, plus deterministic positive controls.
- Syntax reference loaded before editing AILANG: embedded teaching prompt v0.16.8. Example contracts and inline rows are skipped because streaming entries are effectful; its explicit AI and AI+IO rows are demonstrated and Go integration verifies both APIs.

All corpus arms used `AILANG_NO_CACHE=1` and `AILANG_STDLIB_PATH` pointing to the arm's matching source tree. CLI checks used relative source paths from the matching root. Diagnostics compared the first `Error:`/`PAR_` line after normalizing only the arm's root/stdlib prefix; status comparisons use the actual exit code. Version mismatch warnings are excluded from the first-error comparison. The binaries are identified by their source SHA, not by the development version stamp.

## Failing-first tests and mutation evidence

| Experiment | Expected | Observed |
|---|---|---|
| M1 tests before signature change | fail | Both APIs rejected a pure callback and accepted an IO callback under AI alone; builtin rows were closed |
| M1 tests after signature change | pass | Both APIs accept pure callbacks, reject uncharged IO, accept AI+IO callers |
| M2 matrix before guards | fail | IO disappears in both orientations and kinds; substituted tail also leaks |
| M2 callback tests before guards | fail | map/filter/any/foldl, record-field and mixed-row callback programs accepted |
| M2/M3 tests after guards | pass | Effect diagnostics name IO; both engine selections reject before callback output |
| Scratch `9bdf1ddf9` minus both guards, keeping M1 | fail | 12 own-label matrix subtests plus substituted-tail test, 6 pipeline callback cases and 5 CLI leak cases fail; CLI stdout shows `ROW1958_SIDE_EFFECT` |
| Scratch `9bdf1ddf9` minus M1's two signature files, keeping guards | fail | All 6 pipeline streaming cases and both builtin type cases fail; std/ai itself no longer type-checks |

Mutations used separate archives, never worktree file checkout/revert. Commands:

```bash
go test ./internal/pipeline ./internal/builtins -run 'TestAIStream(CallbackEffectRows|BuiltinCallbackEffects)$' -count=1
go test ./internal/types ./internal/pipeline -run 'Test(RowUnification_|EffectRowOpenClosedAnnotatedCallbacks|AIStreamCallbackEffectRows|EffectCeiling_OwnTypesAndCallbacksNotCharged)' -count=1
go test ./cmd/ailang -run '^TestEffectRowGuard' -count=1
go test ./internal/types -run 'TestRowUnification_OpenClosed(OwnLabels|AfterTailSubstitution)$' -count=20
```

The positive-control set includes pure map, effectful mapE lambda, row-polymorphic mapE parameter, declared IO traversal with one observed print, and pure AI callbacks through both streaming APIs. The example `ai_streaming_effect_rows` type-checks, appears first in `ailang examples search "streaming callback"`, and prints `pure callbacks accepted` under `--ai-stub`.

The F2 named-function alternative was tried and discarded: a separately declared `! {Stream, Process}` function correctly violates the fixture's `max=[Stream]` ceiling in its own declaration. The other explicitly approved F2 alternative, an annotated lambda, keeps the authority assertion and passes. Only those two approved fixture spellings changed.

## Corpus differential

| Corpus | Source SHA | Units | Base pass | Fixed pass | Status flips | First-error changes |
|---|---|---:|---:|---:|---:|---:|
| examples + std | c04a44780 | 525 files | 442 | 442 | 0 | 0 |
| ailang-packages | e998928e3dbab4661b47ae94e3e74c593f37e121 | 45 package roots | 43 | 43 | 0 | 0 |
| ailang-parse | 0931946850e23a6897df783731ab74b0b1d43b88 | 2 package roots | 1 | 1 | 0 | 0 |
| ailang-world | df5a0671dadd567ece879fe89afcc018c2deaa23 | 1 package root | 1 | 1 | 0 | 0 |
| motoko production roots | 835c738b8ca50ed38f36c117058847c816db32b8 | 25 package roots | 21 | 21 | 0 | 0 |

External source trees were archived from the listed commits; uncommitted downstream work was untouched. Packages were checked with `ailang check --package .` from each root; historical motoko `.agent` evidence and tool-generator fixtures were excluded. Existing failures include motoko-ext-a2a, the invalid motoko-ext-typefix-agent manifest, parse's docs package, and three motoko packages. The broad motoko root scan timed out at 120 seconds on both arms; a dedicated sweep of the Makefile's `src/core/*.ail` selection exposed the approved width class, documented below. A timeout is not counted as a pass.

## Required gates

- `make test-changed`: PASS at M1/M2/M3; final M3 changed-package phase includes cmd/ailang (193.594s), builtins, pipeline and types, followed by all core packages.
- `make lint`: PASS after each production/test milestone, zero issues.
- `make check-file-sizes`: PASS; companion files keep production modules below 800 lines.
- `make check-boundaries`: PASS.
- `make verify-examples`: PASS on migrated/guarded tree; manifest check reports zero drift/missing files.
- `make freeze-stdlib`: generated only the approved `ai` interface change; subsequent regeneration has no diff.
- `tools/verify-stdlib.sh`: PASS, all 49 interfaces stable.
- `make check-changelog`: PASS.
- `git diff --check`: PASS.

The full repository suite is left to PR CI, per the executor workflow. No paid provider request was made.

## Adjacent work and remaining boundaries

PR #1790 (pure rows/iface purity) and PR #1963 (nested latent callback charging/#1826) remain separate. Neither was merged when this validation was performed. #1958 changes only the row unifier and std/ai signatures; it does not incorporate the nested walker or alter purity hash rules. Whichever lands second must regenerate shared ai goldens and repeat the combined-tree corpus check. Width subsumption is intentionally deferred under the approved F2 ruling.

## Motoko width-class discovery and approved plan amendment

The broad root timeout was not accepted as compatibility evidence. Running the actual Makefile core selection found **60/60 base versus 55/60 fixed**. The five affected entry modules are `agent_loop_v2.ail`, `dst_execution.ail`, `rpc.ail`, `session.ail`, and `supervisor.ail`. Their first diagnostic is the same call at `src/core/test/stub_step.ail:405`: the model-step callback row carries Trace while the published callback slot is closed at IO.

This is the already approved F2 width class: `play_chunks` advertises an IO+Trace callback slot, but the Ports ABI supplies an IO-only callback. The old unifier admits that narrower value by silently absorbing the wider slot label. The strict equality rule intentionally rejects implicit widening. The helper body merely invokes `on_chunk` and recurses; it does not need Trace in the callback slot.

A **scratch-only one-line migration**, retaining the existing outer function row and the entire Ports ABI, restores compatibility:

```diff
-func play_chunks(chunks: [StreamChunk], on_chunk: (StreamChunk) -> () ! {IO, Trace}) -> () ! {IO, Trace} {
+func play_chunks(chunks: [StreamChunk], on_chunk: (StreamChunk) -> () ! {IO}) -> () ! {IO, Trace} {
```

Fresh checks of the same 60 core files after this migration: **base 60/60, fixed 60/60, zero status flips and zero first-error changes**. The original 60→55 evidence remains recorded, not reclassified as a pass. No compiler guard was relaxed. Exact patch context is committed beside this report as `m-effect-row-open-closed-soundness-motoko-migration.patch`.

**Amendment approved by Mark, attended (2026-10-29):** deliver this exact consumer annotation fix on our pinned fork, then update AILANG's Docker source pin and provenance before merging/enabling #1958. The compiler's M4 compatibility gate remains pending until actual consumer delivery and combined-tree validation are complete. This is a migration of the already approved class, not an implementation of width subsumption.

Read-only live consumer check (2026-10-29): `/Users/voightkampff/dev/mk-20261027`, clean branch `sunholo/main-dst-20261027` tracking `fork/sunholo/main-dst-20261027`, HEAD `835c738b8ca50ed38f36c117058847c816db32b8`. Origin is `arniwesth/motoko_agent`; fork is `sunholo-voight-kampff/motoko_agent`. Intended delivery base is the fork's pinned `sunholo/main-dst-20261027` branch; any separate upstream contribution must start from upstream main to avoid including the carried fork commits. Open-PR inventory: upstream 18, fork 0; no play_chunks/callback-row/Trace migration match in titles or bodies. At this classification checkpoint the live consumer checkout was read only; all changed sources were scratch archives.

## Consumer delivery in progress (2026-10-29)

The isolated `/tmp/ailang-backlog-20261029/motoko-1958` worktree starts at the pinned fork SHA above. Commit `c3f4c85eec681d8d5d4787da4d9e56c22c0a2457`, PR `sunholo-voight-kampff/motoko_agent#4`, changes exactly one callback annotation line. Bot identity was verified as `sunholo-voight-kampff`; base is `sunholo/main-dst-20261027`. No upstream writes or live-checkout edits occurred. The upstream path overlap #21 changes imports and `on_pre_step`, not `play_chunks`; the fork had no other open PR.

Fresh compiler `9eb1ebaed78a0ecf6b232b5b329633ed11438358` with matching stdlib passed full `gmake check_core`: **60/60 core files, 9/9 extension boots**, plus all prerequisite runtime guards. Log: `/tmp/ailang-backlog-20261029/1958-motoko-delivery-check-core.log`. A supplemental no-cache sweep passed 53 files and reached the 90-second deadline on seven large entries without a type diagnostic; all seven passed at concurrency two and a 180-second deadline (25–122 seconds). The combined fresh result is **60/60**. Original timeouts remain in `1958-motoko-delivery-core-fresh.json`; successful bounded retries are in `1958-motoko-delivery-core-retry.json`. Final fork delivery and combined-tree checkpoint results are recorded below.

The Docker source pin and MOTOKO.md provenance now name fork merge `af1d56513d3ae314bd831fd149c7b6f2503a361c`, source-tree identical to tested migration commit `c3f4c85eec681d8d5d4787da4d9e56c22c0a2457`. PR #4 merged normally without an admin override or policy change. No cloud image build/deployment or rig switch was performed. M4 remains pending the final standalone branch rebase after #1963 and independent evaluation.

Release posture F3 now targets **v0.56.0**, the next minor after v0.55.0 was tagged during execution. No version bump, release, or deployment is performed here. Historical planned paths remain stable.

## Consumer CI baseline and normal delivery

Fork CI independently passed core **60/60** and extensions **9/9** using its pinned
AILANG `v0.52.5` (tag object `76a23634453f4dcf8c5fe9bb7766ffe089265569`, source commit
`7200786dcd21a61437cb76382ac31f7fa4dbe11d`). Its later skills-refusal step reported
47 fixture failures. A fresh no-cache build of that exact old compiler with its
matching stdlib reproduces the **same** error on pristine motoko base `835c738b`
and migrated head `c3f4c85e`: `scripts/verify_skills_startup.ail:121`,
`HeaderEntry.boot.cost_rates`, expected two fields but got four including
`cache_read_per_1m_millicents` and `cache_write_per_1m_millicents`. Both return exit 1.
The affected skills startup cases all use this shared entry point. The migration
changes none of its pricing code. Logs: `1958-motoko-ci-base-pricing.log`,
`1958-motoko-ci-migrated-pricing.log`, `1958-motoko-fork-ci-core-failed.log`.

CONTRIBUTING.md explicitly requires `make check_core`, passed with the intended
compiler. Mark authorized normal fork-only delivery after this baseline proof;
[PR #4 evidence](https://github.com/sunholo-voight-kampff/motoko_agent/pull/4#issuecomment-6491761805)
records the inherited CI failure. No `--admin`, required-check bypass, CI/policy edit,
pricing edit, or upstream write occurred. The broader CI is not claimed all green.
The fork base now resolves to full merge SHA `af1d56513d3ae314bd831fd149c7b6f2503a361c`.

## Current-dev and disposable combined-tree checks

Own branch rebased cleanly onto `origin/dev` `c2944389afefe99b592ffba387aae37903044126`.
Standalone checkpoint `05ce2e4f4d409100c77e60360a4bf1e9df7212ee` passed focused
row/builtin/streaming/ceiling/two-engine tests across types, builtins, pipeline, CLI
(0.541s / 0.872s / 6.623s / 12.615s), log `1958-currentdev-focused.log`.

A **separate, unpushed** disposable branch/worktree combined that checkpoint with
#1963 head `86d3a5f8e946e12967c38b446e5fb854a884a42e`, creating integration commit
`4358f0542af404bd63d782059b15b5507a8ae1df`. No #1963 commits entered the standalone PR.
The combined tree passed:

- Row guard, streaming callback rows, ceiling authority, and evaluator/bytecode
  rejection before side-effect output: types/builtins/pipeline/CLI green
  (0.395s / 0.336s / 7.330s / 14.242s), `1958-combined-focused.log`.
- Nested callbacks, publication, depth/cycle/union invariants and nested rejection
  before output: types/pipeline green (0.280s / 6.463s), `1958-combined-nested.log`.
- Fresh no-cache check of `examples/runnable/serve_api_mcp_oauth.ail`: green,
  `1958-combined-mcp.log`.
- `tools/verify-stdlib.sh`: all **49** interfaces stable; generated `std/ai` JSON
  byte-identical to the committed golden; no golden diff. Logs
  `1958-combined-stdlib.log` and `1958-combined-ai-iface.json`.

This integration proof does not waive #1963's formal CI or merge requirements.
After #1963 lands, rebase the actual standalone branch and repeat only checks
materially affected by any new delta. The final integrated commit and independent
review are pending; M4 remains unpassed at this checkpoint.

## Actual combined integration: compatibility gate FAILED (2026-10-29)

Actual branch rebased onto the normal #1963 merge `87f846980c8b6fc601173be427b1364968559488`,
creating source checkpoint `970a4669470566163cba09892a351e639b6de60c`. Source diff versus
temporary integration `4358f0542af404bd63d782059b15b5507a8ae1df` is empty across
types, builtins, pipeline, CLI, std, examples and goldens. Focused actual tests pass
(types 0.532s, builtins 0.628s, pipeline 4.530s, CLI 9.980s); MCP OAuth check, lint,
sizes, boundaries and changelog pass. `make freeze-stdlib` regenerates 49 interfaces
with zero golden diff; all 49 verify stable and std/ai JSON is byte-identical.

The final actual no-cache motoko core sweep deliberately repeated the consumer
compatibility gate with the combined compiler. **55/60**, five actual type failures,
not timeouts. All five fail in the newly exercised `fixture_rt`/`fixture_hooks`
pure factory hand-off in `src/core/test/ext_fixture.ail`, naming eleven callback
effects. This is separate from the delivered `play_chunks` migration. The original
60/55 width failure, successful migrated row-only 60/60, and new combined 55/60 remain
distinct measurements. Log/JSON: `1958-final-motoko-core*`.

Four-arm exact-source reductions prove base and row-only accept this factory;
nested-only and combined reject. Direct literal storage stays accepted; implicit
nested hand-off charging is explicitly ratified by #1963. Mixed record behavior
matches that parameter-wide mask plus row-union architecture, so it is not being
silently reclassified as an AC4 exemption. The proposed scratch four-line annotation
of pure fields with `! {}` **still fails**. Both conversion/source inspection and
measured IO-to-empty-slot acceptance show a separate explicit-empty function-type
representation gap. No new compiler semantics or consumer adaptation was delivered.

The focused [new design/amendment](m-effect-explicit-empty-callback-contract.md)
and its evidence JSON preserve exact sources, pins, all outcomes, options and OPEN
human decisions. **M4 remains unpassed.** No final sprint-evaluator PASS is asserted.

Combined example entries also revealed stale aggregate metadata: recorded 248/240
versus actual 249/241. Exactly total, working and coverage fields were corrected.
The initial red gate is retained in `1958-final-examples.log`; the corrected
`make verify-examples` passes, with 244 module entries checked and no drift/missing
files (`1958-final-examples-fixed.log`). No example body or unrelated manifest
entry changed in this correction.


## Final approved explicit-empty compatibility resolution

The prior failures above remain exact historical measurements. Mark approved and
executed the separate explicit-empty callback sprint. Its final compiler checkpoint
`9ab45c98e` on actual dev `9dc689091` passes the original row/streaming/authority controls
and full changed/core gate, plus 49 stable interfaces with no new golden diff.
The exact four-field migration now restores fresh no-cache 60/60 and actual
`gmake check_core` 60 core / 9 extensions. Three initial 180s deadlines pass on serial
retries 127.07/79.44/67.97s; initial results are retained. The reviewed fork PR #5 merged
normally at `083ef43726981099a0bad4fedda18fc72105d6fa` with tested-tree equivalence.
The Docker source pin and MOTOKO §15 provenance select that SHA; live rig/upstream
remain untouched, and old v0.52.5 CI pricing failure remains separately disclosed.
See the explicit-empty validation and compatibility JSON for exact corpus/pins,
cache/engine proof and final checks. M4 implementation is ready for root's final
independent evaluation; no final PASS or full CI success is asserted here.


## Final independent acceptance — 2026-10-29

Root accepted this sprint at **97/100**; the original report and deductions are archived unchanged. The final test-only review repair raises actual Sonar new coverage from 64.8% to 93.5% without threshold/exclusion/policy changes. Actual consumer merge `083ef43726981099a0bad4fedda18fc72105d6fa` passes 60 core files and 9 extension boots; all original failures and deadline retries remain separate historical measurements. [Machine evaluation](../../../.ailang/state/evaluations/eval_M-EFFECT-ROW-OPEN-CLOSED-SOUNDNESS_round_1.json). No release/deployment or completion of deferred Phase 2 is claimed.
