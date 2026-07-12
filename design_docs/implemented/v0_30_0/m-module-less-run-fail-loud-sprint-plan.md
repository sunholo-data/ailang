# Sprint Plan — M-MODLESS-FAIL-LOUD (MOD014)

**Status**: Implemented
**Design doc**: [m-module-less-run-fail-loud.md](./m-module-less-run-fail-loud.md)
**Branch**: `mission/modless-fail-loud` → PR to `dev`
**Executed**: 2026-07-12 (Opus sprint-executor, isolated worktree)

## Goal

`ailang run`/`check` on a `.ail` file that has top-level declarations but NO `module`
declaration must FAIL LOUDLY with a fix-carrying diagnostic (`MOD014`), instead of
silently printing `✓ Running` and exiting 0 with no output (the entry never runs).

## Error-code correction

The design doc originally proposed `MOD011`, but **MOD011 was already taken** (module-path
collision, live since v0.10.9). Allocated codes were MOD001–MOD013 (only MOD008 free).
This feature uses **`MOD014`** everywhere.

## Milestone M1 — MOD014 diagnostic (TDD)

1. Failing Go test first (`internal/pipeline/module_less_test.go`) asserting a module-less
   file with `export func main` yields `MOD014`. Confirmed FAIL before the code change.
2. Replaced the `mod.File.Module == nil { return nil }` early-accept in
   `validateModulePath` (`internal/pipeline/pipeline_module.go`) with a MOD014 error when
   `len(mod.File.Funcs) > 0`, carrying `Fix: add 'module <canonicalID>' as the first line`.
   - **Guard is `len(Funcs) > 0` ONLY.** A bare-expression file (`1 + 1`) with a filename
     routes through the MODULE pipeline (not `runSingle`, contrary to the doc's original
     assumption) and reaches `validateModulePath`; the parser mirrors that expression into
     both `Statements` and `Decls`, so gating on Decls/Statements would break `1 + 1` → `2`.
     A "forgot module" file always has top-level `Funcs`.
3. Regression guards (all green): bare-expr `1 + 1` → `2`; proper module runs/prints;
   `--relax-modules` on a temp-path MISMATCH still relaxes (MOD010, `Module != nil`);
   `std/*` and `pkg/*` (all `Module != nil`) unaffected.
4. Both `ailang run` AND `ailang check` emit MOD014 (shared pipeline chokepoint).
5. Footgun-coverage fixture added to `internal/diag/footgun_fixtures_test.go`
   (`TestFootgunFixture_MOD014_ModuleLess` + `_BareExpressionPreserved`) — real-file,
   module-pipeline, since MOD014 fires only with a filename. Row added to
   `internal/diag/footguns.md`.

Commit M1: `feat(pipeline): MOD014 — fail loudly on module-less run/check (silent exit-0 footgun)`.

## Milestone M2 — block_demo remediation + docs

1. Added `module examples/runnable/block_demo` as the first line of
   `examples/runnable/block_demo.ail` (no `main`; check-only via verify-examples).
2. `make verify-examples`: block_demo passes. 5 pre-existing failures remain (effect-row
   errors, all module-bearing, confirmed identical on base `origin/dev`) — NOT caused by
   this change; blast radius stays exactly 1.
3. Design doc: `MOD011`→`MOD014` throughout, MOD011-taken note added to Conflict Surface,
   Status → Implemented, `git mv` planned/ → implemented/v0_30_0/.
4. Docs: CHANGELOG entry + LIMITATIONS resolved-entry noting the silent-success footgun is
   fixed via MOD014. No stability-tier change (diagnostic, not stdlib surface).

Commit M2: `fix(examples): remediate block_demo module-less; docs + design doc → implemented (MOD014)`.

## Acceptance

- Live repro: module-less file exits ≠ 0 with MOD014 (run + check); `1 + 1` → `2` and a
  proper module file both still work.
- `make test` / `verify-examples` (block_demo green) / `verify-stdlib` / `lint` / `fmt`.
