# M-IO-READLINE-EOF-OPTION — `readLineOpt() -> Option[string]`: EOF is `None`, not `""`

**Status**: Planned
**Target**: v0.52.0
**Priority**: P1 (correctness: a documented consumer pattern — NDJSON-over-stdin — is unimplementable without a heuristic)
**Estimated**: 2 days (1 day implementation + tests, 0.5 day cross-cutting registrations, 0.5 day docs/prompt/example)
**Dependencies**: none. `std/option` (Option/Some/None) shipped since v0.1.x; `Option`-returning builtins have precedent (`_regex_find_first`, v0.30.0).
**Source**: coordinator task `task-98827762` (directive, firestore) — stapledons-godot **AI.4 evaluation finding**. Reproduced on binary v0.51.0 (b99dd25).

---

## Problem Statement

`IO.readLine` (`internal/effects/io.go` `ioReadLine`) cannot distinguish **end of input** from a
**blank line**. On `io.EOF` it returns whatever was read (`""` when nothing was) with **no error**
(internal/effects/io.go:105–110), and a blank input line also returns `""`. Every stdin-line
consumer must therefore pick a broken strategy:

1. **Treat `""` as EOF** — a stray blank line terminates the service early.
2. **Skip `""`** — at real EOF, `readLine()` returns `""` forever, so the loop spins.

Both were reproduced on the shipped binary v0.51.0 (b99dd25):

```bash
# Program: let l = readLine(); if l == "" then () else { println(l); loop() }
$ printf 'a\n\nb\n' | ailang run --caps IO repro.ail
a                 # ← the blank line after "a" is treated as EOF; "b" is never read

# Program: let l = readLine(); if l == "" then loop() else { println(l); loop() }
$ printf 'a\n\nb\n' | ailang run --caps IO spin.ail
a
b
Error: execution failed: RT_REC_003: max recursion depth 10000 exceeded. ...
```

**Root cause (read from the code)**: `reader.ReadString('\n')` returns `(line, io.EOF)` at end of
input; `ioReadLine` swallows the `io.EOF` and returns the partial line as a plain `String`
(internal/effects/io.go:104–110). The signal "no more input" exists at the Go layer and is
discarded at the AILANG layer. The async path already does this right —
`asyncReadStdinLines` (`internal/effects/stream_stdin.go:46–70`) ends its goroutine at EOF and
the channel close is the EOF signal — but the **synchronous** `readLine` has no way to surface it.

**The flawed convention is taught to AI models**: the current teaching prompt
(`cmd/ailang/prompts/v0.16.6.md:461`) says *"readLine reads one line, returns `""` at EOF"*,
and every version since v0.8.0 says the same — 13 files in `cmd/ailang/prompts/` (mirrored in
`prompts/` and `docs/docs/prompts/`, 37 files total). Every AI-generated stdin consumer inherits
the trap — this is why it surfaced as an evaluation finding, not a bug report.

**Impact** (who is affected):
- **AI-generated NDJSON services** (the stapledons-godot finding): a service reading
  newline-delimited JSON from stdin cannot terminate exactly at EOF. The shipped workaround
  (stapledons-godot `ai/service.ail`) skips blank lines and stops after **16 consecutive empty
  reads** — a heuristic that false-terminates on 16 consecutive blank lines and wastes 16 reads
  at real EOF.
- **Any line-oriented pipe consumer**: grep-like filters, line counters, REPL-style drivers. All
  must embed counters or sentinel conventions that are invisible in the type.
- **Effect-typed language credibility**: AILANG's contract is that effects are explicit and
  machine-decidable (A3/A7); "EOF" is the one input condition that is currently undecidable.

## Goals

**Primary Goal:** Give stdin consumers a read that distinguishes EOF from every possible line
content (including `""`), as a single additive builtin — `readLineOpt() -> Option[string] ! {IO}`,
returning `None` **iff** input is exhausted.

**Success Metrics:**
- The repro program, rewritten with `readLineOpt`, prints `a`, ``, `b` for input `a\n\nb\n` and
  terminates exactly at EOF (no counter, no sentinel).
- A blank line yields `Some("")`; EOF yields `None`; a final unterminated line yields
  `Some(partial)` — identical to `bufio.Scanner`/`asyncReadStdinLines` semantics.
