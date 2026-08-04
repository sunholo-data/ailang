# M-BYTECODE-PATTERN-ARITY-FIX — fixed-length list patterns match longer lists under `--bytecode` (silent wrong result)

**Status**: Planned — **UNBLOCKED**, ready to sprint. Spun out of
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

## Root cause (already known — verified by the controller, see parent doc's Verification Log)

**The bytecode VM does not check the LENGTH of a fixed-length list pattern.** A pattern
`[p1, …, pn]` matches any list of length **≥ n**, binding the first `n` elements and silently
discarding the tail. The guard is `len >= n` where it must be `len == n`. Only the *overflow*
direction is unchecked — a list **shorter** than the pattern correctly fails to match, which is why
this survived so long.

Minimal repro (`match` with arms `[] / [x] / [a,b] / [p, ...rest]`):

| input | `ailang run` (correct) | `ailang run --bytecode` |
|---|---|---|
| `[]` | `empty` | `empty` |
| `[7]` | `one:7` | `one:7` |
| `[7,8]` | `two:7,8` | **`one:7`** |
| `[7,8,9]` | `many:7` | **`one:7`** |

Confirmed general at n=1, 2 **and** 3 (`[a,b]` arm swallows `[7,8,9]` → `two:7,8`; `[a,b,c]` arm
swallows `[7,8,9,10]` → `three:7,8,9`).

This fully explains `recursion_quicksort.ail` (the flagship symptom): the `[x] => [x]` arm captures
the whole 8-element input and returns the head as a singleton, so `ailang run --bytecode` silently
prints `[3]` instead of the correct `[1, 1, 2, 3, 4, 5, 6, 9]`, exit 0, no error, no fallback
warning on stderr. `sortBy`'s identical `[3]` is the same arm in `std/list`, not a second bug.
`show`, `filter`, `concat`, recursion-as-argument, and closure capture of a pattern-bound variable
were each ruled out (all MATCH under both engines) before landing on this cause.

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

`internal/gen/lower/` / `internal/vm/` / `internal/bytecode/` — exact site to be confirmed by
whoever picks this up (the parent doc's B1 milestone did not localize the opcode/function, only
the semantic cause). `ailang disasm` the repro below and trace the list-pattern-match opcode's
length check; compare against the evaluator's equivalent (which correctly requires `len == n`).

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

- `internal/gen/lower/` / `internal/vm/` / `internal/bytecode/` (exact files unknown until the fix
  site is located) — list/closure codegen is shared by *all* list-recursive programs, not just
  `recursion_quicksort.ail`; run the whole-corpus parity harness (`go run
  ./scripts/verify_bytecode_parity.go`) before/after as a blast-radius check even though this doc
  does not depend on the harness's classification scheme.

**Programs that MUST still work post-change (verified MATCH at parent-doc HEAD `33be8f5a7`):**
`examples/runnable/cons_expression.ail`, `examples/runnable/block_recursion.ail`,
`examples/runnable/adt_list_fields.ail`, `examples/runnable/effectful_list.ail`.

## Testing Strategy

Golden: minimal repro + exact-output goldens (AC1–AC3) under `tests/golden/bytecode/`.
Whole-corpus: full parity harness before/after, as a regression/blast-radius check (not a gate —
this doc doesn't own the harness's bucket definitions).

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
