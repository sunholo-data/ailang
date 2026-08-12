# M-BYTECODE-PATTERN-ARITY-FIX — fixed-length list patterns match longer lists under `--bytecode` (silent wrong result)

**Status**: Planned — **QUORUM-RESOLVED 2026-08-12, ready to sprint.** Two blocked rounds, both on
premise objections the controller then measured (see the Quorum verification log at the end);
resolved via the narrow-refinement carve-out with V-H and V-I appended. Spun out of
[m-bytecode-vm-parity-bugs.md](m-bytecode-vm-parity-bugs.md) (Milestones B1's quicksort half + B2)
per Mark's 2026-08-04 decision on that doc's parked A2 question: **option C** — split the doc, land
this P0 fix immediately since it does not depend on A2's harness-classification redesign at all.
The parent doc's remaining scope (A1, A2, B3 closure-dispatch family, B4 replay policy) stays
parked pending a second design round on A2's semantic-effect-extraction question; do not block this
doc on that round.

**Target**: v1.0.0 (clause-2 soundness residue on the V1 mission queue)
**Priority**: **P0** — silent wrong result, no error, no fallback. Direct NO-SILENT-FALLBACKS and
axiom A1 (determinism) violation.
**Estimated**: ~0.5–1 day (root cause already known — see below; this is fix + regression tests,
not investigation).
**Dependencies**: none. Explicitly does NOT depend on m-bytecode-vm-parity-bugs.md's parked A2
question — that question is about the *parity harness's* classification scheme
(`scripts/verify_bytecode_parity.go`), not about program correctness. This fix changes VM/codegen
behavior directly; its acceptance criteria are golden-output tests, not harness bucket counts.
**Filed bug**: [#505](https://github.com/sunholo-data/ailang/issues/505)

## Root cause (localized and mutation-verified at HEAD `8ecebc0e1`)

**The bytecode lowering path does not check the LENGTH of a fixed-length list pattern.** A pattern
`[p1, …, pn]` matches any list of length **≥ n**, binding the first `n` elements and silently
discarding the tail. The guard is `len >= n` where it must be `len == n`. Only the *overflow*
direction is unchecked — a list **shorter** than the pattern correctly fails to match, which is why
this survived so long.

Minimal repro (`match` with arms `[] / [x] / [a,b] / [a,b,c] / [p, ...rest]`):

| input | `ailang run` (correct) | `ailang run --bytecode` |
|---|---|---|
| `[]` | `empty` | `empty` |
| `[7]` | `one:7` | `one:7` |
| `[7,8]` | `two:7,8` | **`one:7`** |
| `[7,8,9]` | `three:7,8,9` | **`one:7`** |
| `[7,8,9,10]` | `many:7` | **`one:7`** |

The earliest non-empty fixed-length arm swallows every list at least that long. In this ordered
repro, `[x]` therefore captures every non-empty input and the n=2 and n=3 arms are never reached.
The same `len >= n` lowering applies at n=1, 2, and 3; an isolated n=2 or n=3 arm likewise accepts
a longer list. AC2 tests each arity independently so arm ordering cannot hide that generality.

This fully explains `recursion_quicksort.ail` (the flagship symptom): the `[x] => [x]` arm captures
the whole 8-element input and returns the head as a singleton, so `ailang run --bytecode` silently
prints `[3]` instead of the correct `[1, 1, 2, 3, 4, 5, 6, 9]`, exit 0, no error, no fallback
warning on stderr. `sortBy`'s identical `[3]` is the same arm in `std/list`, not a second bug.
`show`, `filter`, `concat`, recursion-as-argument, and closure capture of a pattern-bound variable
are not alternative fix sites: the mutation experiment in V-E changes only the list-pattern
condition and makes both the minimal repro and flagship symptom byte-identical to the evaluator.

**Impact is wider than one example**: every fixed-length list pattern in every `--bytecode` program
can silently return a wrong answer.

## What is explicitly OUT of scope here (stays in the parent doc)

- The **closure-dispatch family** (`array_basic`'s `GET_TAG on Closure`, `array_grid`'s
  `arith MUL on Closure`, `module_let_helpers`'s `arith ADD: type mismatch Int vs Closure`) — this
  is the OTHER half of the parent doc's Milestone B1, has an **unknown** root cause (unlike this
  bug), and was not named by Mark's option-C carve-out. Stays in the parent doc's Milestone B1/B3.
- The parity harness's classification scheme (A1 eval tuple-show, A2 harness honesty) — parked on
  the A2 semantic-effect-extraction question.
- The unsafe-effect-replay fallback policy (B4) — depends on A2's new `VM_UNSAFE_REPLAY` bucket
  existing in the harness (its ACs assert harness-reported counts), so it stays parked with A2.
- Closing the M-BYTECODE-2E bridge gaps (Result.Ok/Err TaggedValue, MapValue, BytesValue).

## Fix location

The fix site is `internal/gen/lower/match.go`, function `lowerPatternCond` (declared at line 390),
in `case *core.ListPattern` at lines 410–427. Line 414 special-cases the empty pattern to
`stmt.OpEq`; lines 421–427 currently emit `stmt.OpGte` for every other list pattern. The existing
node discriminator is `core.ListPattern.Tail` (`internal/core/core.go:370`):

- `Tail == nil`: fixed-length pattern, emit `stmt.OpEq` (including the n=0 empty pattern).
- `Tail != nil`: tailed pattern, emit `stmt.OpGte` because “at least n” is intentional.

Pattern matching is lowered away in `internal/gen` before execution. `internal/vm` and
`internal/bytecode` are not involved: neither contains a `Pattern` occurrence in any `.go` file,
while the same search finds 13 files under `internal/gen` (V-D).

## Milestone — fix the pattern-arity bug (P0, ~0.5–1d)

Renumbered from the parent doc's B2; acceptance criteria carried over verbatim (they were already
written against this exact bug, not against harness state):

- **AC1** (was AC8): golden test asserting `examples/runnable/recursion_quicksort.ail` under
  `--bytecode` prints exactly `Quicksort: [1, 1, 2, 3, 4, 5, 6, 9]` and
  `sortBy:    [1, 1, 2, 3, 4, 5, 6, 9]` — fails on HEAD (`[3]`). This AC is the one a
  DIVERGE-count AC cannot fake.
- **AC2** (was AC8b, generality — the bug is #505, not one example): a table-driven test asserting
  **fixed-length list-pattern arity** under `--bytecode` for n=1, 2 and 3, in both directions: a
  pattern of length `n` must match a list of length exactly `n`, must NOT match a **longer** list
  (fails on HEAD: `[x]` swallows `[7,8]`, `[a,b]` swallows `[7,8,9]`, `[a,b,c]` swallows
  `[7,8,9,10]`), and must NOT match a **shorter** one (passes on HEAD — include it so a fix that
  over-corrects into the underflow direction is caught). AC1 alone is satisfiable by a
  quicksort-shaped special case; this AC is what forces the real fix.
- **AC3** (was AC9): the minimal quicksort repro passes under `--bytecode` with **no fallback**
  (stderr asserted free of `"falling back to evaluator"`).

Commit a minimal failing `.ail` repro under `tests/golden/bytecode/` (the pattern-arity table
above is already minimal — do not re-derive it).

## Conflict Surface

- `internal/gen/lower/match.go:lowerPatternCond` is the shared statement-IR lowering path for every
  list pattern in every program using the bytecode pipeline. The real regression risks are (1)
  over-correcting into the underflow direction so a fixed-length pattern accepts a shorter list,
  and (2) changing a tailed pattern from `>= n` to `== n`. AC2 covers both boundaries and both
  fixed arity outcomes; its tailed fallback also exercises the `Tail != nil` path. Run the
  whole-corpus parity harness (`go run
  ./scripts/verify_bytecode_parity.go`) before/after as the blast-radius check even though this doc
  does not depend on the harness's classification scheme.

**Programs that MUST still work post-change (verified MATCH under the probe at HEAD `8ecebc0e1`,
V-F):**
`examples/runnable/cons_expression.ail`, `examples/runnable/block_recursion.ail`,
`examples/runnable/adt_list_fields.ail`, `examples/runnable/effectful_list.ail`.

## Testing Strategy

Golden: minimal repro + exact-output goldens (AC1–AC3) under `tests/golden/bytecode/`.
Whole-corpus: full parity harness before/after, as a regression/blast-radius check (not a gate —
this doc doesn't own the harness's bucket definitions).

The existing generator suite does not pin this behavior in either direction: it remains green
under the probe (`go test ./internal/gen/...`, rc=0; V-G). AC2 is therefore the regression test
that will prevent a future return to unconditional `OpGte` lowering.

## Verification Log

All rows were measured by the controller on 2026-08-12 at HEAD `8ecebc0e1`. The probe mentioned
below was restored byte-identically and is not committed.

| ID | Command / check | Observed output |
|---|---|---|
| V-A | `ailang run --caps IO examples/runnable/recursion_quicksort.ail`; then `ailang run --bytecode --caps IO examples/runnable/recursion_quicksort.ail` | Evaluator (known-positive control): `Quicksort: [1, 1, 2, 3, 4, 5, 6, 9]` / `sortBy:    [1, 1, 2, 3, 4, 5, 6, 9]`. Bytecode: `Quicksort: [3]` / `sortBy:    [3]`, rc=0; stderr has no `falling back` line. |
| V-B | Run the inline minimal repro with arms `[] / [x] / [a,b] / [a,b,c] / [p, ...rest]` and inputs `[]`, `[7]`, `[7,8]`, `[7,8,9]`, `[7,8,9,10]`, once with `ailang run` and once with `ailang run --bytecode` | Evaluator (known-positive control): `empty / one:7 / two:7,8 / three:7,8,9 / many:7`. Bytecode: `empty / one:7 / one:7 / one:7 / one:7`; the first non-empty fixed arm swallows all later non-empty inputs. |
| V-C | `sed -n '390,430p' internal/gen/lower/match.go`; `sed -n '360,378p' internal/core/core.go` | `lowerPatternCond`, `case *core.ListPattern`, builds `OpEq` for empty at line 414 but `OpGte` for every non-empty pattern at lines 421–427. `core.ListPattern` has `Elements []CorePattern` and `Tail *CorePattern`; `Tail == nil` means exact length, `Tail != nil` means at least n. |
| V-D | Search `.go` files under `internal/vm`, `internal/bytecode`, and control `internal/gen` for `Pattern` | `internal/vm` and `internal/bytecode`: zero occurrences. Known-positive control `internal/gen`: 13 matching files. Pattern matching is lowered before the VM sees it. |
| V-E | Apply the probe at V-C (`OpEq` when `p.Tail == nil`, retain `OpGte` when non-nil); assert source hash changes; `go build`; run patched bytecode minimal repro and quicksort; run evaluator control; restore backup and hash again | Probe landed; build rc=0. Patched bytecode minimal repro: `empty / one:7 / two:7,8 / three:7,8,9 / many:7`, byte-identical to evaluator. Patched quicksort: exact V-A evaluator output. Evaluator unchanged. Restored SHA-256: `fd1d890e7d3e1ed58211c0fec4eab41640aad7d8274ca31b09d25b6aa87500a2`. This proves reachability and causation at the pinned site. |
| V-F | Under the probe, run `cons_expression`, `block_recursion`, `adt_list_fields`, and `effectful_list` under both engines; compare SHA-256 of the last 20 output lines | All four engine pairs MATCH. Distinct hash prefixes: `2230a604cb25`, `d4295f6e4c13`, `5430d1a50e8f`, `69fd90893805`. Known-positive control: mutually distinct hashes show the instrument discriminates. Do not use `md5` on this machine; the unavailable command previously produced vacuous empty-string matches. |
| V-G | With the probe applied: `go test ./internal/gen/...` | rc=0; `block`, `emitgo`, `golang`, `lower`, and `stmt` all pass. This green suite is the control showing the probe does not conflict with an existing expectation; it also contains no regression that catches unconditional `OpGte`, so AC2 is required. |
| V-H | **Isolated-arm arity, both directions.** Two single-fixed-arm modules with only a `[p, ...rest] => "other"` fallback, so no shorter arm can intercept. n=2 arms `[a,b]`, inputs `[7]` / `[7,8]` / `[7,8,9]`; n=3 arms `[a,b,c]`, inputs `[7,8]` / `[7,8,9]` / `[7,8,9,10]`. Each run under `ailang run` and `ailang run --bytecode` | n=2 evaluator (known-positive control): `other / two:7,8 / other`. n=2 bytecode: `other / two:7,8 / `**`two:7,8`**. n=3 evaluator: `other / three:7,8,9 / other`. n=3 bytecode: `other / three:7,8,9 / `**`three:7,8,9`**. So **underflow is correct on both engines** (measured, previously only asserted) and the **overflow bug is confirmed independently at n=2 and n=3** with no arm-ordering masking. This row exists because V-B's combined block lets `[x]` intercept every non-empty input, which makes V-B silent about the n=2/n=3 arms — an ordering artifact the round-2 quorum correctly refused to accept as proof. |
| V-I | **The `Tail` invariant the fix branches on, verified by exhaustive enumeration of construction sites** rather than from the struct definition. Search all non-test `.go` for `core.ListPattern{` / `&ListPattern{` | Exactly **two** construction sites exist, both in `internal/elaborate/patterns.go` (a third hit, `internal/core/gob.go:38`, is a `gob.Register` type registration, not a construction). **`:150`** is the `::` cons-constructor case: it always sets `Tail: &tailPat`, so `Tail != nil`, and "at least 1" is the intended semantics → `OpGte` stays correct there. **`:203`** is the `ast.ListPattern` case: `tail` is left `nil` unless the surface pattern has a `Rest` (`...rest`), so `Tail == nil` ⟺ the source pattern was a closed `[…]` with no rest → exactly-n. Known-positive control: the same search matches `ListPattern` across 10+ files, so it is not a broken pattern. **The invariant therefore holds over the whole codebase, not merely over the type declaration**, and the fix cannot silently reinterpret another list-pattern form because no other form is ever constructed. |

## Quorum verification log

- **Round 1 (2026-08-12T20:19:29Z) — BLOCKED.** Both reviewers raised the *same premise* objection:
  the doc claimed a known root cause while leaving the fix site unlocalized across three packages,
  and cited the parent doc's log rather than verifying anything here. Per the mission's rule 3f the
  controller measured the premise instead of forwarding the objection; the result is V-A .. V-G,
  and the localization shrank the fix site from three packages to one function and one operator.
- **Round 2 (2026-08-12T20:25:02Z) — BLOCKED, and both objections were correct.**
  `gpt5-6-sol`: the `Tail == nil` ⟺ exact-length invariant was read off the struct rather than
  established across constructors. `gemini-3-1-pro`: V-B's ordered match block masks the n=2/n=3
  arms, so both the underflow-correctness claim and the isolated overflow claim were asserted, not
  proven. Neither objection disputed the design direction; both asked for a measurement.
- **Resolution — narrow-refinement carve-out, bounded 2nd revision.** The controller ran both
  checks and appended **V-H** (isolated-arm arity, both directions, both arities) and **V-I**
  (exhaustive enumeration of `core.ListPattern` construction sites). Both objections are satisfied
  by measurement rather than overridden. No design decision was made by the controller, and the
  milestone, AC numbering, estimate and scope boundaries are unchanged from the pre-quorum doc.

## Related Documents

- [m-bytecode-vm-parity-bugs.md](m-bytecode-vm-parity-bugs.md) — parent doc; retains A1/A2 (parked
  on A2's semantic-extraction question) and B3/B4 (closure-dispatch family + replay policy)
- Issue [#505](https://github.com/sunholo-data/ailang/issues/505)

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|----------------|
| A1: Determinism | +1 | Silent cross-backend divergence (quicksort-class) eliminated |
| A11: Structured Failure | +1 | The silent wrong-result path becomes impossible to ship |
| Others | 0 | No language-surface change |

**Net Score: +2** → **Proceed.**

---

**Document created**: 2026-08-04 (spun out of m-bytecode-vm-parity-bugs.md per Mark's option-C decision)