- `readLine` behavior is **byte-identical** (existing tests pass unchanged; zero migration).
- `ailang doctor builtins` green with the new spec registered.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| **Additive `readLineOpt`; `readLine` semantics frozen** | Every prompt version since v0.8.0 (37 files across the three prompt directories) + unknown external consumers (motoko fleet, stapledons-godot runtime) rely on `""`-at-EOF; changing `readLine` to return Option is a breaking change to every prompt and consumer at once | human | design | high |
| **`Option[string]` return (not `Result`, not a sentinel error)** | EOF is "no value", not a failure — matches the established `Option` convention (`std/map.lookup`, `_regex_find_first`); `Result` would invent an error taxonomy for a non-error | human | design | low |
| **`None` ⇔ EOF with zero bytes read; final unterminated line → `Some(partial)`** | Dropping a trailing unterminated line would corrupt NDJSON streams (last record silently lost); matches `bufio.Scanner` and `asyncReadStdinLines` | compiler (semantics) | design | med |
| **Shares the persistent buffered reader with `readLine` (`ctx.GetIOReader`)** | Mixed `readLine`/`readLineOpt` use must not lose buffered bytes; a second reader would steal stdin data | compiler (semantics) | design | med |
| **No new capability; `! {IO}` + IO capability gate unchanged** | Reading stdin is already IO authority; adding a capability would fragment the gate | compiler (semantics) | design | low |
| **`IO.readLineOpt` classified non-deterministic in traces** | Replay tolerance parity with `IO.readLine` (stdin data varies across runs); omitting it makes trace comparison flag legitimate variance | compiler (semantics) | design | low |
| **Teaching prompt updated via a NEW version, not in-place edit** | `check-prompt-freeze` (make/code-health.mk:236) pins each prompt's SHA256; in-place edits fail CI | human | implementation | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Additive `IO.readLineOpt` effect op + `_io_readLineOpt` builtin spec + `std/io.readLineOpt`
      wrapper; `readLine` unchanged (breaking change rejected).
- [x] Signature: `() -> Option[string] ! {IO}` (unit-arg per the S-CALL0 zero-arg convention).
- [x] `None` only when EOF is hit with zero bytes read; `Some(line)` otherwise, with the same
      `\n`/`\r` trimming as `readLine`.
- [x] Uses `ctx.GetIOReader()` (shared, persistent, lazily initialized).
- [x] Repeated calls after EOF keep returning `None` (EOF is sticky, matching `readLine`'s
      sticky `""`).

## Solution Design

### Overview

Add one new IO effect operation, `IO.readLineOpt`, that returns the EOF signal the Go layer
already has. `ioReadLineOpt` is a sibling of `ioReadLine` in `internal/effects/io.go` with the
same reading and trimming logic; the only difference is that `err == io.EOF && line == ""`
maps to the ADT value `None` (from `std/option`) instead of the string `""`, and every other
outcome maps to `Some(trimmedLine)`. Above it sit the three standard registration layers every
`std/io` builtin already has (effects op → `BuiltinSpec` → `std/io.ail` wrapper), plus the
cross-cutting registries that must know the new name (trace non-determinism map, eval-harness
IO auto-import list, builtin docs).

`Option` construction follows the existing `_regex_find_first` precedent exactly
(`internal/builtins/regex.go:196–204`): `&eval.TaggedValue{ModulePath: "std/option",
TypeName: "Option", CtorName: "Some"|"None", Fields: ...}`, and the type is built with
`T.App("Option", T.String())` (`internal/types/builder.go:75`).

### Architecture

**Components:**

1. **Effect op `IO.readLineOpt`** (`internal/effects/io.go`, +~45 LOC): `ioReadLineOpt` — reads
   via `ctx.GetIOReader().ReadString('\n')`; non-EOF errors propagate as errors (fail loudly,
   no silent fallback); `EOF && line == ""` → `None`; otherwise trim and return `Some(line)`.
   Registered via `RegisterOp("IO", "readLineOpt", ioReadLineOpt)`.
2. **Builtin spec `_io_readLineOpt`** (`internal/builtins/io.go`, +~35 LOC): `BuiltinSpec` with
   `Module: "std/io"`, `NumArgs: 1` (unit-arg, S-CALL0 convention — same as `_io_readLine`),
   `Effect: "IO"`, `Type: () -> Option[string] ! {IO}` via
   `T.Func(T.Unit()).Returns(T.App("Option", T.String())).Effects("IO")`, Impl routing through
   `effects.Call(ctx, "IO", "readLineOpt", nil)` for trace recording. Includes `BuiltinMetadata`
   (description, `Since: v0.52.0`, `StabilityStable`, tags) so `ailang doctor builtins` and the
   builtin docs pick it up.
