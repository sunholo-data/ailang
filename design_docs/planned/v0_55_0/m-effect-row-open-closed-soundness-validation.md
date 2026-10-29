# M-EFFECT-ROW-OPEN-CLOSED-SOUNDNESS validation

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

## Motoko width-class discovery and proposed plan amendment

The broad root timeout was not accepted as compatibility evidence. Running the actual Makefile core selection found **60/60 base versus 55/60 fixed**. The five affected entry modules are `agent_loop_v2.ail`, `dst_execution.ail`, `rpc.ail`, `session.ail`, and `supervisor.ail`. Their first diagnostic is the same call at `src/core/test/stub_step.ail:405`: the model-step callback row carries Trace while the published callback slot is closed at IO.

This is the already approved F2 width class: `play_chunks` advertises an IO+Trace callback slot, but the Ports ABI supplies an IO-only callback. The old unifier admits that narrower value by silently absorbing the wider slot label. The strict equality rule intentionally rejects implicit widening. The helper body merely invokes `on_chunk` and recurses; it does not need Trace in the callback slot.

A **scratch-only one-line migration**, retaining the existing outer function row and the entire Ports ABI, restores compatibility:

```diff
-func play_chunks(chunks: [StreamChunk], on_chunk: (StreamChunk) -> () ! {IO, Trace}) -> () ! {IO, Trace} {
+func play_chunks(chunks: [StreamChunk], on_chunk: (StreamChunk) -> () ! {IO}) -> () ! {IO, Trace} {
```

Fresh checks of the same 60 core files after this migration: **base 60/60, fixed 60/60, zero status flips and zero first-error changes**. The original 60→55 evidence remains recorded, not reclassified as a pass. No compiler guard was relaxed. Exact patch context is committed beside this report as `m-effect-row-open-closed-soundness-motoko-migration.patch`.

**Proposed amendment, awaiting Mark's approval:** apply this exact consumer annotation fix before merging/enabling #1958 against the pinned motoko consumer. The compiler's M4 compatibility gate remains pending until that amendment is authorized and the consumer delivery is coordinated. This is a migration of the already approved class, not an implementation of width subsumption.

Read-only live consumer check (2026-10-29): `/Users/voightkampff/dev/mk-20261027`, clean branch `sunholo/main-dst-20261027` tracking `fork/sunholo/main-dst-20261027`, HEAD `835c738b8ca50ed38f36c117058847c816db32b8`. Origin is `arniwesth/motoko_agent`; fork is `sunholo-voight-kampff/motoko_agent`. Intended delivery base is the fork's pinned `sunholo/main-dst-20261027` branch; any separate upstream contribution must start from upstream main to avoid including the carried fork commits. Open-PR inventory: upstream 18, fork 0; no play_chunks/callback-row/Trace migration match in titles or bodies. The live consumer checkout was read only; all changed sources are scratch archives.
