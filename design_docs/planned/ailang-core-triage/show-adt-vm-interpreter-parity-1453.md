# show() of an ADT diverges: VM prints `<adt#0 6>`, interpreter prints `Some(6)` (#1453)

- **Date**: 2026-10-01
- **Class**: bug
- **Recommend**: design-doc
- **Searched**: `adt#0` / `show parity` / `ADT constructor name rendering` across `design_docs/`; `strict-bytecode`; `show diverges run vs compile`; read `internal/vm/builtins.go` `showValue`
- **Estimate**: ≥3 files (`internal/bytecode` encoder+decoder to carry constructor-name metadata, `internal/vm/builtins.go` to consume it, tests) — far past DIRECT_FIX_MAX_LINES=2 / MAX_FILES=1

The report is a status note from `stapledons_godot` ("fixed on dev": #1447, #1448, #1450, run-flag
refusal, named-record entry args) whose design doc already exists and is IMPLEMENTED —
`design_docs/implemented/v0_50_2/m-vm-pure-builtin-coverage.md` covers exactly that work and, at
its line 267, states it "found and filed separately: #1453 (`show` of an ADT differs VM vs
interpreter, pre-existing)". So the only actionable content is #1453.

The defect is real and already localised in-source: `internal/vm/builtins.go` `showValue`,
`case bytecode.TagADT` (~lines 324–329) — the ADT tag is a per-type ordinal, not a constructor
name, and the comment explicitly defers: mapping back to a name "requires the compiler's type
table (§4.3), which the VM does not currently carry… Fixing this cleanly is M3 scope
(cross-module ADT/record merging)." The interpreter rendering is the reference semantics per the
established ruling in `design_docs/planned/m-array-show-diverges-run-vs-compile.md` (that doc
covers the Go-codegen backend, a different member of the show-divergence class; ADT constructors
there were closed via PR #822 — the VM path was never in its scope).

No doc scopes the fix itself: `design_docs/planned/v1_0_0/m-bytecode-vm-parity-bugs.md` lists
"ADT constructor name rendering (M3 of M-BYTECODE-MULTIMODULE scope)" in its out-of-scope section
(line ~517), and `m-bytecode-multimodule-sprint-plan.md` M3 covers imported-ADT resolution and
CoreTI merging, not show rendering. So this is a known, pre-diagnosed gap with no owning design.

Why design-doc and not direct-fix: the fix requires the VM to receive constructor-name/type-table
metadata it does not carry today, which means the bytecode module serialization format gains new
data (row 4: file-format change), spans more than one file (row 5), and admits at least two
shapes — full type-table plumbing vs a minimal constructor-name string table per TypeDecl (row 3).
It should be scoped alongside or after the v0.50.2 strict-VM work it was carved out of, and it
will need a golden/differential fixture gating VM-vs-interpreter byte parity the way
`TestInterpreterCompiledDifferential` does for the compile backend.