3. **Std wrapper** (`std/io.ail`, +~5 LOC): `import std/option (Option)` +
   `export func readLineOpt() -> Option[string] ! {IO} = _io_readLineOpt()`.
4. **Cross-cutting registrations** (1–2 lines each):
   - `internal/trace/schema.go` `nonDeterministicOps`: add `"IO.readLineOpt": true` (parity with
     `IO.readLine`, schema.go:135).
   - `internal/eval_harness/normalize.go` `needsIO` list (normalize.go:118): add
     `"readLineOpt("` and `"readLineOpt "` so eval-generated code using `readLineOpt` still gets
     `std/io` auto-imported.
5. **Docs + teaching** (see Implementation Plan Phase 3).

**Why the interpreter path needs no other wiring** (read from the code): the module runtime
resolves `_`-prefixed references through `moduleGlobalResolver` → `runtime.builtins.Get`
(`internal/runtime/resolver.go:63–67`), and `BuiltinRegistry.registerFromSpecRegistry` walks
`builtins.AllSpecs()` (`internal/runtime/builtins.go:117–121`) — so a registered `BuiltinSpec`
is automatically callable from any module that imports `std/io`. The legacy
`eval.Builtins` map (`internal/eval/builtins_io.go`) is a deprecated path
(`internal/eval/builtins_call.go:5–8`) and needs **no** entry — `_io_flush`, `_io_writeBytes`,
`_io_printErr`, `_io_eprintln` are absent from it and work fine.

### Implementation Plan

**Phase 1: Language surface** (~4 hours)
- [ ] `ioReadLineOpt` + `RegisterOp` in `internal/effects/io.go` (mirror `ioReadLine`'s arg
      validation and trimming; EOF branch → `None`).
- [ ] `_io_readLineOpt` `BuiltinSpec` in `internal/builtins/io.go` (unit-arg validation, type,
      metadata).
- [ ] `std/io.ail`: `import std/option (Option)` + `readLineOpt` wrapper.
- [ ] `internal/trace/schema.go`: non-deterministic op entry.
- [ ] `internal/eval_harness/normalize.go`: `needsIO` entries.
- [ ] `ailang doctor builtins` green; `ailang check` on a probe module using
      `match readLineOpt() { Some(l) => ..., None => ... }`.

**Phase 2: Tests** (~4 hours)
- [ ] `internal/effects/io_test.go`: sibling tests —
      `TestIOReadLineOpt_BlankLine → Some("")`, `TestIOReadLineOpt_EOF → None`,
      `TestIOReadLineOpt_FinalUnterminatedLine → Some(partial)`,
      `TestIOReadLineOpt_CRLF → Some("a")`, `TestIOReadLineOpt_MissingCapability`,
      `TestIOReadLineOpt_MixedWithReadLine` (buffer sharing), `TestIOReadLineOpt_PostEOFStickyNone`.
- [ ] Integration: pipe-driven `ailang run --caps IO` — `printf 'a\n\nb\n'` prints `a`, ``, `b`
      then terminates at EOF; empty stdin terminates immediately.
- [ ] Regression: `TestIOReadLine_EOF` (asserts `""` on EOF, io_test.go:167–191) and
      `TestIOReadLine_Success` pass **unchanged** — `readLine` is frozen.

**Phase 3: Cross-cutting + docs** (~4 hours)
- [ ] `docs/docs/reference/effects.md`: std/io table rows (readLine row gains a cross-reference;
      new readLineOpt row), line 75 + 469.
- [ ] New prompt version (via prompt-manager flow; **not** an in-place edit —
      `check-prompt-freeze` pins SHA256): teach
      `readLineOpt() -> Option[string]` — `None` at EOF, `Some("")` for blank lines — and mark
      the old `""`-at-EOF readLine teaching as legacy.
- [ ] `examples/runnable/io_readline_eof.ail`: echo loop terminating at EOF via `match`
      (runs headless in `verify-examples`: empty stdin → immediate `None` → clean exit), plus
      manifest entry.
- [ ] Changelog entry under `[Unreleased]` (changelogs/v0.32-current.md).

### Files to Modify/Create

