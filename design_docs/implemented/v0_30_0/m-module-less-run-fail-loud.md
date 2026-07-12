# M-MODULE-LESS-RUN-FAIL-LOUD: `ailang run`/`check` must not silently succeed on a module-less file

**Status**: Implemented
**Target**: v0.30.0
**Priority**: P1 (silent-success class — violates "NO SILENT FALLBACKS, FAIL LOUDLY"; a fleet-tier footgun per strategy R1/R4, bar-v2 clause 3)
**Estimated**: ~0.5 day (pipeline diagnostic ~30 LOC + block_demo remediation + fixtures + regression)
**Dependencies**: none; complements [m-diagnostic-coverage (implemented)](../../implemented/v0_29_0/m-diagnostic-coverage.md) — this is one more footgun → fix-carrying diagnostic
**Author**: Opus mission session, requested by Mark 2026-07-12 (found while triaging a nightly regression that turned out to be unrelated)

---

## Problem Statement

A `.ail` file that has top-level declarations (`export func main …`) but **no `module` declaration**
type-checks clean, prints `✓ Running`, and **exits 0 with no output and no error** — the entry never
runs. Verified live on v0.29.2 (dev):

```
$ printf 'import std/io (println)\nexport func main() -> () ! {IO} { println("SHOULD_PRINT") }\n' > /tmp/nomod.ail
$ ailang run --entry main --caps IO /tmp/nomod.ail
→ Type checking...
→ Effect checking...
✓ Running /tmp/nomod.ail
$ echo $?
0                    # ← silent success; SHOULD_PRINT never printed, main never ran
$ ailang check /tmp/nomod.ail
✓ No errors found!   # ← check is silent too
```

Adding `module scratch/nomod` makes it run and print correctly. The failure is invisible: no output,
no error, exit 0 — the worst possible signal for an AI author (and it cost real debugging time in the
2026-07-12 session; it also silently sinks any model-generated benchmark solution that omits the
module line — those show as empty-stdout failures with no cause).

## Root Cause (traced)

1. `internal/pipeline/pipeline_module.go:665` — `validateModulePath` returns `nil` (accepts) when
   `mod.File.Module == nil`. A file with no module declaration is silently accepted and produces **no
   exports**.
2. `cmd/ailang/main_run_exec.go:278` — the runner only resolves/executes an entrypoint
   `if result.Interface != nil && len(result.Interface.Exports) > 0`. With no exports it falls to the
   `else` at line 615 ("Non-module mode") → `printNonModuleResult(result.Value, …)`
   (`run_helpers.go:536`) which prints only a **non-unit** value. The declarations evaluate to unit,
   nothing prints, and the function returns success.

So a "forgot `module`" file is misclassified as a bare-expression eval that happens to yield unit.

## Goals

`ailang run`/`check` on a file that has declarations but no `module` declaration **fails loudly**
with a fix-carrying diagnostic — while every legitimate mode is preserved.

**Success**: exit ≠ 0 + a message like
`MOD014: no 'module' declaration — add 'module <derived/path>' at the top of the file` (the derived
path computed the same way MOD010 already suggests one). Both `check` and `run` emit it (one fix,
pipeline-level — systemic per AUDIT-BEFORE-PATCHING).

## Solution Design

In `validateModulePath` (the single chokepoint check/run/eval/test all pass through), replace the
`mod.File.Module == nil { return nil }` early-accept with:

```go
if mod.File.Module == nil {
    if len(mod.File.Funcs) > 0 {
        return <MOD014 error: no module declaration; suggest `module <canonicalID>`>
    }
    return nil // genuinely empty file OR a bare-expression eval — unchanged
}
```

Use the real AST fields (`ast.File.Funcs`). Suggested path = `loader.CanonicalModuleID(modID)`
(already used by MOD010).

> **Implementation correction (2026-07-12):** The original proposal gated on
> `len(Funcs) > 0 || len(Statements) > 0 || len(Decls) > 0` on the assumption that a bare
> expression (`1 + 1`) routes through `runSingle` and never reaches `validateModulePath`. That
> assumption is **false** for a bare-expression *file with a filename*: `RunWithContext` routes any
> non-REPL file with a filename through the **module** pipeline, so `1 + 1` in a `.ail` file DOES
> reach `validateModulePath`. The parser mirrors that bare expression into BOTH `Statements` and
> `Decls` (back-compat, `parser_file.go`), so the 3-way OR would have made `ailang run file-with-1+1`
> fail with MOD014 — breaking the eval escape hatch. The shipped guard is **`len(Funcs) > 0` only**:
> a genuine "forgot `module`" file always has top-level `Funcs` (`export func main …`), while the
> bare-expression path has none. Verified live: `1 + 1` → `2` still works, module-less `export func`
> now fails loudly.

## Conflict Surface Analysis

