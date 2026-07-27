# M-EFFECT-REPLAY-SUBSUMPTION: Validate-Path Effect-Mode Subsumption

**Status**: Ready for sprint planner — Mark's 2026-07-27 ratification is recorded in the
`design_docs/v1-mission.md` queue row and commit `4d32c71bb` message body
**Target**: v1.0.0, mission clause 4 (orchestration flagship), **SHARED GATE**
**Priority**: P0 — blocks runnable seeded/crypto Rand and the Clock/Net/FS mode ports
**Estimated**: 18–22 hours (2.5–3 days)
**Dependency / Parent**:
[M-EFFECT-REPLAY-CONTRACTS](../../implemented/v0_30_0/m-effect-replay-contracts.md)
(landed partial)

## Framing

> The parent sprint shipped replay-contract labels and mode-aware Rand dispatch, but the
> validate pass prevents a function declared `! {Rand[mode=seeded]}` or
> `! {Rand[mode=crypto]}` from calling the bare `! {Rand}` wrappers in `std/rand`.
> Mark's ratified decision is narrow: an explicitly declared mode subsumes an os/bare
> requirement in the `SubsumeEffectRows` validation path. Function-value effect-row
> unification remains invariant.

**Verified clean baseline (2026-07-27):** The worktree was pinned to
`4d32c71bb04e5dcbfd46920d36fee0a04812dcc8`, with this uncommitted document temporarily
removed and the staged binary deleted. `git status --porcelain` produced no output;
`git rev-parse HEAD` returned the full SHA above; and `git describe --tags --dirty` returned
`v0.30.0-201-g4d32c71bb` with no `-dirty` suffix. `make build` exited 0, and the resulting
`bin/ailang --version` reported:

```text
AILANG v0.30.0-201-g4d32c71bb
Commit: 4d32c71
Full:   4d32c71bb04e5dcbfd46920d36fee0a04812dcc8
```