**Modified files:**
- `internal/effects/io.go` (+~45 LOC) — `ioReadLineOpt` + `RegisterOp`
- `internal/builtins/io.go` (+~35 LOC) — `_io_readLineOpt` BuiltinSpec + metadata
- `std/io.ail` (+~5 LOC) — import + wrapper
- `internal/trace/schema.go` (+1 LOC) — non-deterministic op map
- `internal/eval_harness/normalize.go` (+~1 LOC) — `needsIO` list
- `internal/effects/io_test.go` (+~140 LOC) — new sibling tests
- `docs/docs/reference/effects.md` (+~3 LOC) — reference rows
- `changelogs/v0.32-current.md` (+entry) — Unreleased section

**New files:**
- `examples/runnable/io_readline_eof.ail` (~25 LOC) — gated example (verify-examples)

**No changes**: `internal/eval/builtins_io.go` (deprecated path), `internal/vm/*` (effectful
builtins are not wired in bytecode mode — see Non-Goals), `internal/builtins/registry_codegen_io.go`
(`_io_readLine` itself has no compiled-mode spec; parity), `internal/trace/comparator.go`
(comment-only mention).

## Examples

### Example 1: NDJSON-style line loop (the AI.4 case)

**Before (the two broken strategies + heuristic workaround):**
```ailang
-- Strategy 1: blank line kills the service
let l = readLine()
if l == "" then () else { println(l); loop() }

-- Strategy 2: spins at EOF (RT_REC_003 after 10k reads, or hangs if iterative)
let l = readLine()
if l == "" then loop() else { println(l); loop() }

-- Shipped workaround (stapledons-godot ai/service.ail): counter heuristic
-- stop after 16 consecutive empty reads — false-terminates on 16 blank lines
```

**After (verified shape — compiles and runs today with a mock standing in for the builtin):**
```ailang
import std/io (println, readLineOpt)

func loop() -> () ! {IO} {
  match readLineOpt() {
    Some(l) => { println("line: ${l}"); loop() },   -- blank line: Some("")
    None => println("EOF")                          -- end of input: None
  }
}
```
For stdin `a\n\nb\n` this prints `line: a`, `line: `, `line: b`, `EOF` — every line
delivered, exact termination. (Mock-verified: the identical program with
`mockNext(n) -> Option[string]` in place of `readLineOpt()` passes `ailang check` and prints
exactly this — see Verification Log V5.)

### Example 2: composability with std/option

```ailang
import std/io (readLineOpt)

-- Skip blank lines without an EOF trap:
func nextNonBlank() -> Option[string] ! {IO} {
  match readLineOpt() {
    Some("") => nextNonBlank(),   -- real blank line: keep reading
    Some(l) => Some(l),
    None => None                  -- EOF: stop, precisely
  }
}
```

## Conflict Surface

This change touches `internal/effects/` — the section is required. It adds a **name** at five
existing registration positions; it changes **no syntax, no parser production, no type rule**
(all programs parse/typecheck through unchanged machinery — the new symbol is a normal
`std/io` function once the wrapper exists).

### Positions touched and what else lives there

| Position | New occupant | Existing occupants of the same position | Collision check |
|----------|--------------|------------------------------------------|-----------------|
| `RegisterOp("IO", …)` op-name slot (`internal/effects/io.go`) | `readLineOpt` | `print`, `println`, `readLine`, `writeBytes`, `exit`, `flush`, `printErr`, `eprintln` | Distinct name; no op named `readLineOpt` exists (V8) |
| `BuiltinSpec` name slot (`_io_*` namespace) | `_io_readLineOpt` | `_io_print`, `_io_println`, `_io_readLine`, `_io_exit`, `_io_writeBytes`, `_io_flush`, `_io_printErr`, `_io_eprintln` | Spec registry rejects duplicates at registration (spec.go:104); grep confirms absent (V8) |
| `std/io` exported-function name | `readLineOpt` | `print`, `println`, `readLine`, `writeBytes`, `flush`, `printErr`, `eprintln`, `exit` | No `.ail` in repo defines/imports `readLineOpt` (V8); std modules are name-spaced by import |
| `nonDeterministicOps` map key (`internal/trace/schema.go`) | `"IO.readLineOpt"` | `"IO.readLine"`, `Clock.*`, `Net.*` | Additive map key; `IsNonDeterministic` lookups for other pairs unaffected |
| `std/io.ail` import set | `std/option (Option)` | (none — std/io imports nothing today) | `std/option` imports nothing (pure leaf module); no cycle is possible |

### Disambiguation strategy

Not applicable in the parser/typechecker sense: no grammar production or typing rule changes.
For the registries, uniqueness is enforced mechanically (spec-registry duplicate check) and was
verified by grep (V8). The one soft conflict — `std/io.ail` gaining its first import — is
resolved by `std/option` being an import-free leaf module (V11).