**Error-code collision (2026-07-12 correction):** the original doc allocated `MOD011` for this
diagnostic, but **`MOD011` was already taken** — it is the module-path-collision diagnostic
(`internal/pipeline/pipeline_module.go`, live since v0.10.9: "module X is declared in two different
files"). Allocated codes at implementation time are MOD001–MOD013 (only MOD008 is a free gap). This
feature ships as **`MOD014`** (next fresh, unambiguous) everywhere — in code and in this doc.

`validateModulePath` is shared by **check, run, eval, test, package-check**. Adjacent behavior that
must NOT regress:
- **MOD010** (declared-but-mismatched path) + its temp-path/`--relax-modules` relaxation — untouched
  (the MOD014 branch is the `Module == nil` case, disjoint from MOD010's `Module != nil` path).
- **`std/*` bypass** and **`pkg/*` prefix mapping** — untouched (also `Module != nil`).
- **Bare-expression eval** (`ailang run` on a `.ail` file containing `1 + 1` → `2`): **CORRECTION** —
  a bare-expression *file with a filename* routes through the **module** pipeline and DOES reach
  `validateModulePath` (only the REPL / no-filename path goes to `runSingle`). It is preserved because
  the MOD014 guard is `len(Funcs) > 0` only, and a bare expression has no top-level `Funcs`. Verified:
  `1 + 1` → `2` still works. MUST stay working.
- **Top-level effectful eval** in genuine non-module files — confirm the fixture set still behaves.

**Blast radius (audited, 2026-07-12; re-confirmed at implementation)**: exactly **1** corpus file —
`examples/runnable/block_demo.ail` (has `export func compute/singleLine/multiStep`, no `module`, no
`main`; currently a silent no-op). Remediated: added `module examples/runnable/block_demo` as the first
line. It has no `main`, and `scripts/verify-examples.sh` already treats no-entrypoint files as
check-only, so it now checks clean with no manifest change. All other examples/stdlib/benchmarks
already carry module declarations, so the fix makes zero currently-passing case fail (a module-less
file cannot pass today — it produces no output).

`make verify-examples` at implementation showed 5 pre-existing failures (`effectful_list`,
`effectful_list_t7_chain_combinators`, `mcp_tools`, `stream_multi_source`, `stream_process_source`) —
all effect-row / type-unification errors, all **module-bearing**, and all confirmed failing identically
on the base `origin/dev` binary. They are unrelated to MOD014; block_demo passes.

## Testing Strategy (TDD)

1. **New fixture, must FAIL before the fix**: module-less file with `export func main` → non-zero
   exit + message containing `module` and the suggested path.
2. **Regression guards (must stay green)**: `1+1` → `2` (bare expr); a proper `module` file runs and
   prints; `--relax-modules` on a temp-path *mismatch* still relaxes (MOD010 path).
3. **CI fixture** in the footgun-coverage harness (per m-diagnostic-coverage) asserting the diagnostic
   text carries the fix.
4. Full: `make test` + `make verify-examples` (after block_demo remediation) + `make verify-stdlib`.

## Routing (PROGRAM.md §4)

**AILANG fix** — `internal/pipeline` + a new `MOD014` diagnostic. Not motoko, not core. Directly
serves strategy **R1** (error-time teaching: the compiler *is* the prompt) and bar-v2 **clause 3**
(fleet-tier footgun burn-down). Could fold into the m-diagnostic-coverage footgun table as one row.

## Non-Goals

- Auto-inserting the `module` line (suggest, don't rewrite — same stance as MOD010).
- Changing `runSingle` bare-expression behavior.
- Making `check` require a `main` entry (that's an entry-resolution concern, separate).

## Verification Log

| Claim | Method | Result |
|---|---|---|
| Silent success on module-less file | live `ailang run` + `echo $?`, v0.29.2 | Confirmed (exit 0, no output) |
| `check` also silent | `ailang check /tmp/nomod.ail` | Confirmed (✓ No errors found!) |
| Root cause = validateModulePath early-accept | read pipeline_module.go:665 | Confirmed |
| Runner non-module fall-through | read main_run_exec.go:278/615, run_helpers.go:536 | Confirmed |
| Bare-expr eval unaffected (different path) | `ailang run` on `1+1` → `2` | Confirmed still works |
| Blast radius = 1 file | grep corpus for decls-without-module | `examples/runnable/block_demo.ail` only |
| AST fields | internal/ast/ast.go:139-142 | `Module`, `Funcs`, `Statements`, `Decls`(deprecated) |

## Related Documents

- [m-diagnostic-coverage](../../implemented/v0_29_0/m-diagnostic-coverage.md) — the footgun-diagnostic program this joins
- [m-fable-strategy-review](../m-fable-strategy-review.md) — R1 (error-time teaching) this instances
- [v1-mission.md](../../v1-mission.md) — bar-v2 clause 3 (fleet-tier accessibility)

---

**Document created**: 2026-07-12