There was no freshness warning, and a post-build `git status --porcelain` again produced no
output. The fixture contents and exact unpiped check commands are recorded in
[Appendix A](#appendix-a-clean-baseline-fixtures-and-commands). The clean results were
identical to the earlier exploratory results in every case, so the problem statement and
semantic matrix are unchanged.

## Correction: Modes Are Not Currently Invariant Across Calls

The parent document and mission queue say that `SubsumeEffectRows` makes effect modes invariant
on the runnable `.ail` path. That premise is refuted.

The following source-valid fixtures were live-checked:

| Caller declaration | Locally declared callee | Current result |
|---|---|---|
| no effect annotation | `Rand[mode=seeded]` | exit 1, `Missing effects: Rand` (`c4.ail`) |
| bare `Rand` | `Rand[mode=seeded]` | exit 0 (`c3.ail`) |
| explicit `Rand[mode=os]` | `Rand[mode=seeded]` | exit 0 (`c7.ail`) |
| bare `Rand` | `Rand[mode=crypto]` | exit 0 (`c8.ail`) |
| `Rand[mode=seeded]` | bare `Rand`, with a real `rand_int` call | exit 1, empty `Missing effects:` (`c6.ail`) |

What is actually true is directional and partly accidental:

1. A missing effect label is enforced: `c4.ail` proves the callee's `Rand` requirement
   propagates.
2. A non-default mode on a **locally declared callee** does not propagate. `ValidateEffects`
   constructs `map[string][]string` with `ast.EffectNames` at
   `internal/pipeline/validate_effects.go:109-114`; `EffectNames` deliberately returns names
   only at `internal/ast/ast.go:116-123`. A local call reconstructs its row from those labels
   at `internal/pipeline/validate_effects.go:362-371`.
3. That reconstructed bare `Rand` normalises to `mode=os` through `effectiveParamsOf` and
   `DefaultModeFor` at `internal/types/effects.go:206-227`.
4. Mode comparison therefore currently rejects mainly when the **enclosing declaration** is
   non-default and the body is observed as default. It cannot enforce a non-default
   locally-declared callee requirement.

This is mode-blindness across the local call graph, not invariant mode checking.

## Problem Statement

There are three coupled defects on the same validation path:

1. **Blocked dispatch.** The verified `blocker.ail` (the `subsum_repro` case) and `c2.ail`
   fail with exit 1 when seeded/crypto declarations call `std/rand.rand_int`; explicit os
   (`c1.ail`) succeeds.
   Consequently the parent sprint's Go-level seeded/crypto implementations are unreachable
   from the intended `.ail` surface.
2. **Empty diagnostic.** The lambda check rejects at
   `internal/pipeline/validate_effects.go:158-165`, then uses label-only
   `EffectRowDifference` (`internal/types/effects.go:651-665`). Equal labels plus unequal
   modes produce `Missing effects:` with no item. The top-level formatter has the related
   fallback “no specific missing effects identified” at
   `internal/pipeline/validate_effects.go:539-546`.
3. **Mode-blind local calls.** Parameter erasure at
   `internal/pipeline/validate_effects.go:109-114,362-371` currently accepts an os caller of
   an explicitly seeded or crypto local callee. Merely widening the blocked direction would
   make non-default declarations effectively unchecked across those calls and would undercut
   the replay contracts in `internal/replay/contracts.go:23-40,53-74`.

## Ratified Semantic Rule

For the closed-row validate path, let `required(E)` be the effective mode required by the body
or callee for effect `E`, after bare/default normalisation, and let `declared(E)` be the
effective mode on the enclosing function's declaration.

A declared effect covers a required effect exactly when:

1. the required label exists in the declared row; and
2. either the effective modes are equal, **or** the effect schema explicitly registers an edge
   from the declared mode to the required mode.

For Rand today, the resulting ordering is:

| Required mode | Declarations that subsume it |
|---|---|
| `os` (whether required spelling was bare or explicit `mode=os`) | `os`, `seeded`, `crypto` |
| `seeded` | `seeded` only |
| `crypto` | `crypto` only |

This is not a total capability hierarchy. `seeded` and `crypto` are incomparable; neither
subsumes the other. The only widening is the ratified **declared explicit mode covers required
default mode** edge, registered for Rand as `seeded -> os` and `crypto -> os`.
`DefaultModeFor` supplies default normalisation only; registering a default grants no
subsumption. At this baseline, a live check confirms that `Clock[mode=pinned]` parses but is
rejected with `EFF_PARAMS_NOT_SUPPORTED`; registering Clock modes and any intended subsumption
edges remains the downstream sprint's responsibility.

The explicitness condition matters. Bare `! {Rand}` and explicit
`! {Rand[mode=os]}` have the same runtime contract, but a bare declaration does not opt into a
non-default override. Both cover an os requirement by equality; neither covers a seeded or
crypto requirement.

### Ratification evidence

The durable decision artifact is both the queue row in
[`design_docs/v1-mission.md`](../../v1-mission.md) and the message body of commit
`4d32c71bb04e5dcbfd46920d36fee0a04812dcc8`, titled “docs(mission): record Mark's attended
decisions — subsumption YES, parity Lane A+B, M4b approved ($20 cap, post-quota-reset).” The
commit message records:

> 1. m-effect-replay-subsumption: YES — declared mode subsumes bare/os requirement; narrow
> SubsumeEffectRows validate-path relaxation only. Unparks effect sprints 2-4.

That commit modifies `design_docs/v1-mission.md` (`+12/-3`), whose queue row records the same
decision. Mark's recorded wording ratifies **declared mode subsumes bare/os requirement**. The
specific `seeded -> os` and `crypto -> os` edges are this document's Rand-only instantiation
of that ratified rule; Mark did not separately enumerate those two edges in the artifact.

### Replay-contract justification

`internal/replay/contracts.go:26-40,67-74` classifies:

- `Rand[mode=seeded]` as **deterministic**;
- `Rand[mode=os]` as **re-sampleable**;
- `Rand[mode=crypto]` as **opaque**.

The outer explicit declaration is the runtime dispatch authority: the parent implementation
captures the declared Rand mode and pushes a non-os mode at function entry
(`internal/eval/value.go:313`, `internal/eval/eval_expressions.go:199-211`). Allowing a
seeded/crypto declaration to call a default wrapper therefore preserves the declared contract:
the wrapper's builtin operation executes under the explicit outer mode.

The currently accepted reverse direction (`c3`, `c7`, `c8`) is a **separate existing bug
fixed by this item**, not correct subsumption. A local callee explicitly declares seeded or
crypto behavior and pushes that mode at its own entry; an os caller cannot truthfully claim
that operation under the re-sampleable os contract. Once callee parameters are preserved,
those cases must reject. This does not widen Mark's decision; it restores enforcement needed
to make the narrow widening meaningful.

## Goals

**Primary goal:** Make explicit non-default declarations able to use bare/default wrappers
while preserving enforceable replay contracts across local calls.

Success means:

- seeded and crypto Rand are runnable end to end through `std/rand`;
- local callee modes reach validation without parameter erasure;
- os declarations cannot hide seeded/crypto callees;
- a mode mismatch always names the effect, required mode, and declared mode;
- function-value unification remains invariant.

## Non-Goals

- No redesign of effect modes, row polymorphism, capability semantics, or runtime dispatch.
- No hierarchy between non-default modes.
- No change to `effectParamsCompatible` or `Unifier.unifyRows`.
- No Clock/Net/FS schemas, subsumption edges, or dispatch implementation. Downstream mode-port
  missions must register and test their own edges explicitly.
- No change to replay-contract labels, seed sourcing, `rand_seed`, trace schema, or bare-Rand
  runtime behavior.
- No source changes outside the narrow type/pipeline validation implementation, its tests,
  `examples/modal_rand.ail`, and directly corresponding documentation.

## Solution Design

### 1. Preserve declared rows across the validation call graph

Replace the validation-only `map[string][]string` representation with a representation that
retains each source declaration's complete `*types.Row` (labels, parameters, budgets, and
tail). Construct rows from `ast.EffectAnnotation` using the existing
`types.ElaborateEffectRowWithBudgets` path
(`internal/types/effects.go:364-417`), rather than re-parsing names or inventing a second
normaliser.

