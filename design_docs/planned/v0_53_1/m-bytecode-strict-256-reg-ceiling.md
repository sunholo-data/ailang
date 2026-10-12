# M-BYTECODE-STRICT-256-REG-CEILING — strict-VM register-pressure failures are silent until RUN time: >256-element list literals and if/else-if chains over wide records become `evaluator-only` with no compile-time, `check`-time, or non-strict-run warning

**Status**: Planned
**Target**: v0.53.1
**Priority**: P0 (strict-VM coverage silently degrades for exactly the AI-generated code shapes `--strict-bytecode` exists to cover; reported from the field twice in one mission)
**Estimated**: 3–4 days (~450 LOC, pure compiler + CLI surface; no VM, no parser, no language semantics change)
**Dependencies**: None (touches `internal/bytecode/compiler/`, `internal/runner/`, `cmd/ailang/`; no in-flight siblings)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Removes a class of engine divergence: today the evaluator runs a function the strict VM refuses, so `ailang run` and `ailang run --bytecode --strict-bytecode` disagree on the same program (one succeeds, one traps). After the fix, both reported shapes run identically on both engines. |
| A2: Replayability | 0 | No trace/replay surface change. |
| A3: Effect Legibility | 0 | All affected constructs are pure functions; no effect annotations change. |
| A4: Explicit Authority | 0 | No capability surface change. |
| A5: Bounded Verification | +1 | `ailang check --strict-bytecode` (new) gives CI a *compile-time*, machine-readable gate for strict-VM coverage — previously verifiable only by running and watching for a CALL trap. |
| A6: Safe Concurrency | 0 | No concurrency change. |
| A7: Machines First | +1 | The two failure shapes are exactly what AI generators emit: flat LUT literals (601 floats) and long else-if dispatch over a wide record (16 fields). The human workaround (manual chunking, splitting `applyOverride` into two functions) is precisely the machine-friendly friction this removes. |
| A8: Minimal Syntax | +1 | No new syntax; fixes lowering/compilation of already-taught constructs that already type-check. |
| A9: Cost Visibility | 0 | No resource semantics change. (Chunked concat copies are noted in Solution Design §B1.) |
| A10: Composability | 0 | Fix is internal to one compiler module + CLI reporting. |
| A11: Structured Failure | +1 | A late, mid-execution runtime trap (`vm: CALL: ... is evaluator-only`) becomes an early, coded, positioned compile diagnostic (`BC001`) with the function name, file:line, and allocator reason — and non-strict runs get a visible warning instead of a silent fallback (a direct "no silent fallbacks" guardrail violation today). |
| A12: System Boundary | 0 | No boundary change. |

**Net Score: +5** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): removes implicit nondeterminism between engines (no new nondeterminism introduced)
- [x] A3 (Effects): no hidden side effects
- [x] A4 (Authority): no ambient access granted
- [x] A7 (Machines First): fixes a machine-facing gap; optimizes nothing for human convenience

## Problem Statement

Reported from stapledons-godot (v0.52.0, bf2436a) twice in one mission:

1. A pure function returning a list literal of 601 floats (`pkg/stapledons/sim/data/ism.efficacyLut`) — and, in the minimal form, `[0.0, … 300 elements]` — compiles, type-checks, and runs on the evaluator, but on `ailang run --bytecode --strict-bytecode` fails **at CALL time**, mid-execution of `main`:
   `vm: CALL: pkg/stapledons/sim/data/ism.efficacyLut is evaluator-only (register allocator: contiguous block of 601 would exceed 256) but no interop bridge is wired`
2. A long if/else-if chain of record updates over a 16-field record (`sim/protocol.ail applyOverride`):
   `applyOverride is evaluator-only (register allocator: contiguous block of 16 would exceed 256)`

**Reproduced and measured on v0.52.5** (transcripts in Verification Log):

- 601-element float list literal: `ailang check` → `✓ No errors found!`; `ailang run --bytecode` (non-strict) → prints `601`, **exit 0, no warning** (silent bridge fallback); `ailang run --bytecode --strict-bytecode` → the reported CALL trap, exit 1.
- 16-field record, else-if chain of full-record updates: **15 arms pass, 16 arms fail** with `contiguous block of 16 would exceed 256`. The failure threshold is a property of *nesting depth × record width*, not of program semantics — the exact same 16-arm function is fine on the evaluator.

### Why the failure is invisible until run time

The register allocator error is a **compile-time** error (`internal/bytecode/compiler/regalloc.go:66-67`), but per-function compile failures never fail the image: `internal/bytecode/compiler/compiler.go:120-176` (M-BYTECODE-2D M3 / M-BYTECODE-BATCH) rolls the prototype back and tags it `EvalOnly` with the compile error as `EvalReason`, **emitting no diagnostic**. At run time, `internal/runner/vm.go:206-222` wires the evaluator bridge in non-strict mode (silent; nothing is printed) and intentionally leaves it unwired in strict mode, so the first human-visible signal is a VM CALL trap deep in the callee's caller (`internal/vm/vm.go:289`).