### Programs that MUST still work (regression fixtures)

1. `internal/effects/io_test.go` `TestIOReadLine_Success` / `TestIOReadLine_EOF` /
   `TestIOReadLine_MissingCapability` — assert the frozen `readLine` semantics (`""` at EOF),
   byte-identical pre/post.
2. `examples/regex_capture.ail` — exercises the exact Option-returning-builtin + `match` pattern
   this feature reuses (`_regex_find_first` → `Some/None`); runs green today (V6).
3. Every existing `std/io` consumer — spec registration is append-only; `std/io.ail` wrappers
   resolve through `moduleGlobalResolver` → spec registry (V10), which is untouched.
4. The Problem Statement repro (`printf 'a\n\nb\n' | ailang run --caps IO`) — must still print
   only `a` under `readLine` (frozen semantics), and print `a`, ``, `b` + terminate under
   `readLineOpt`.
5. `examples/runnable/stream_multi_source.ail` (asyncReadStdinLines path) — unaffected; the
   stream EOF signal already exists and is untouched.

### What deliberately changes

Nothing existing. No previously-valid program changes meaning: `readLine` is frozen, no syntax
changes, no capability set changes. The only new behaviors are the new names
(`readLineOpt`, `_io_readLineOpt`, `IO.readLineOpt`) which were previously unbound identifiers.

## Success Criteria

- [ ] `readLineOpt() -> Option[string] ! {IO}` available from `std/io`; `None` ⇔ EOF with zero
      bytes read (integration test: empty stdin → `None` immediately).
- [ ] Blank line → `Some("")`; final unterminated line → `Some(partial)`; CRLF → trimmed
      (unit tests, internal/effects/io_test.go).
- [ ] The repro program prints `a`, ``, `b` and terminates exactly at EOF (integration test).
- [ ] `readLine` behavior unchanged — `TestIOReadLine_*` pass without modification.
- [ ] `ailang doctor builtins` green with `_io_readLineOpt` registered.
- [ ] `IO.readLineOpt` in `nonDeterministicOps` (trace replay tolerance parity).
- [ ] Eval harness: `needsIO` recognizes `readLineOpt(` (auto-import parity).
- [ ] `make test` + `make verify-examples` green; new example in the manifest.
- [ ] `docs/docs/reference/effects.md` updated; new prompt version teaching `readLineOpt`
      (prompt-manager flow; freeze gate stays green).
- [ ] Changelog entry under `[Unreleased]`.

## Testing Strategy

**Unit tests** (`internal/effects/io_test.go`, mirroring the existing `ioReadLine` suite):
- `Some`/`None` construction for: normal line, blank line, EOF (empty input), final
  unterminated line, CRLF line, post-EOF sticky `None`, non-EOF read error propagation.
- Capability gate: `readLineOpt` without `--caps IO` → `E_*` capability error (same as
  `readLine`).
- Buffer sharing: interleaved `readLine`/`readLineOpt` calls over one piped reader deliver every
  byte exactly once.

**Integration tests:**
- `ailang run --caps IO` under piped stdin (`printf 'a\n\nb\n'`): full-line delivery + exact
  EOF termination; empty stdin: immediate clean exit.

**Regression-surface tests** (per Conflict Surface fixtures):
- Existing `TestIOReadLine_*` untouched and green (frozen semantics).
- `examples/regex_capture.ail` still runs (Option-builtin match pattern).
- `verify-examples` green with the new `examples/runnable/io_readline_eof.ail` (headless-safe:
  empty stdin → immediate `None` → exit 0).

**Manual testing:**
- The Problem Statement repro, before/after, both `readLine` and `readLineOpt` variants.
- REPL: `readLineOpt()` on closed stdin returns `None` (REPL grants IO caps, repl.go:97).

## Deferred Decisions

- Exact prompt version number and wording for the teaching update — prompt-manager flow,
  human at review (freeze gate requires a new version; content sketch is in Phase 3).
- Whether `std/io.ail` imports `std/option (Option)` only or `(Option, Some, None)` (house
  style in `std/regex.ail` is all three; only the type is needed in the signature — both
  verified to compile, V7) — agent may choose.
- Example file naming/shape beyond the headless-safe constraint — agent may choose.
- `BuiltinMetadata` wording (description/long description/tags) — agent may choose, following
  the `_io_readLine`/`_io_flush` entries' style.