Thread that row map through `validateDecl`, `validateLambdaAnnotations`, and
`collectRequiredEffects`. At a locally-bound direct call, use the preserved row instead of
`stringSliceToEffectRow`. Imported/global calls continue to obtain their effect row from
`CoreTypeInfo` at `internal/pipeline/validate_effects.go:375-380`.

This closure of mode-blindness is **in scope and required**. Estimated cost is 0.5–1 day,
including focused pipeline tests. Deferring it would cause `c3`, `c7`, and `c8` to remain
green and would make the seeded/crypto replay declarations unenforceable at local call
boundaries.

Rows stored in the map must be copied or treated as immutable; collection/union must not mutate
the source-declared row. Existing budget and open-tail behavior must remain unchanged.

When a body has multiple required local calls with distinct modes of the same effect,
`collectRequiredEffects` must merge their parameters using the same asymmetric subsumption
rules, or preserve multiple bounds for later validation. It must not apply an invariant union
that fails during collection. This mirrors collection of imported multi-mode requirements and
ensures that collection preserves enough information for the enclosing declaration to be
checked against every requirement.

### 2. Apply asymmetric compatibility only in validation subsumption

Keep label subset checking in `SubsumeEffectRows`, but replace invariant parameter comparison
for this validate-only API with the ratified rule above:

- equal effective parameter maps pass;
- for the `mode` key only, an effect-schema edge may allow the declared mode to cover the
  required mode;
- all other parameter keys remain invariant;
- a non-default required mode never passes against a different mode;
- unknown/unregistered modes never gain a fallback.