`ailang check` has **no flag that compiles bytecode at all** (verified: `ailang check --help` — `--debug-compile` is phase timing; nothing runs `CompileBytecodeFromResult`), and `ailang test --strict-bytecode` fails loudly only when the **test body itself** is evaluator-only (`internal/testing/bytecode_engine.go:157`) — an evaluator-only *helper* still traps mid-run exactly like `run` does.

### Root causes (both code-verified)

**Cause A — oversized contiguous block (list literal):** `compileListLit` (`internal/bytecode/compiler/collections.go:101`) and `compileTupleLit` (:125) call `allocContig(n)` for all n elements in one contiguous register block. For n > 256 the bump path errors. This ceiling is architectural to the frozen instruction encoding: `EncodeABC` operands are 8-bit (`internal/bytecode/opcode.go:191-198`) and `OpMakeList`/`OpMakeTuple` read `R[B..B+C-1]` with `C: uint8`, so a single instruction can never materialize more than 255 elements regardless of registers. `OpConcat` (`R[A] = R[B] ++ R[C]`, opcode.go:58; vm.go:244) already exists — chunked lowering needs **no new opcode**.

**Cause B — per-level register accumulation in else-if chains:** `compileIfExpr` (`internal/bytecode/compiler/control_flow.go:92-141`) allocates one fresh **result register per nesting level before compiling the else branch**, so a chain of k arms keeps k result registers live. Worse, each level's result register is taken (LIFO) from the top of the 16-register free run the previous arm just released, so `findContigInFreeList` can never find a 16-run at level ≥ 2 and every level **bumps 16 fresh registers** (`regalloc.go:66`). Arithmetic: 2 pinned params + level 1 bump (regs 3–18) + 15 more levels × 16 = 243 live at the start of level 16, whose 16-block would end at 259 > 256 — matching the measured 16-arm failure threshold exactly. The same accounting explains the field report: `applyOverride` on a 16-field record (each arm compiles a full-record `MAKE_RECORD`, needing a 16-block, `collections.go:378-396`).

The evaluator has none of these limits (tree-walking, no register model), so the constructs are legal AILANG — this is strictly a strict-VM coverage and diagnosability gap, not a language bug.

### Impact

- **AI code generators** (primary users) emit both shapes naturally: flat data tables and else-if dispatch. The language accepts them, then the strict VM — the determinism gate the whole program leans on — refuses them at runtime with an error that names neither a source line nor a fix.
- **Field cost:** stapledons-godot worked around it by hand (chunk LUTs to 100 and concatenate; split `applyOverride` into `applyOverride2`) — silent workarounds in `sim/tools/ism_build.ail` and `sim/protocol.ail` that a compiler fix makes unnecessary.
- **Silent fallback violates the repo's own guardrail:** non-strict `--bytecode` prints the right answer with no signal that part of the program ran on the evaluator, i.e. strict-VM coverage degraded without anyone knowing (same degradation class as M-BYTECODE-NESTED-PATTERN-LOWERING, which called it "the exact list-walking code --strict-bytecode exists to cover").

## Goals

**Primary Goal:** Make both reported shapes (and their size generalizations) compile to bytecode and run correctly under `--strict-bytecode`, and make *any* remaining `EvalOnly` tagging visible at `ailang check` time with a structured diagnostic instead of a runtime trap.

