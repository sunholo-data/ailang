## M-ONBOARDING-ERGONOMICS: Smooth the 0→1 cliff — stdlib helpers + doc-truth fixes

**Status**: IMPLEMENTED (on `dev`, pending release) — §1 examples+gate, §2 `nth_or`/`head_or`/`last_or`, §3 clearer `.ail` error, §5 module-let docs, §6 caps-per-module docs, §7 stdin docs. (§4 `pure func` already in effects reference.) Two runtime bugs surfaced by the gate — `deriving_eq` Eq-dict + `effect_budget_demo` cumulative `@limit` — logged for separate follow-up.
**Target**: v0.27.0 (tentative)
**Priority**: P2 (Medium — none of these blocks outright, but together they triple a first build's time)
**Estimated**: 1 day (mostly docs + one small stdlib add)
**Dependencies**: None. Independent of [M-TERMINAL-IO](m-terminal-io.md), which covers the two hard runtime bugs from the same report.

**Reported from**: Snake Showdown build team (external 0→1 user), AILANG v0.26.1 — see `ailang-feedback-for-mark.md`, Friction #1, #3, #4, #6, #7, #8, #9.
**Verified against**: v0.26.2 (current `dev`).

---

## Verdict: VALID — these are mostly doc-truth gaps and one cheap stdlib add, not language changes

A 0→1 user's build took 3.5× longer than the Python equivalent. After excluding the two
genuine runtime bugs (escapes + flush — separate doc), the remaining friction is almost
entirely **documentation that is missing or ambiguous, plus learning material (examples) that
is broken**, plus one missing convenience helper. The cheapest, highest-leverage wins in the
whole report live here. The two most damaging items are where our own teaching material is
*wrong* rather than merely absent: 8 examples that don't run (§1) and tutorials that show a
rejected `.ailang` extension (§3) — users trust those over prose, so they actively build the
wrong mental model.

The capability system, `Option`-returning `nth`, and `pure func` enforcement are all *good
designs the user praised* — the fix is discoverability, not redesign.

---

## Items

### 1. `++` on strings — the docs are right, but 8 shipped examples are broken (rotted examples + a verification gap)

`++` is **list-only by design** since v0.13.0 (M-CONCAT-DISAMBIG Phase 2). This is the
intended, tested contract — [internal/pipeline/concat_string_error_test.go:25](../../internal/pipeline/concat_string_error_test.go#L25)
asserts `string ++ string` *must* error — and the MCP `onboarding_guide` correctly teaches
it. **The docs are not the bug.** Verified empirically on v0.26.2:

```
$ ailang run --caps IO --entry main /tmp/test.ail   # body: println("a" ++ "b")
Error: ++ operator: `++` is for lists only. For strings use "${expr}" interpolation,
concat([parts]), or join(sep, parts).
```

The actual bug: **8 examples in `examples/` still use `++` on strings and do not run.** They
predate the v0.13.0 change and were never migrated:

`bounded_xml_fold.ail`, `datetime_demo.ail`, `deriving_eq.ail`, `effect_budget_demo.ail`,
`first_non_repeat.ail`, `short_circuit_and.ail`, `short_circuit_or.ail`, `string_replace.ail`

`string_replace.ail` was last touched 2026-03-30 — *after* the operator became list-only —
so it has been shipping broken. None of the 8 appear in `examples/examples_report.json` or
`examples/STATUS.md`, which is exactly why they rotted undetected.

**This is what actually caused the user's Friction #3.** A 0→1 user reaches for an example,
copies its `++` string-concat pattern, and it fails to type-check — triggering their "rewrite
~30 string operations" pain. The error message is good; the *learning material* lied by
demonstration.

**Fix (required):** migrate all 8 examples to `${…}` interpolation / `concat([...])` /
`join(sep, parts)`. **Fix (required, process):** close the verification gap so a broken
`examples/*.ail` fails CI — these 8 are not in the report/manifest at all, so whatever
`make verify-examples` covers, it isn't gating them. Every example under `examples/` (minus
`archive/`, `expected_fail/`) should be run-or-typecheck gated; rotted examples are the most
damaging docs we have because users trust them over prose. (See `test-coverage-guardian` /
`docs-sync` skills for the sweep.)

**Note:** `concat_String` still exists as a registered builtin
([internal/builtins/registry.go:196](../../internal/builtins/registry.go#L196)) — it backs
the `concat()`/`join()` stdlib functions — but the `++` *operator* deliberately no longer
lowers to it. Do not "fix" this by re-enabling `++` on strings; that would revert
M-CONCAT-DISAMBIG.

### 2. `nth_or` / safe-indexing helpers (cheap stdlib add)

`nth` correctly returns `Option[a]` ([std/list.ail:120-126](../../std/list.ail#L120-L126)) —
a defensible safety choice the user agreed with. But every call site needs a match/unwrap;
they wrote their own `nth_i(lst, i, default)` and used it 6×.

**Fix:** ship the boilerplate-killer in `std/list`, preserving safety:

```ailang
export pure func nth_or[a](xs: [a], idx: int, default: a) -> a =
  match nth(xs, idx) { Some(v) => v, None => default }

export pure func head_or[a](xs: [a], default: a) -> a = nth_or(xs, 0, default)
```

Document `nth`'s `Option` return prominently under "list indexing" with the `nth_or` pointer.
(Consider a small family: `nth_or`, `head_or`, `last_or` — but `nth_or` alone removes the
reported pain.)

### 3. File extension `.ail` is canonical — purge `.ailang` from docs

`.ailang` is rejected: a warning fires
([cmd/ailang/main_run_exec.go:126](../../cmd/ailang/main_run_exec.go#L126),
[cmd/ailang/compile.go:100](../../cmd/ailang/compile.go#L100)) and the module resolver only
ever appends `.ail` ([internal/module/resolver.go:106-190](../../internal/module/resolver.go)),
so imports of a `.ailang` file genuinely fail. The user used `.ailang` because **early
docs/tutorials show it**.

**Fix (required, docs):** grep all docs/tutorials/onboarding for `.ailang` and standardize on
`.ail`. **Fix (recommended, code):** upgrade the warning to a precise error —
`"AILANG source files must use the .ail extension (got .ailang)"` — so the first-run failure
self-explains. (We deliberately do **not** start silently accepting `.ailang`: one canonical
extension keeps tooling/module resolution unambiguous.)

### 4. `pure func` vs `func` — promote to getting-started (docs)

The pure/effectful split is a praised feature ("machine-verified purity") but is inferred
from examples, not taught. Add a short, early "Effects & purity" section to the language
tour / getting-started and to the agent prompt: when to write `pure func`, when effects
must appear in the signature (`! {IO}`), and that `pure` is enforced by the type system.

### 5. Module-level `let` IS supported — just say so (docs)

The user defined constants as zero-arg `pure func` because the docs only show `let … in …`
in expression position. But top-level `let name : T = expr` works today —
[tests/codegen-harness/let_binding.ail:7,12](../../tests/codegen-harness/let_binding.ail#L7),
[examples/deriving_eq.ail:12-36](../../examples/deriving_eq.ail#L12-L36) (with type
annotations). **Fix:** add a "module-level bindings" line to the language reference confirming
`let name : T = expr` is valid at module scope.

### 6. Which stdlib modules need which capability (docs)

`std/rand` requires `Rand` ([std/rand.ail](../../std/rand.ail), all fns `! {Rand}`); the user
built a hand-rolled LCG to stay within their `--caps IO,Clock` spec rather than discover the
flag. The capability design is right; discoverability is the gap.

**Fix:** add a "capabilities required" line to every effectful stdlib module's docs
(`rand → Rand`, `io → IO`, `fs → FS`, `clock → Clock`, `net → Net`, …), and surface it in the
MCP `stdlib_module`/`effects_catalog` output so agents see it before writing code. A
capability-by-module table in the effects guide closes this for both humans and agents.

### 7. stdin: clarify what exists vs what doesn't (docs)

The report says "no stdin." That is too strong: `readLine()` (blocking, line-buffered stdin)
exists today ([std/io.ail:17](../../std/io.ail#L17)). What's missing is **real-time /
non-blocking / raw-mode keyboard input** — tracked in #231 (framed there as cancellation
during `std/ai.step`). **Fix:** in the capability/limitations docs, state precisely:
"`readLine()` reads a line from stdin (blocks on Enter); non-blocking/raw keyboard input is
not yet supported (#231)." That lets a developer choose an architecture (self-playing,
poll-on-newline, etc.) up front instead of discovering the wall mid-build.

---

## Acceptance Criteria

- [ ] All 8 `++`-on-strings examples migrated to interpolation/`concat`/`join` and run clean;
      every non-archive `examples/*.ail` is gated by example verification (so the next rot fails CI).
      The `onboarding_guide`/`prompt` keep teaching "`++` is list-only" — that part is correct.
- [ ] `nth_or` (and `head_or`) exported from `std/list`, with tests and a doc note on `nth`'s
      `Option` return.
- [ ] Zero occurrences of `.ailang` in docs/tutorials; non-`.ail` files produce a clear error.
- [ ] Getting-started has an "Effects & purity" section; language reference documents
      module-level `let name : T = expr`.
- [ ] Every effectful stdlib module's docs (and MCP output) names its required capability.
- [ ] Limitations/capability docs state the precise stdin status (readLine yes, raw keyboard no).

## Why this is worth a day

Six of these are doc-or-one-helper changes with no risk to the type system or runtime, and
they target the exact moments the user lost time. Two (the 8 broken `++` examples and the
`.ailang` tutorials) are cases where our own teaching material is *wrong by demonstration*,
which is strictly worse than missing docs — those are the first to fix. The agent-facing MCP
tools (`onboarding_guide`, `stdlib_module`,
`effects_catalog`, `prompt_get`) are the highest-leverage place to land them, since every AI
harness reads those before writing a line.