Add the opt-in subsumption-edge metadata alongside `effectSchema`, the existing single source
of truth for legal effect/mode pairs. Register only Rand's `seeded -> os` and `crypto -> os`
edges in this sprint, and have validation consult that relation. `DefaultModeFor` must not
imply or synthesize an edge.

All production callers of `SubsumeEffectRows` are the validation sites enumerated below.
Document the function as validation subsumption so it is not reused as function-type
compatibility. If implementation clarity requires a specifically named validation helper,
the three call sites may move to it, but the behavior must remain confined to those sites.

### 3. Report mode mismatches structurally

Add a validation difference result that distinguishes:

- missing labels; and
- parameter mismatches, including effect name, key, required value, and declared value.

Both the inline-lambda error at `internal/pipeline/validate_effects.go:160-165` and
`formatEffectError` at `:539-565` must use it. Required user-visible wording is semantic, not
punctuation-sensitive, for example:

```text
Effect mode mismatch: Rand requires mode=seeded; declaration provides mode=os
```

The message must never show an empty `Missing effects:` line. For a genuinely absent label,
the existing `Missing effects: Rand` form remains valid. Suggestions must not propose a
label-only union that retains the wrong mode.

## Conflict Surface

This item changes type/pipeline behavior, so every comparison path is explicit:

1. **`internal/pipeline/validate_effects.go:160` — inline lambda annotations.** Compares an
   inline lambda body's required row with its original source-declared closed row. It adopts
   the new asymmetric validate rule and the structured mode diagnostic.
2. **`internal/pipeline/validate_effects.go:221` — top-level `LetRec` bindings.** Compares each
   recursive binding's collected requirements with its declaration. It adopts the same rule
   and must receive full declared rows.
3. **`internal/pipeline/validate_effects.go:245` — top-level `Let` bindings.** Same behavior for
   non-recursive bindings; it adopts the same rule and full-row propagation.
4. **Local call collection at `internal/pipeline/validate_effects.go:325,362-380`.** The
   `map[string][]string`/`stringSliceToEffectRow` path is replaced with full declared rows so
   callee modes are not erased.
5. **Function-value unification at
   `internal/types/unification_records.go:411-441`.** It directly uses
   `effectParamsCompatible` and remains invariant. Passing a seeded function where an
   os-function value is required (or vice versa) must continue to fail.
6. **`effectParamsCompatible` at `internal/types/effects.go:247-253`.** Unchanged. It remains
   equality after default normalisation and must not call the new asymmetric predicate.
7. **Existing guard `TestSubsumeEffectRows_InvariantOnParams` at
   `internal/types/effect_params_test.go:372-394`.** This test directly calls
   `SubsumeEffectRows`; it asserts the old validate-path behavior, not unification. Do not
   delete it. Rename/rewrite it as a table test for the new asymmetric rule: seeded declared
   covers os required; crypto declared covers os required; equal modes pass; os declared does
   not cover seeded/crypto required; seeded and crypto do not cover each other. Add or retain
   a separate test through `Unifier.unifyRows` proving function-value modes remain invariant.

No other production call sites of `SubsumeEffectRows` were found by `rg` at the verified
baseline. Existing direct unit tests in `internal/types/effects_test.go` continue to guard
label subset/no-hierarchy behavior.

## Examples and Acceptance Behavior

These snippets were live-checked for syntax at the baseline. Post-change expected outcomes:

```ailang
module subsum_repro
import std/rand (rand_int)
export func seeded_roll() -> int ! {Rand[mode=seeded]} = rand_int(1, 6)
```

`ailang check` changes from exit 1 with empty `Missing effects:` to exit 0.

```ailang
module c2
import std/rand (rand_int)
export func f() -> int ! {Rand[mode=crypto]} = rand_int(1, 6)
```

`ailang check` changes from exit 1 to exit 0.

The existing `c3`, `c7`, and `c8` syntax remains valid, but checking changes from exit 0 to
exit 1 with a non-empty mode mismatch. `c4` remains exit 1 with `Missing effects: Rand`.
`c1` remains exit 0. `c6` changes from exit 1 with an empty cause to exit 0 because the
explicit seeded declaration intentionally covers the bare/default wrapper requirement.