**Success Metrics:**
- 601-element (and 1000-element) list literals and tuple literals compile and produce the evaluator's exact output under `--bytecode --strict-bytecode`.
- A 60-arm if/else-if chain of record updates over a 16-field record compiles and matches the evaluator under `--strict-bytecode` (measured today: fails at 16 arms).
- `ailang check --strict-bytecode` exits ≠ 0 with one `BC001` diagnostic per `EvalOnly` prototype (function name, file:line, reason) on any module that would degrade; exits 0 on the two fixed repros.
- `ailang run --bytecode` (non-strict) prints a stderr warning listing `EvalOnly` prototypes instead of falling back silently.
- Zero new VM opcodes; zero instruction-encoding changes; all existing bytecode goldens and engine-parity tests unchanged (except new fixtures).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Surface `EvalOnly` prototypes at `ailang check --strict-bytecode` as errors (new flag) rather than adding diagnostics to plain `ailang check` | Plain `check` must not start failing programs that run fine on the evaluator (non-strict default); the strict gate is opt-in, mirroring `run --strict-bytecode` naming | human | design | med |
| Fix the two shapes in the **compiler** (chunked lowering + if-expr result reuse); **do not** raise the 256-register ceiling or widen instruction operands | The ceiling is baked into the frozen 8-bit ABC encoding (opcode.go:191-198) — raising it is a VM re-freeze (core-floor lane), not an AILANG fix | human | design | high (avoided) |
| New diagnostic code `BC001` in a new `BC` family (not `MOD016`) | The diagnostic is a bytecode-compile-phase finding, not a module-structure error; both `BC001` and `MOD016` verified unallocated (grep, Verification Log V9) | agent | design | low |
| Chunk size K for collection lowering is a compile-time constant (default 64), not user-configurable | No new env var / flag surface (simplicity gates); 64 leaves 192 registers of headroom for element subexpressions and callers' live sets | agent | implementation | low |
| `EvalOnly` prototypes keep their reason string as the diagnostic detail (extend the report with the FuncDecl's `Line`) rather than re-running any analysis | FuncDecl already carries `File`/`Line` (`internal/gen/stmt/stmt.go:67-81`); no new analysis pass is needed | agent | implementation | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Fix lane: compiler-only (chunking + chain fix + surfacing); no VM, parser, typechecker, or stdlib changes.
- [x] Strict `check` gate is a new `--strict-bytecode` flag (naming aligned with `run`/`test`); plain `check` output unchanged.
- [x] `BC001` allocated in a new `BC` family; `MOD016` left for the next module-lane error.
- [x] The full-rebuild record-update path (`compileRecordUpdate` via `MAKE_RECORD`, sorted fields) is NOT switched to `OpUpdateRecord` in this sprint — see Deferred Decisions (the sorted-field invariant makes it non-trivially risky, and the chain fix removes the reported pressure without touching record-update codegen).
- [ ] Sprint-planner confirms the strict-VM coverage corpus sweep (how many `std/`, `examples/`, `goldens/` protos are `EvalOnly` today and which of the two causes account for them) lands as the first sprint task, since it sizes the residual after Phases A–B.

## Conflict Surface

This design touches `internal/bytecode/compiler/` (codegen) and `cmd/ailang/` + `internal/runner/` (CLI reporting). Required enumeration per the design-doc gate:

1. **Syntactic/semantic positions extended:** none. No parser or typechecker change; both fixed shapes are already accepted constructs.
2. **Codegen positions extended:**
   - `compileListLit` / `compileTupleLit` (collections.go:90-145): today always one `allocContig(n)` + one `MAKE_LIST`/`MAKE_TUPLE`. After: n ≤ K keeps the existing single-instruction path **byte-identical**; n > K emits a chunk fold.
   - `compileIfExpr` (control_flow.go:92-141): today one fresh result register per level; after: an else-position `IfExpr` reuses the outer result register (new internal helper `compileIfExprInto`). All other expressions unchanged; `compileIfStmt` (statement form) untouched — it has no per-level result register.
3. **Other valid constructs already living in those positions (audited — every `allocContig` call site):**
   - `compileCall` (call.go:36) and tail-call (call.go:246): block of arity+1. Calls with arity > 255 are already impossible to encode (`B: uint8`); arity 100–255 compiles today and must keep working (chunking does not touch call frames).
   - `compileRecordLit` (collections.go:188) and `compileRecordUpdate` (:378): blocks of field-count. Record literals/updates with ≤ 255 fields must keep working; > 255-field records remain `EvalOnly` (now *reported* at check time — see Non-Goals).
   - `compileADTConstructor` (collections.go:457), builtins (builtins.go:96), cons/`_record_get`/by-name-update (collections.go:272/306): small fixed blocks, unaffected.
   - The list-literal chunk fold uses `OpConcat`, which today is only emitted for `stmt.OpConcat` (expr.go:338-339) — the new emitter must produce the same opcode semantics on list values (VM case already handles `TagList`, vm.go:244).
4. **Parser/typechecker disambiguation:** N/A — no syntax change.
5. **Programs that MUST still work post-change (regression fixtures — all exist, verified):**
   - `internal/bytecode/compiler/collections_test.go` — list/tuple literal compile tests (single-instruction path preserved for n ≤ K).
   - `internal/bytecode/compiler/control_flow_test.go` — if-expr/jump-patching tests (result-reuse must keep MOVE elision and jump targets identical where no nesting exists).
   - `internal/bytecode/compiler/regalloc_test.go` — allocator unit tests (untouched API).
   - `cmd/ailang/run_bytecode_match_test.go` — run-vs-bytecode parity corpus.
   - `internal/testing/engine_parity_test.go` — evaluator-bridge parity (must keep passing with the new warning on stderr; assertions are on stdout/return values — sprint must confirm).
   - The reported programs themselves: `sim/tools/ism_build.ail`, `sim/protocol.ail` (stapledons-godot, external) — acceptance is the two minimal repro classes, which are in-repo testable.
6. **What deliberately changes:**
   - n > K list/tuple literals now emit `MAKE_LIST(chunk)` × ⌈n/K⌉ + `OpConcat` folds instead of failing. Disassembly of large-literal functions changes (expected; no golden pins these today — sprint must grep `goldens/` to confirm).
   - Strict runs of modules with `EvalOnly` protos fail **before dispatch begins** with `BC001` diagnostics instead of a mid-run CALL trap. Strict users who relied on the trap's specific message text must move to the diagnostic (no in-repo test asserts that message; `internal/vm/vm.go:289`'s message remains for the bridge-unwired case reached by other means).
   - Non-strict `--bytecode` runs print a new stderr warning when `EvalOnly` protos exist. Scripts scraping stderr that assume silence will see the warning (fail-loud direction; accepted).

## Solution Design

### Overview

Three coordinated changes, all compiler/CLI-side:

1. **Phase A — surface, fail early (P0):** a strict-VM compile report. `CompileBytecodeFromResult` gains an `EvalOnlyReport` (or the check path re-derives it from the image + FuncDecls); `ailang check --strict-bytecode` turns it into `BC001` errors (exit ≠ 0); `ailang run --bytecode --strict-bytecode` fails **before running** with the same list (instead of a mid-run CALL trap); non-strict `--bytecode` prints a stderr warning (no silent fallback).
2. **Phase B1 — chunked collection lowering:** n > K list/tuple literals compile as K-element chunks (`MAKE_LIST`) folded with `OpConcat`.
3. **Phase B2 — if-expr chain result reuse:** an else-position `IfExpr` writes into the caller's already-allocated result register, collapsing per-level accumulation to a constant ~20 registers for chains of any length.

Phases A and B are independent; A lands first (it converts every *unknown* future pressure pattern into an early, actionable error), B removes the two known patterns.

### Architecture

**Components:**

1. **Strict-VM compile report** (`internal/bytecode/compiler` + `internal/runner` + `cmd/ailang`):
   - After image compilation, collect every `EvalOnly` prototype: canonical name, `File`, reason. The prototype already carries `Name`, `File`, `EvalReason` (`internal/bytecode/image.go:62-75`); the source **line** lives on the FuncDecl (`internal/gen/stmt/stmt.go:67-81`) — plumb it onto `FuncPrototype` at creation (additive field, ~5 LOC) or return `(img, report)` from a thin wrapper in `internal/runner/vm.go` next to `CompileBytecodeFromResult` (:45). Exact plumbing is a Deferred Decision; the report is a plain `[]EvalOnlyEntry` — no new analysis pass.
   - `ailang check --strict-bytecode` (new flag in `cmd/ailang/commands_language.go:365+` check command + `check.go`): runs the existing pipeline (check already produces `pipeline.Result`), compiles the image via `CompileBytecodeFromResult`, and for each report entry emits `BC001: function <module>.<name> is evaluator-only under --strict-bytecode (<reason>) at <file>:<line>` through the existing diagnostic format (`--json`/`--format agent` included). Exit ≠ 0 if any entry. Without the flag: no behavior change.
   - `ailang run --bytecode --strict-bytecode` (`internal/runner/vm.go` `tryRunEntryViaVM`): before dispatch, if the report is non-empty, fail with the same `BC001` list. (Today only an `EvalOnly` **entry** is pre-flighted at runner/vm.go:222; helpers fail mid-run at vm.go:289.)
   - `ailang run --bytecode` non-strict: print one stderr warning line per `EvalOnly` proto (capped, e.g. first 10 + count) before bridging.

2. **Chunked collection lowering** (`internal/bytecode/compiler/collections.go`):
   - In `compileListLit`/`compileTupleLit`: if n ≤ K (K = 64), keep the current single-`allocContig(n)` path byte-identical. If n > K: for each chunk c of ≤ K elements, `allocContig(len(c))` → `compileExprIntoSlot` each element (source order preserved) → `MAKE_LIST`/`MAKE_TUPLE` into a temp; then fold temps left-to-right with `OpConcat` (for lists; tuples chunk only if the tuple type permits — **see note**). Free each chunk's registers before allocating the next so pressure stays ≤ K + temps, constant in n.
   - Tuple note: tuple concatenation is not a list operation — chunked lowering applies to **list literals only** in this sprint; tuple literals with n > 256 stay `EvalOnly` (rare; now reported by Phase A). Scope-shrinking a tuple is left to Future Work.
   - Cost note (A9): `OpConcat` copies the left operand (vm.go:244 `OpCons`-family semantics; std/list.ail documents `++` as copying). Linear fold over ⌈n/K⌉ chunks costs O(n²/K) element copies — for a 601-element LUT with K=64: ~3.5k copies, negligible; for a pathological 100k literal, ~78M copies — acceptable for a data-table construct and strictly better than today's hard failure. The evaluator's single-pass build remains the non-strict path, so costs only differ on the strict VM.