- Whether to also add a Go-codegen stub for `_io_readLineOpt` (`registry_codegen_io.go`
  panics-with-message style) — agent may choose; `_io_readLine` itself has no compiled-mode
  spec, so parity = do nothing (see Non-Goals).

## Non-Goals

- **Changing `readLine` semantics** — frozen for compatibility (every prompt version since
  v0.8.0 teaches the `""` convention; external consumers depend on it). A future deprecation
  cycle, if ever, is a separate doc.
- **`readLineResult` / error-channel variant** — EOF is absence, not failure; `Result` would
  invent an error type for a non-error and cost more syntax (A8) for zero extra information.
- **Fixing other stdin/EOF surfaces** — audited: none share the ambiguity. `asyncReadStdinLines`
  signals EOF by channel close (stream_stdin.go:46–70); `std/fs`, `std/net`, `std/process` have
  no line readers (V9). This doc is the one fix for the one gap.
- **Bytecode/VM path** — effectful builtins are not wired in the VM
  (`internal/vm/vm.go:585–591`: `OpBuiltinTrap` "Phase 2E"); `_io_readLineOpt` inherits
  `_io_readLine`'s status quo. Wiring effectful IO into the VM is a separate lane.
- **Go compiled mode** — `_io_readLine` has no codegen spec either (registry_codegen_io.go);
  parity is "no spec". Adding one is deferred (above).
- **Raw-mode/character reads** — out-of-core by design (docs/LIMITATIONS.md:28).

## Timeline

**Day 1** (~6 hours): Phase 1 — effect op, builtin spec, std wrapper, trace + harness
registrations, doctor/check green (4h); Phase 2 unit tests (2h start).
**Day 2** (~6 hours): Phase 2 finish — integration + regression tests (2h); Phase 3 — docs
reference, example + manifest, changelog (2h); new prompt version via prompt-manager (1h);
buffer (1h).

**Total: ~2 days** (implementation 4h, tests 4h, cross-cutting + docs 4h, buffer).

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Mixed `readLine`/`readLineOpt` use loses buffered bytes | High | Both ops use the same `ctx.GetIOReader()` persistent reader (context.go:683–692); dedicated buffer-sharing test |
| Two code paths drift (trimming, CRLF, EOF-stickiness) | Medium | `ioReadLineOpt` mirrors `ioReadLine`'s exact branch structure; table-driven tests covering both ops with the same inputs |
| `std/io` gains a dependency (`std/option`) | Low | `std/option` is an import-free leaf module — no cycle; several std modules already import it (std/map, std/regex) |
| AI models keep generating the `""`-at-EOF pattern | Medium | New prompt version teaches `readLineOpt` as the loop idiom; `docs/docs/reference/effects.md` cross-references from the `readLine` row |
| Eval-harness generated code uses `readLineOpt` without importing std/io | Low | `needsIO` list updated (normalize.go:118) — auto-import parity test |

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | EOF becomes a deterministic, decidable signal; removes the nondeterministic 16-empty-reads heuristic from consumer code |
| A2: Replayability | 0 | No trace-schema change; op added to the same non-deterministic tolerance list as `readLine` |
| A3: Effect Legibility | +1 | Same explicit `! {IO}` effect and capability gate; the one hidden input condition becomes legible in the return type |
| A4: Explicit Authority | 0 | No new capability; stdin stays behind the existing IO capability |
| A5: Bounded Verification | +1 | EOF decidability is local — one `match`, no probe loops or counters |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Machine-decidable `None` replaces sentinel-string conventions and retry counters; pattern match is structured |
| A8: Minimal Syntax | 0 | No new syntax — one named function reusing existing ADT machinery |
| A9: Cost Visibility | 0 | No resource-cost surface change |
| A10: Composability | +1 | Return type composes with std/option (`map`, `getOrElse`, `isSome`, …) |
| A11: Structured Failure | +1 | Absence is structured (`None`), not an overloaded sentinel value; genuine read errors still fail loudly |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +7** → **Decision: Move forward** ✅

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism introduced
- [x] A3 (Effects): no hidden side effects
- [x] A4 (Authority): no ambient access granted
- [x] A7 (Machines First): not optimizing for human convenience over machine analysis

## Verification Log

All commands run on 2026-10-02 against the shipped binary `ailang` v0.51.0 (b99dd25, the
binary named in the finding) unless noted; repo at d980558a.