## Milestones

### M1 — Preserve modes and pin the semantic matrix (0.75 day)

- Change the validation-only declaration map to full rows.
- Add pipeline regressions for `c1`–`c8` and the direct wrapper blocker.
- Independently verifiable: collected requirements retain `seeded`/`crypto`, and the
  pre-relaxation matrix exposes both directions accurately.

### M2 — Implement validate-only asymmetric subsumption and diagnostics (1 day)

- Add the schema-owned Rand edges and implement the exact ordering above at the three
  validation call sites.
- Rewrite the direct subsumption guard; add/retain invariant unification coverage.
- Add structured missing-label/mode-mismatch output.
- Independently verifiable: the complete matrix has the specified green/red outcomes and no
  rejection contains an empty cause.

### M3 — End-to-end Rand acceptance and documentation (0.75–1 day)

- Update `examples/modal_rand.ail`: delete the `KNOWN LIMITATION` block and add runnable
  seeded and crypto calls through `std/rand`.
- Run seeded twice with the same `AILANG_SEED` and require identical output; exercise crypto
  successfully with its opaque contract.
- Update the parameterised-effects guide and parent status note only where the limitation is
  now obsolete.
- Independently verifiable: the example checks and runs end to end; repository gates pass.

Total: 2.5–2.75 days nominal, with a hard ceiling of 3–4 days. If full-row propagation reveals
a wider type-system change, stop and return it to Mark rather than expanding the sprint.

## Testing Strategy

### Exact `.ail` regression fixtures