3. **If-expr chain result reuse** (`internal/bytecode/compiler/control_flow.go`):
   - Add `compileIfExprInto(e, dst)` used when `e.Else` is itself a `stmt.IfExpr`: the nested chain compiles both its arms writing directly into the shared `dst` (the outer level's already-allocated result register), eliminating the per-level `allocTemp` and the trailing `MOVE`. Jump placeholders/patching logic is reused unchanged. Condition registers stay as-is (temps, freed immediately).
   - Effect: the previous arm's freed 16-register run is no longer fragmented by a per-level result register, so `findContigInFreeList` reuses the same ~19 registers at every level; a chain of any length compiles with constant pressure. Measured failure at 16 arms becomes: 60 arms pass (acceptance test).
   - Risk containment: the transformation applies only when the else branch is literally an `IfExpr` (the else-if shape); all other expressions keep the existing `compileIfExpr` untouched, so the non-nested codegen is byte-identical.

### Implementation Plan

**Phase A: Strict-VM compile report (P0)** (~1.5 days)
- [ ] Plumb FuncDecl `Line` onto `FuncPrototype` (or return the report alongside the image) so diagnostics carry file:line.
- [ ] `ailang check --strict-bytecode` flag + `BC001` diagnostic (human, `--json`, `--format agent`), exit ≠ 0 on any `EvalOnly` entry; add `BC001` to `internal/errors/codes.go` with phase/category metadata.
- [ ] `ailang run --bytecode --strict-bytecode`: pre-flight failure with the `BC001` list before dispatch (replaces the mid-run trap for the compile-tag case).
- [ ] `ailang run --bytecode` non-strict: stderr warning listing `EvalOnly` prototypes (count + names).
- [ ] `ailang test` strict mode: report helpers too (test body gate already exists at `internal/testing/bytecode_engine.go:157`).
- [ ] Corpus sweep: run the report over `std/`, `examples/`, `goldens/`; bank the current `EvalOnly` census (per reason) as the baseline the sprint-evaluator diffs against.

**Phase B1: Chunked list-literal lowering** (~1 day)
- [ ] `compileListLit` chunked path (K=64 constant) with per-chunk register free; source-order element evaluation.
- [ ] Unit tests: 601-, 1000-, and K-boundary literals (K-1, K, K+1, 2K) compile, disasm sanity, strict-VM run equals evaluator output byte-for-byte.
- [ ] Confirm `cmd/ailang/run_bytecode_match_test.go` corpus unchanged.

**Phase B2: If-expr chain result reuse** (~0.5 day)
- [ ] `compileIfExprInto` for else-position `IfExpr`; `compileIfExpr` delegates when the shape matches.
- [ ] Unit tests: 60-arm else-if chain over a 16-field record (today's 16-arm failure threshold becomes any-length), jump-patch equivalence vs. the nested form on small chains, `control_flow_test.go` green.
- [ ] Re-run the Phase A census: the two reported cause classes should drop to zero across the corpus.

**Phase C: Documentation** (~0.5 day)
- [ ] `docs/` strict-bytecode reference: `BC001`, the check gate, the K-chunking behavior note (visible in disasm).
- [ ] CHANGELOG entry; note that the stapledons-godot manual workarounds (chunk-to-100, `applyOverride2`) can be reverted upstream.

### Files to Modify/Create

**Modified files:**
- `internal/bytecode/compiler/compiler.go` — carry FuncDecl `Line` onto protos; expose the `EvalOnly` report (~30 LOC)
- `internal/bytecode/image.go` — additive `Line` field on `FuncPrototype` if chosen over the report wrapper (~10 LOC)
- `internal/bytecode/compiler/collections.go` — chunked `compileListLit` (~60 LOC)
- `internal/bytecode/compiler/control_flow.go` — `compileIfExprInto` chain reuse (~40 LOC)
- `internal/errors/codes.go` — `BC001` (+ tests file) (~15 LOC)
- `cmd/ailang/commands_language.go`, `cmd/ailang/check.go` — `--strict-bytecode` check flag (~80 LOC)
- `internal/runner/vm.go` — strict pre-flight + non-strict warning (~40 LOC)
- `internal/testing/bytecode_engine.go` — strict test reporting of helpers (~15 LOC)
- Tests alongside each (~200 LOC)

**New files:** none.

## Examples

### Example 1: Large LUT literal (reported repro)

**Before** (`ailang run --bytecode --strict-bytecode big_list.ail`):
```
Error: bytecode execution failed: vm: vm: CALL: big_list.xs is evaluator-only
(register allocator: contiguous block of 601 would exceed 256) but no interop
bridge is wired (in big_list.main at big_list.ail:4, ip 3, op CALL)
```
**After:**
```
601
```
(disasm shows `MAKE_LIST` × 10 chunks of ≤ 64 + `OpConcat` folds; `ailang check --strict-bytecode big_list.ail` → exit 0.)

### Example 2: else-if dispatch over a wide record (reported repro)

**Before:** 16 arms → mid-run CALL trap (15 arms pass, 16 fail — measured).
**After:** 60 arms → identical result on strict VM and evaluator; `check --strict-bytecode` exit 0.

### Example 3: A shape this sprint deliberately does not fix

A 300-field record literal compiles to an `EvalOnly` proto as today, but is now **reported at the earliest gate**:
```
$ ailang check --strict-bytecode weird.ail
BC001: function m.wide is evaluator-only under --strict-bytecode
  (register allocator: contiguous block of 300 would exceed 256)
  at weird.ail:3
```
(exit ≠ 0; plain `ailang check` and non-strict runs unchanged apart from the stderr warning.)

## Success Criteria

- [ ] 601- and 1000-element float list literals run under `--bytecode --strict-bytecode` with the evaluator's exact output (acceptance: new `cmd/ailang` test mirroring `run_bytecode_match_test.go`).
- [ ] A 60-arm else-if chain of record updates over a 16-field record runs under `--strict-bytecode` matching the evaluator (measured pre-fix threshold: fails at 16 arms).
- [ ] `ailang check --strict-bytecode` exits ≠ 0 with one `BC001` per `EvalOnly` proto on a module with an oversized construct, and exits 0 on both fixed repros.
- [ ] `ailang run --bytecode` (non-strict) emits the `EvalOnly` warning; no silent fallbacks remain.
- [ ] All existing tests passing: `make test` (incl. `make check-boundaries` — no layer violations, `internal/` only), `control_flow_test.go`, `collections_test.go`, `regalloc_test.go`, engine-parity corpus.
- [ ] Documentation updated (`docs/` strict-bytecode page, CHANGELOG).

## Testing Strategy

**Unit tests:**
- Allocator: chunk boundary sizes (K-1, K, K+1, 2K, 601, 1000) do not exceed the 256 ceiling (`highWater` assert).
- Codegen: chunked literal emits the documented instruction sequence; element evaluation order is source order (observable via effectful elements in a non-pure test harness — or by construction for pure code plus disasm asserts).
- Chain reuse: nested else-if codegen equals the nested-form codegen semantics on small chains (same results on both paths via the VM).

**Integration tests:**
- `run_bytecode_match_test.go`-style parity for both repros: strict VM output === evaluator output, exit codes, and the two failure transcripts replaced by success.
- `check --strict-bytecode` on a module with an `EvalOnly` helper: exit ≠ 0, `BC001` present in `--json` and `--format agent` outputs.
- Non-strict warning on stderr does not pollute stdout result printing (engine-parity and golden scripts keep passing).

**Manual testing:**
- `ailang disasm` on a 601-element literal before/after; confirm chunk folds and register pressure.
- Run the Phase A census over `std/`, `examples/`, `goldens/` before and after Phase B — the two known cause classes go to zero; residual entries are individually triaged (each is now visible and named).

## Deferred Decisions

The following are intentionally left open for the implementer:

- **Report plumbing** — add `Line` to `FuncPrototype` vs. return `(img, []EvalOnlyEntry)` from a wrapper next to `CompileBytecodeFromResult`. Both are additive; agent may choose (prefer the proto field: `ailang disasm` benefits too).
- **Chunk size K** — any constant in [32, 128] that keeps `K + caller working set` safely under 256; default 64; agent may tune from census data.
- **Non-strict warning verbosity** — one line per proto vs. capped summary; agent may choose, but the count must always be exact.
- **Static-typed `compileRecordUpdate` via `OpUpdateRecord`** (block of `len(overrides)+1` instead of full `len(fields)` rebuild) — reduces per-update pressure from 16 to 2 and would widen the chain headroom further, **but** `OpUpdateRecord` preserves the base record's field order (vm.go:461-492: copy + replace/append by name) while the static path's `MAKE_RECORD` rebuild guarantees the *sorted* field order that subsequent indexed `OpGetField` (C = sorted index, opcode.go `OpGetField` comment) relies on. Switching is only safe if every record reaching the static path provably has sorted field order. Deferred: needs that invariant proven (or a verification pass added) first; the chain fix makes it unnecessary for the reported repro. Agent may prototype behind the census data; human approves any switch.
- **`ailang check --package --strict-bytecode` batch form** — mechanical extension of the single-file flag once the single-file gate exists; agent may include or follow up.

## Non-Goals

**Not attempted in this feature:**
- **Raising the 256-register ceiling or widening instruction operands** — 8-bit ABC operands are frozen VM encoding (core-floor lane, would re-freeze the VM); chunking makes it unnecessary for the reported shapes.
- **General register spilling (`OpSpill`/`OpRestore`)** — ~500 LOC of new VM ABI per the v0.11 assessment (M-BYTECODE-REGALLOC-FIX), overkill once the two known patterns compile; revisit only if the post-fix census shows real programs still failing (each will now be visible as `BC001`).
- **A smarter general-purpose allocator (linear-scan, defragmentation)** — regalloc is deliberately simple (`regalloc.go:11-14`); the chain fix removes the known pathology without redesigning it.
- **Tuple-literal chunking** — no `OpConcat` analogue for tuples; > 256-element tuple literals stay `EvalOnly` (now reported). Rare in real code (the typechecker allows it, the encoding does not).
- **> 255-field record literals/updates** — same encoding cap (`MAKE_RECORD` C operand); stay `EvalOnly`, reported by Phase A.
- **Emitting errors from plain `ailang check`** — programs that run fine on the evaluator must keep checking clean; the strict gate is opt-in.

## Timeline

**Week 1** (~3.5 days):
- Phase A: report + `BC001` + check/run/test gates + census baseline (1.5 d)
- Phase B1: chunked list lowering + tests (1 d)
- Phase B2: if-expr chain reuse + tests (0.5 d)
- Phase C: docs + CHANGELOG (0.5 d)

**Total: ~3.5–4 days** (estimates are 2× initial guess; Phase A is independently shippable if B overruns).

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Chunk fold changes evaluation timing of large literals (element effects) | Med | List elements in a literal are pure expressions in the common case; element order is preserved exactly; parity tests on the repro corpus (evaluator is untouched as reference). |
| Result-reuse if-expr codegen breaks jump patching on nested chains | Med | Transformation only when else branch is literally `IfExpr`; jump-placeholder machinery reused unchanged; small-chain equivalence tests vs. the old nested form; `control_flow_test.go` is the regression net. |
| Non-strict stderr warning breaks golden scripts that assume silence | Low | Warning is stderr-only, stdout byte-identical; sweep `goldens/` + `internal/testing` for stderr assertions in sprint. |
| Census reveals other `EvalOnly` causes beyond the two fixed | Low (it's the point) | Each residual entry is named with reason + position; triage routes them to follow-up docs — Phase A guarantees visibility regardless. |
| `BC001` JSON format churn for tooling | Low | Follows the existing `errors/codes.go` ErrorInfo shape used by every other code; new family, no renames. |

## Related Documents

**Implemented (may inform design):**
- [m-bytecode-regalloc-fix.md](../implemented/v0_11_0/m-bytecode-regalloc-fix.md) — the v0.11 allocator work: scope recycling (`scopeStack`) and contiguous-from-free-list (`findContigInFreeList`) both live in today's `regalloc.go`; this doc's Cause B is the residual pathology those fixes did not cover (per-level result registers fragmenting freed runs), and its "spilling (~500 LOC), likely overkill" note is carried into Non-Goals.
- [m-bytecode-nested-pattern-lowering.md](../implemented/v0_52_0/m-bytecode-nested-pattern-lowering.md) — the sibling strict-VM-coverage degradation class (same reporter project, same `EvalOnly` mechanism at `compiler.go:159-176`, different root cause in pattern lowering); its evaluator-as-reference and fail-loud rulings are adopted here.
- [m-bytecode-2d-parity.md](../implemented/v0_11_0/m-bytecode-2d-parity.md) — the EvalOnly stub + bridge contract this design surfaces rather than changes.
- [v0.10-v0.17-bytecode-vm.md changelog](../../../changelogs/v0.10-v0.17-bytecode-vm.md) — history of the strict/bridge split.

**Planned (checked for overlap):**
- None — `ailang docs search` (SimHash + neural) on the topic found no planned doc covering register pressure or `EvalOnly` surfacing (the script's auto-query "reg ceiling" was too narrow; the manual sweep above is the authoritative result).

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- Issue: this task (coordinator-dispatched; field report from stapledons-godot v0.52.0, `sim/tools/ism_build.ail`, `sim/protocol.ail`)
- [M-BYTECODE-2D M3 contract](../implemented/v0_11_0/m-bytecode-2d-parity.md) — why per-function compile failure tags `EvalOnly` instead of failing the image (this design does not change that contract for non-strict use; it adds the strict-mode gate on top)

## Verification Log

Every "does/doesn't" claim above, with how it was checked. All live checks on installed `ailang` v0.52.5 (binary md5 2dd082832… is the reporter's v0.52.0; repo HEAD is v0.53.0-dev with the same `regalloc.go:66` code path — the workspace has no Go toolchain, so the installed binary is the verification instrument).

| # | Claim | Method | Result |
|---|-------|--------|--------|
| V1 | 601-element list literal passes `ailang check` | Live: `ailang check big_list.ail` | `✓ No errors found!`, exit 0 (V-transcript A) |
| V2 | Same program on `--bytecode` (non-strict) prints the right answer with **no warning** | Live: `ailang run --bytecode big_list.ail` | prints `601`, exit 0, stderr clean of any EvalOnly notice (V-transcript A) |
| V3 | Same program on `--bytecode --strict-bytecode` fails at run time with the reported message | Live: `ailang run --bytecode --strict-bytecode big_list.ail` | `Error: bytecode execution failed: vm: vm: CALL: big_list.xs is evaluator-only (register allocator: contiguous block of 601 would exceed 256) …`, exit 1 (V-transcript A) |
| V4 | The if/else-if chain failure reproduces and has a measurable threshold | Live: generated chains of 10–60 arms, 16-field record, full-record update per arm | 15 arms pass, 16 arms fail with `contiguous block of 16 would exceed 256` (V-transcript B) |
| V5 | The allocator error is compile-time and becomes a silent `EvalOnly` tag (no diagnostic emitted) | Code: `regalloc.go:66-67` (error site), `compiler.go:120-176` (tag-on-compile-failure; no stderr/log call in that block — read in full) | Confirmed |
| V6 | Non-strict wires the bridge silently; strict leaves it nil so CALL traps; strict pre-flight exists only for the *entry* | Code: `runner/vm.go:206-222` (bridge wiring + entry pre-flight), `vm/vm.go:289` (CALL trap) | Confirmed |
| V7 | `ailang check` has no flag that compiles bytecode | Live: `ailang check --help` (flags: debug-compile, format, json, package, quiet, relax-modules, strict-syntax, timeout, verify*) — none compile the image; grep: `CompileBytecodeFromResult` is referenced only by `cmd/ailang/disasm.go`, `internal/runner/{vm,batch}.go`, `internal/testing/bytecode_engine.go` — zero references in `cmd/ailang/check.go` (which contains no "bytecode" token at all) | Confirmed |
| V8 | `ailang test --strict-bytecode` fails loudly only for an evaluator-only **test body**, not helpers | Code: `internal/testing/bytecode_engine.go:157` (body check), `:185-197` (harnessBridge for helpers in non-strict) | Confirmed |
| V9 | Diagnostic code `BC001` and family prefix `BC` are unallocated; `MOD016` is unallocated (fallback) | Grep: `grep -rn '"BC0\|BC00\|BC01' internal/ cmd/ --include='*.go'` → empty; `grep -rn 'MOD016' internal/ cmd/ --include='*.go'` → empty; MOD001–MOD015 allocated in `internal/errors/codes.go` | Confirmed — BC001 free, MOD016 free |
| V10 | 8-bit operand encoding caps single-instruction collection construction at 255 (ceiling is architectural) | Code: `opcode.go:191-198` (`EncodeABC(op, a, b, c uint8)`), opcode comments for `OpMakeList`/`OpMakeTuple`/`OpMakeRecord`/`OpUpdateRecord` (`R[B..B+C-1]`) | Confirmed |
| V11 | `OpConcat` exists, handles list values, needs no new opcode | Code: `opcode.go:58`, `vm/vm.go:244` (dispatch case) | Confirmed |
| V12 | `OpUpdateRecord` copies base field order and replaces/appends by name (the invariant risk behind the Deferred Decision) | Code: `vm/vm.go:461-492` (`fields := append(..., base.AsRecord()...)`; replace-by-name; append if absent) | Confirmed |
| V13 | Cause-B mechanism: one result register per nesting level allocated before the else branch, stealing the freed run and forcing 16 fresh bumps per level | Code: `control_flow.go:92-141` (`result := allocTemp()` before `compileExpr(e.Else)`; freeContig LIFO in `regalloc.go:125-129`; findContigInFreeList in `regalloc.go:71-118`) + arithmetic matching the measured V4 threshold (2 params + 16·15 levels = 242, level 16 needs 259 > 256) | Confirmed — mechanism matches measurement |
| V14 | `compileRecordUpdate` (static path) allocates a block of **all** fields (16), not just overrides | Code: `collections.go:378-396` (`n := len(info.sortedFields)`; `allocContig(n)`; `MAKE_RECORD` with all values) | Confirmed |
| V15 | Regression fixtures exist: `collections_test.go`, `control_flow_test.go`, `regalloc_test.go`, `run_bytecode_match_test.go`, `engine_parity_test.go` | `ls` of `internal/bytecode/compiler/`, `cmd/ailang/`, `internal/testing/` | All exist |
| V16 | FuncDecl carries `File`/`Line`; FuncPrototype carries `Name`/`File`/`EvalReason` (diagnostic plumbing is additive) | Code: `internal/gen/stmt/stmt.go:67-81`, `internal/bytecode/image.go:62-75` | Confirmed |
| V17 | Match lowers to if-*statement* chains (not if-exprs), so the chain fix's scope claim ("statement chains unaffected") holds | Code: `internal/gen/lower/match.go:161` (`stmt.IfStmt{...}` general path; `:58` IfExpr only for the 2-arm literal special case) | Confirmed |
| V18 | No planned/implemented design doc already covers register-pressure or `EvalOnly` surfacing (duplicate gate) | Script auto-search (empty) + manual sweep of `design_docs/` (`grep -rln "register allocator\|regalloc\|256-register\|strict-bytecode"`) → only v0.11 regalloc-fix (shipped, different scope) and v0.52 nested-pattern-lowering (different root cause) | Confirmed — no duplicate |

**V-transcript A** (v0.52.5, `/usr/local/bin/ailang`):
```
$ ailang check big_list.ail
→ Type checking big_list.ail... → Effect checking...
✓ No errors found!
$ ailang run --bytecode big_list.ail
601
$ ailang run --bytecode --strict-bytecode big_list.ail
Error: bytecode execution failed: vm: vm: CALL: big_list.xs is evaluator-only
(register allocator: contiguous block of 601 would exceed 256) but no interop
bridge is wired (in big_list.main at big_list.ail:4, ip 3, op CALL)   # exit 1
```

**V-transcript B** (16-field record `V`, arms `if x == i then { r | <all 16 fields> } else …`):
```
arms=15 => 3.0          # compiles and runs on the strict VM
arms=16 => Error: … applyOverride is evaluator-only (register allocator:
          contiguous block of 16 would exceed 256) … (exit 1)
arms=18, 20, 30, 40, 60 => same error
```

## Future Work

- **Tuple-literal chunking** — needs a tuple-concat opcode or a builder form; only if the census shows real > 256-element tuple literals.
- **Static record-update via `OpUpdateRecord`** — pending the sorted-field-order invariant proof (see Deferred Decisions); would cut per-update pressure from `len(fields)` to `len(overrides)+1`.
- **General spilling** — only if post-fix `BC001` census shows real programs still hitting the ceiling.
- **Wider-operand instruction format** — core-floor lane; explicitly out of scope, but the census data from Phase A is the evidence a future re-freeze decision would need.