| # | Claim | How verified | Result |
|---|---|---|---|
| V1 | `readLine` returns `""` at EOF with no error (the gap) | read `ioReadLine`, internal/effects/io.go:104–110 | Confirmed |
| V2 | Treat-`""`-as-EOF drops trailing lines (repro prints only `a`) | `printf 'a\n\nb\n' \| ailang run --caps IO repro.ail` → output `a`, exit 0 | Reproduced |
| V3 | Skip-`""` spins at EOF | `… spin.ail` (loop on `""`) → prints `a`,`b` then `RT_REC_003: max recursion depth 10000 exceeded` | Reproduced (recursive form errors after 10k no-op reads; iterative form would hang) |
| V4 | Current teaching prompt teaches `""`-at-EOF | `cmd/ailang/prompts/v0.16.6.md:461` ("readLine reads one line, returns \"\" at EOF"); `grep -rl 'returns "" at EOF' cmd/ailang/prompts/ prompts/ docs/docs/prompts/` → 37 files (13 in cmd/ailang/prompts, current + 12 earlier, since v0.8.0; mirrored in two doc dirs) | Confirmed |
| V5 | The proposed `match readLineOpt() { Some/None }` loop shape is valid AILANG | mock program (`mockNext(n) -> Option[string]` + identical match/loop) passes `ailang check` and `ailang run --caps IO` prints `line: a`,`line: `,`line: b`,`EOF` | Confirmed |
| V6 | Builtin-returned `Option` is matchable end-to-end (precedent) | `examples/regex_capture.ail` runs green (`ailang run --relax-modules --caps IO`, exit 0) — matches on `_regex_find_first`'s `Option[RegexMatch]` | Confirmed |
| V7 | `-> Option[string]` signature + `import std/option (Option)` compiles | `ailang check` on probe modules (signature-only import and Some/None construction) → `✓ No errors found!` | Confirmed |
| V8 | No existing `readLineOpt`/`readLineResult`/`_io_readLineOpt` anywhere | `grep -rn … internal/ std/ docs/ examples/ design_docs/planned/ changelogs/` → no hits (exit 1) | Confirmed (absent) |
| V9 | No other line-reader shares the EOF ambiguity | grepped `std/process.ail`, `std/net.ail`, `std/fs.ail` (no line readers); `asyncReadStdinLines` EOF = channel close (internal/effects/stream_stdin.go:46–70) | Confirmed — sync `readLine` is the only gap |
| V10 | New `BuiltinSpec` auto-resolves for importers; legacy `eval.Builtins` map not needed | `internal/runtime/resolver.go:63–67` (`_`-prefixed → `runtime.builtins.Get`), `internal/runtime/builtins.go:117–121` (`registerFromSpecRegistry` walks `AllSpecs`); `_io_flush`/`_io_writeBytes` absent from eval.Builtins yet working; `internal/eval/builtins_call.go:5–8` marks the path deprecated | Confirmed |
| V11 | `std/option` is an import-free leaf module (no import cycle from std/io) | `std/option.ail` head: no import declarations; `std/map.ail`, `std/regex.ail` already import it | Confirmed |
| V12 | `Option` value construction + type-builder precedent | `internal/builtins/regex.go:196–204` (`TaggedValue{ModulePath:"std/option", TypeName:"Option", CtorName:"Some"|"None"}`), type `T.App("Option", …)` regex.go:134; `internal/types/builder.go:75` (`App`), :211 (`Effects`) | Confirmed |
| V13 | Unit-arg convention for zero-arg builtins (S-CALL0) | `internal/builtins/io.go` `_io_readLine` spec: `NumArgs: 1`, unit validation, comment "FIXED (v0.4.2)… Zero-arg builtins now take unit" | Confirmed |
| V14 | Shared persistent stdin reader across `readLine` calls | `internal/effects/context.go:683–692` (`GetIOReader` lazily init, reused, "buffered data is preserved between readLine() invocations") | Confirmed |
| V15 | `IO.readLine` is classified non-deterministic for traces | `internal/trace/schema.go:135` `nonDeterministicOps["IO.readLine"]`; comparator honors it (comparator.go:144–152) | Confirmed |
| V16 | Eval harness auto-imports std/io via the `needsIO` name list | `internal/eval_harness/normalize.go:118` `ioFuncs = {"print(", "println(", "readLine(", …}` | Confirmed |
| V17 | Existing tests assert the frozen `readLine` ""-at-EOF semantics | `internal/effects/io_test.go:167–191` `TestIOReadLine_EOF` expects `""`, no error | Confirmed |
| V18 | Prompts are SHA-pinned; in-place edit fails CI | `make/code-health.mk:236–238` `check-prompt-freeze: ailang prompt freeze --check` ("pins each prompt's SHA256") | Confirmed |
| V19 | Effectful builtins are unwired in bytecode mode (VM status quo) | `internal/vm/vm.go:585–591` `OpBuiltinTrap` → "not yet wired (Phase 2E)"; `OpEffectTrap` → not implemented | Confirmed |
| V20 | `_io_readLine` has no Go-codegen spec (compiled-mode parity) | `internal/builtins/registry_codegen_io.go` — only `_io_println`/`_io_print` have helpers; `_io_readLine` absent | Confirmed |
| V21 | No example in the gated corpus uses `readLine` (new example is the fixture) | `grep -rln readLine examples/ tests/` → only build-cache artifacts, no `.ail` source | Confirmed |
| V22 | Version facts: current released v0.51.1 (2026-10-02); target folder v0_52_0 exists | `std/VERSION` = v0.51.1; `changelogs/v0.32-current.md` `[v0.51.1] - 2026-10-02`; `ls design_docs/planned/v0_52_0/` (2 docs already target it) | Confirmed |
| V23 | Builtin registry validated by doctor | `ailang doctor builtins` → "✅ All builtins are valid!" | Confirmed |