Promote the verified `/tmp` cases into durable integration fixtures (names may follow the
repository's testdata convention):

| Fixture | Required green result |
|---|---|
| `blocker` (`subsum_repro` case): seeded declaration → imported bare `rand_int` | `ailang check` exit 0 |
| `c1`: explicit os declaration → imported bare `rand_int` | exit 0 |
| `c2`: crypto declaration → imported bare `rand_int` | exit 0 |
| `c3`: bare os caller → local seeded callee | exit 1; names `Rand`, required `seeded`, declared `os` |
| `c4`: pure caller → local seeded callee | exit 1; `Missing effects: Rand` |
| `c6`: seeded declaration → local bare wrapper containing `rand_int` | exit 0 |
| `c7`: explicit os caller → local seeded callee | exit 1; names required `seeded`, declared `os` |
| `c8`: bare os caller → local crypto callee | exit 1; names required `crypto`, declared `os` |

Record exit codes directly, without pipelines. Diagnostics may be golden-tested after normalising
temporary paths, but the effect/mode values are mandatory assertions.

### Unit and integration coverage

- Table-test all Rand ordering pairs, bare/default equivalence, a non-mode parameter mismatch,
  a missing label, nil/pure rows, and open-row behavior already supported by validation.
- Test that `DefaultModeFor` alone grants no subsumption. Downstream mode-port missions must
  register and test their own Clock/Net/FS edges explicitly.
- Test all three validate call sites (`:160`, `:221`, `:245`), including recursive and inline
  lambdas.
- Test local declaration row preservation for parameters and budgets.
- Test a body that calls both a seeded local helper and an os local helper: collection must not
  fail by invariant union, and the enclosing declaration must be validated against both
  requirements using the asymmetric rule. Include the corresponding imported multi-mode case.
- Test function-value unification separately: unequal modes still reject.
- Test mode diagnostics for both inline-lambda and top-level formatters; forbid empty
  `Missing effects:`.

### End-to-end acceptance bar

`examples/modal_rand.ail` must contain runnable seeded and crypto functions that call the
existing `std/rand` wrapper. Its current `KNOWN LIMITATION` block
(`examples/modal_rand.ail:28-37`) is deleted.

- `ailang check examples/modal_rand.ail` succeeds.
- With a valid entry/capability invocation chosen consistently with the existing example,
  two seeded runs under the same `AILANG_SEED` produce identical seeded results.
- The crypto path runs successfully without `AILANG_SEED`; it is not asserted deterministic.
- Trace coverage continues to report `deterministic` for seeded and `opaque` for crypto.
- `make test`, `make verify-examples`, and `make lint` are green, or any unrelated baseline
  failure is documented with a reproduced pre-change result.

## Risks and Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Relaxation leaks into function-value unification | High | Do not change `effectParamsCompatible`; direct unequal-mode unification guard |
| Parameter erasure remains on one local-call form | High | Fixtures cover `Let`, `LetRec`, inline lambda, imported/global, and one-hop local wrapper |
| Explicit declaration does not actually control runtime dispatch | High | End-to-end seeded repeatability and crypto execution through `std/rand` are hard gates |
| Full-row map aliases mutable rows | Medium | Treat stored declaration rows as immutable/copy before union; focused budget/param test |
| Diagnostic suggests adding an already-present label | Medium | Structured parameter difference; golden mismatch message |
| Unregistered effects acquire accidental ordering | Medium | Consult explicit schema edges only; test that a registered default alone grants no subsumption |

## Open Questions

No semantic question remains for the ratified Rand rule.

**Escalation trigger for Mark (single answerable question):** If preserving full declared rows
cannot be confined to `internal/pipeline/validate_effects.go` and existing row elaboration,
should this item expand beyond the validate path to change type inference, or stop and create a
separate prerequisite? Recommendation: stop and create the prerequisite; do not silently widen
this item.

## Quorum Verification Log

Designer: `codex:gpt-5.6-sol` (mission designer rotation). Reviewers: `gpt5-6-sol` +
`gemini-3-1-pro` + the controller (opus, in-session). Three rounds, all objections resolved with
the reviewers' OWN `proposed_fix` text — no controller-invented resolutions, no objection
overridden.

| Round | Verdict | Blocking objection | Resolution |
|---|---|---|---|
| R1 | blocked | `gpt5-6-sol`: the rule auto-applied to every effect with a registered default, an unverified cross-effect hierarchy wider than the ratified decision | Designer revision: rule is now **schema-edge opt-in** registered alongside `effectSchema`, Rand-only (`seeded -> os`, `crypto -> os`); doc states `DefaultModeFor` grants no subsumption; Non-Goals excludes downstream edges |
| R1 | blocked | `gemini-3-1-pro`: runtime-dispatch and elaboration premises asserted but absent from the Verification Log | Designer revision: rows added; controller independently re-verified each citation before it was marked Verified |
| R2 | blocked | `gpt5-6-sol`: behavioural results came from a `-dirty` binary while attributed to `4d32c71bb`; fixtures/commands omitted, so the matrix was not reproducible | **Controller executed the reviewer's proposed fix rather than arguing it**: pinned a worktree to the exact SHA, confirmed `git status --porcelain` empty, rebuilt (`v0.30.0-201-g4d32c71bb`, no `-dirty`), and re-ran all nine fixtures unpiped. Results **identical** — so the matrix stands on a clean baseline. Fixtures + exact commands embedded in Appendix A |
| R2 | pass | `gemini-3-1-pro` passed; raised a non-blocking catch on unioning conflicting local modes in `collectRequiredEffects` | Folded into Solution Design + Testing Strategy |

**Narrow-refinement carve-out applied** (mission-control Gate 2). The sole R2 blocker disputed
**attribution/reproducibility of evidence**, not the design direction, and carried a concrete
reviewer-authored `proposed_fix`. The controller performed that fix and the clean re-run
reproduced every result, so the objection is satisfied **by evidence**. Had the clean results
differed, the reviewer would have been correct and the problem statement would have required
rewriting — that outcome was live, which is why the check was run rather than reasoned past.

## Related Documents

- [M-EFFECT-REPLAY-CONTRACTS](../../implemented/v0_30_0/m-effect-replay-contracts.md) — parent
- [M-EFFECT-CLOCK-NET-FS-MODES](m-effect-clock-net-fs-modes.md) — downstream shared-gate consumer
- [M-EFFECT-REFINEMENT](m-effect-refinement.md) — parameterised-effect program
- [Parameterised effects guide](../../../docs/docs/guides/parameterised-effects.md)

## Verification Log

| Fact | Source / command | Status |
|---|---|---|
| Clean source state | `git status --porcelain` before build: empty; `git rev-parse HEAD`: `4d32c71bb04e5dcbfd46920d36fee0a04812dcc8`; `git describe --tags --dirty`: `v0.30.0-201-g4d32c71bb` | Verified 2026-07-27 |
| Clean build identity | `make build` exit 0; `bin/ailang --version`: `AILANG v0.30.0-201-g4d32c71bb`, short commit `4d32c71`, full commit `4d32c71bb04e5dcbfd46920d36fee0a04812dcc8`; no `-dirty` suffix or freshness warning; post-build `git status --porcelain`: empty | Verified 2026-07-27 |
| Earlier dirty suffix was documentation-data-only | `git diff --stat HEAD -- '*.go' 'go.mod' 'go.sum' 'Makefile'`: empty; the complete tracked diff comprised only `docs/static/benchmarks/latest.json`, `docs/static/benchmarks/os/history.json`, and `docs/static/benchmarks/os/latest.json` | Verified 2026-07-27 |
| Blocked seeded and crypto wrapper calls; os control succeeds | Exact unpiped commands `bin/ailang check /tmp/subsum_fixtures/blocker.ail`, `bin/ailang check /tmp/subsum_fixtures/c2.ail`, and `bin/ailang check /tmp/subsum_fixtures/c1.ail`; exits 1, 1, 0; fixtures and diagnostics in Appendix A | Verified clean baseline 2026-07-27 |
| Current directional matrix | Exact unpiped commands `bin/ailang check /tmp/subsum_fixtures/{c3,c4,c6,c7,c8}.ail`, expanded and run separately; exits 0, 1, 1, 0, 0; fixtures and diagnostics in Appendix A | Verified clean baseline 2026-07-27 |
| Mode-mismatch diagnostic can be empty | Clean blocker/c2/c6 output has an empty `Missing effects:` payload; `validate_effects.go:160-165,539-546`; label-only difference at `effects.go:651-665` | Verified clean baseline |
| Local declaration map erases params | `validate_effects.go:109-114,325,362-371`; `ast.go:116-123` | Verified |
| Default normalisation maps bare Rand to os | `effects.go:206-227,255-270` | Verified |
| Three production `SubsumeEffectRows` call sites | `rg -n "SubsumeEffectRows" internal`; `validate_effects.go:160,221,245` | Verified |
| Unification is a separate invariant path | `unification_records.go:411-441`; `effects.go:247-253` | Verified |
| Existing named guard directly tests subsumption, not unification | `effect_params_test.go:372-394` | Verified |
| Replay contract labels and Rand assignments | `internal/replay/contracts.go:23-40,53-74` | Verified |
| Runtime dispatch captures and pushes declared mode | `internal/eval/value.go:313`; `internal/eval/eval_expressions.go:199-211` | Verified |
| `types.ElaborateEffectRowWithBudgets` exists and handles budgets/tails | declaration at `internal/types/effects.go:369`; implementation at `:369-417` | Verified |
| `DefaultModeFor` registers defaults but no subsumption relation | `internal/types/effects.go:160-170` | Verified |
| `effectSchema` is the existing source of truth for legal effect/mode pairs | `internal/types/effects.go:172-180` | Verified |
| Ratified semantic rule | Queue row in [`design_docs/v1-mission.md`](../../v1-mission.md) and commit `4d32c71bb` message body both record “declared mode subsumes bare/os requirement; narrow `SubsumeEffectRows` validate-path relaxation only”; Rand's `seeded -> os` and `crypto -> os` edges are this document's instantiation, not separately enumerated by Mark | Verified durable artifact 2026-07-27 |
| Current Clock parameter boundary | Exact unpiped command `bin/ailang check /tmp/subsum_fixtures/clock.ail`; exit 1 with `EFF_PARAMS_NOT_SUPPORTED: effect 'Clock' does not support parameters (found: mode)` | Verified clean baseline 2026-07-27 |

## Appendix A: Clean-Baseline Fixtures and Commands

These are the complete fixture contents used for the clean-baseline verification. The sprint
promotes them into repository testdata; embedding them here makes the cited baseline durable
until that happens. Each command below was run separately from the clean worktree root in the
form `bin/ailang check /tmp/subsum_fixtures/<name>.ail`. Exit codes were read directly, with no
pipe.

### `blocker.ail`

```ailang
module blocker
import std/rand (rand_int)
export func seeded_roll() -> int ! {Rand[mode=seeded]} = rand_int(1, 6)
```

Command: `bin/ailang check /tmp/subsum_fixtures/blocker.ail`  
Exit: 1  
Diagnostic: lambda at `blocker.ail:3:8` uses effects not declared in its
`! {Rand[mode=seeded]}` annotation; `Missing effects:` has an empty payload.

### `c1.ail`

```ailang
module c1
import std/rand (rand_int)
export func f() -> int ! {Rand[mode=os]} = rand_int(1, 6)
```

Command: `bin/ailang check /tmp/subsum_fixtures/c1.ail`  
Exit: 0  
Diagnostic: no errors.

### `c2.ail`

```ailang
module c2
import std/rand (rand_int)
export func f() -> int ! {Rand[mode=crypto]} = rand_int(1, 6)
```

Command: `bin/ailang check /tmp/subsum_fixtures/c2.ail`  
Exit: 1  
Diagnostic: effects not declared in the `! {Rand[mode=crypto]}` annotation;
`Missing effects:` has an empty payload.

### `c3.ail`

```ailang
module c3
export func g() -> int ! {Rand[mode=seeded]} = 42
export func f() -> int ! {Rand} = g(())
```

Command: `bin/ailang check /tmp/subsum_fixtures/c3.ail`  
Exit: 0  
Diagnostic: no errors.

### `c4.ail`

```ailang
module c4
export func g() -> int ! {Rand[mode=seeded]} = 42
export func f() -> int = g(())
```

Command: `bin/ailang check /tmp/subsum_fixtures/c4.ail`  
Exit: 1  
Diagnostic: `Effect checking failed for function 'f'`; `Missing effects: Rand`.

### `c6.ail`

```ailang
module c6
import std/rand (rand_int)
export func g() -> int ! {Rand} = rand_int(1, 6)
export func f() -> int ! {Rand[mode=seeded]} = g(())
```

Command: `bin/ailang check /tmp/subsum_fixtures/c6.ail`  
Exit: 1  
Diagnostic: effects not declared in the `! {Rand[mode=seeded]}` annotation;
`Missing effects:` has an empty payload.

### `c7.ail`

```ailang
module c7
export func g() -> int ! {Rand[mode=seeded]} = 42
export func f() -> int ! {Rand[mode=os]} = g(())
```

Command: `bin/ailang check /tmp/subsum_fixtures/c7.ail`  
Exit: 0  
Diagnostic: no errors.

### `c8.ail`

```ailang
module c8
export func g() -> int ! {Rand[mode=crypto]} = 42
export func f() -> int ! {Rand} = g(())
```

Command: `bin/ailang check /tmp/subsum_fixtures/c8.ail`  
Exit: 0  
Diagnostic: no errors.

### `clock.ail`

```ailang
module clock
export func f() -> int ! {Clock[mode=pinned]} = 1
```

Command: `bin/ailang check /tmp/subsum_fixtures/clock.ail`  
Exit: 1  
Diagnostic: `EFF_PARAMS_NOT_SUPPORTED: effect 'Clock' does not support parameters (found: mode)`.

---

**Document created**: 2026-07-27  
**Decision authority**: Mark; durable record in the `design_docs/v1-mission.md` queue row and
commit `4d32c71bb` message body
