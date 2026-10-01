# Sprint Plan: M-VM-PURE-BUILTIN-COVERAGE

**Design doc**: [m-vm-pure-builtin-coverage.md](m-vm-pure-builtin-coverage.md)
**Issues**: #1447, #1448 · **Duration**: 2–3 days · **Risk**: medium (new opcode; AST printer change can move goldens)
**Registry reuse**: none — every milestone is compiler/VM/test-harness internals, not a package-shaped capability.

## Milestones

### ✅ M1 — Coverage ratchet, honest error, table lockstep (~250 LOC)
- `internal/vm/builtin_coverage_test.go` (package `vm_test`): registry `IsPure` × {native, adapted, unported-allowlist}.
- `internal/vm/builtin_unported.go`: `UnportedPure map[string]string` (reason ∈ `map-value|adt-result|closure-arg|polymorphic-needs-native-port|non-convertible`).
- Real `validateBuiltinTables` (compiler vs VM table lengths, plus name check if the VM side can carry names) called from `vm.New`.
- `compileBuiltinCall`: registry-pure but unwired → `compiler: pure builtin %q has no VM implementation (%s)`.
- **AC**: ratchet test fails if a new pure builtin is unbucketed, or an allowlisted one becomes covered; error text test; lockstep test (mutation: drop one VM entry → fails).

### ✅ M2 — Generic registry adapter (~350 LOC, ~170 moved)
- Move converters `internal/runner/bridge.go` → `internal/vm/convert.go` (exported); runner calls them.
- `adaptable(type)`: no type vars, Map, function types, ADT in params/result.
- `OpBuiltinCallAdapted` + disasm + VM dispatch; table sorted by name.
- Panics in an `Impl` recovered into a VM error.
- **AC**: eligibility unit tests (accept + each reject reason); a CLI-level strict run of a monomorphic adapted builtin (e.g. a `std/bytes` or `std/string` call) matches the interpreter; existing bridge tests green; allowlist shrinks by the adapted count (reported).

### ✅ M3 — Native polymorphic ports (~250 LOC)
`__list_reverse/take/drop/zip/range/contains/head/extract`, `__str_repeat`, `__string_reverse`, copying evaluator semantics exactly.
- Parity table test per port: registry `Impl` vs VM func (empty, n<0, n>len, ADT elements, nested lists).
- `examples/vm_strict_pure_builtins.ail` runs under `--strict-bytecode`; added to verify-examples manifest if required.
- **AC**: #1447 `rev.ail` repro strict → `[3, 2, 1]`; example output identical across interpreter / `--bytecode` / strict.

### ✅ M4 — #1448 float literals in test blocks (~80 LOC)
- `ast.FormatFloat` (moved from `format.formatFloat`, which delegates); `Literal.String()` FloatLit branch uses it.
- `PrintAILANGSource`: remove false comment; non-finite float → error.
- **AC**: round-trip test (`4.0, 0.0, -4.0, 100.0, 1e20, 2.25`); `ailang test` on the #1448 repro 3/3; goldens inspected before regeneration.

### Close-out
Changelog fragment `changelogs/unreleased/2026-10-01-vm-pure-builtin-coverage.md`; `make test`, `make lint`, `make verify-examples`, `make check-file-sizes` green; final commit `Fixes #1447, Fixes #1448`.

## Order
M4 is independent (do first: smallest, de-risks goldens). Then M1 → M2 → M3 (M1's ratchet measures M2/M3 shrinking the allowlist).

## Outcome (2026-10-01)

- [x] M4 `50147a135`: #1448 repro 3/3; round-trip test; red before fix
- [x] M1+M2 `df5130741`: ratchet (3 directions mutation-checked), lockstep panic at init (mutation-checked), honest error tests; adapter covers 81 builtins
- [x] M3: 7 native ports, parity table (-count=20), extract-clamp mutation caught; #1447 repro `[3, 2, 1]` strict; example identical in all three engines
- Deviations: see design doc "Implementation Notes"