*Duplicate/coverage gate*: related-doc search (SimHash + neural) was run on query
"io readline eof option". Neural mode fell back to SimHash (Ollama embeddings unavailable in
this environment — "model: fallback-simhash" in output), and its top matches
(m-script-invoke, M-AI-OPENROUTER, M-TASK-HIERARCHY; planned: m-v1-simplification-s4-sprint-plan,
m-vm-determinism) are unrelated to stdin/EOF/Option — no doc ≥ 0.45 relevance on this topic.
No planned or implemented doc covers line-reading EOF disambiguation. Note: the
`create_planned_doc.sh` search step aborts on GNU grep (its `grep -E "^\d+\."` pattern is
literal-`d` on Linux, so `merge_results` exits 1 under `pipefail` and `set -e` stops the
script) — the doc scaffold was created via the script's no-ailang fallback path, with the
search performed manually as above.

## Related Documents

**Implemented (inform this design):**
- [m-vm-match-lowering](../../implemented/v0_51_1/m-vm-match-lowering.md) — `Some(Some(x))`
  and nested Option patterns match in bytecode mode (v0.51.1)
- [v0_4_1_m-s-call0-zero-arg-builtin-bug](../../archive/v0_4_1_m-s-call0-zero-arg-builtin-bug.md)
  — the unit-arg convention for zero-arg builtins this design follows
- [m-async-io-process-stdin](../../implemented/v0_9_0/m-async-io-process-stdin.md) (M-ASYNC-IO,
  v0.9.0) — `asyncReadStdinLines`: the channel-close EOF signal this design mirrors
  synchronously

**Planned (checked for overlap — none):**
- [m-ifc-declared-record-labels](./m-ifc-declared-record-labels.md) and
  [m-mcp-oauth-package](./m-mcp-oauth-package.md) — the other v0.52.0 docs; unrelated surfaces

## References

- **Finding**: stapledons-godot AI.4 evaluation — NDJSON service cannot distinguish blank line
  from EOF; workaround: skip blank lines, stop after 16 consecutive empty reads
  (`ai/service.ail`). Coordinator task `task-98827762`.
- **Repro**: `printf 'a\n\nb\n' | ailang run --caps IO prog.ail` (see Verification Log V2/V3).
- [Design Axioms](/docs/references/axioms) — the 12 non-negotiable principles
- [docs/LIMITATIONS.md](../../docs/LIMITATIONS.md) — input-surface boundaries (line input works;
  raw keypress out-of-core)
- [effects reference](../../docs/docs/reference/effects.md) — std/io effect table to update

## Future Work

- Prompt deprecation cycle for the `""`-at-EOF `readLine` teaching once `readLineOpt` has been
  the taught idiom for a few versions (prompt-manager lane).
- Wiring effectful IO builtins into the bytecode VM (Phase 2E lane) — `readLineOpt` inherits
  whatever `readLine` gets there.
- Go compiled-mode helper for stdin reads if `ailang build` consumers ask for it
  (`registry_codegen_io.go` stub table).
- `std/stream`-style line source over an arbitrary reader handle (generalizing beyond stdin) —
  only if a consumer appears (YAGNI for now).
