---
sidebar_position: 2
title: Upgrading from 0.x to 1.0
description: Breaking changes, compiler diagnostics, and mechanical fixes for migrating AILANG 0.x programs and packages to 1.0.0.
reviewed: 2026-11-26
reviewBy: 2027-02-26
---

# Upgrading from 0.x to 1.0

`1.0.0` is the final `0.x` pre-soak release (`v0.64.0`) plus documentation updates — there are no new language breaks between `v0.64.0` and `1.0.0`. Anyone upgrading from `v0.57.x` or earlier (or from `v0.58.0`–`v0.63.0`) should run the check-and-fix loop below before publishing or pinning `1.0.0`.

## Quick Migration Workflow

1. **Apply compiler quick-fixes** (missing `! {Effects}`, string `++` → `"${...}"`, `TC_MATCH_002` stubs, `= {` block syntax, missing stdlib imports):
   ```bash
   ailang check --fix path/to/file.ail
   ```
2. **Rewrite the 12 uniquely named stdlib renames** (for codebases still using pre-`v0.64.0` snake_case names):
   ```bash
   sed -E -f tools/migrate-stdlib-renames.sed -i.bak path/to/*.ail
   ```
   Update `std/fs.glob` → `walkWithSuffix`, `std/bytes.slice` → `sliceLen`, `std/list.member(x, xs)` → `contains(xs, x)` (note swapped argument order), and `std/array.unsafeGet` → `get` by hand.
3. **Check, test, and re-lock** the whole package:
   ```bash
   ailang check --package .
   ailang test --package .
   ailang lock
   ```

See the [Stability Promise](/docs/reference/stability) for what is covered across `1.x` and what remains Experimental or Internal.

---

## Breaking Changes (`v0.58.0` → `1.0.0`)

Each entry states what changed, the diagnostic you will see, the mechanical fix, and the version where it landed. For full release notes, see [CHANGELOG.md](https://github.com/sunholo-data/ailang/blob/dev/CHANGELOG.md).

### 1. `pure func` requires an empty effect row (`! {}`)

- **What changed:** `pure` means `! {}`. Declaring `pure func` alongside a non-empty or open effect row (`! {IO}`, `! {Declassify}`, `! {e}`) is a type error, and `ailang iface` derives `"pure"` from the checked effect row rather than the keyword.
- **Diagnostic:** `The pure keyword contradicts the declared effect row ! {IO}. Suggested fix: remove pure to keep the effect row, or remove the effects and use ! {} with a pure body.`
- **Mechanical fix:** Drop `pure` on functions that declare effects (or run `ailang check --fix`, which removes contradictory `pure ` when stamping missing effects).
- **Since:** `v0.58.0` (`#1443`).

```ailang
module docs/upgrade_pure

-- Before (rejected): pure func sanitize(...) -> string<sanitized> ! {Declassify}
-- After: drop `pure` when keeping the effect row
export func sanitize(raw: string<email>) -> string<sanitized> ! {Declassify} = "[sanitized]"
```

### 2. Cross-module IFC sink & label enforcement

- **What changed:** Information-flow control (IFC) sink refinements (`T{not LABEL}`) and structural labels inside lists, records, tuples, and ADTs are enforced across module boundaries, and sink constraints travel with function values through aliases, higher-order calls, and wrappers.
- **Diagnostic:** `information-flow violation: value labelled <...> reaches parameter "..." which forbids it` or `param_label_cover`.
- **Mechanical fix:** Route tainted values through an explicit `! {Declassify}` function before passing them to imported sinks, and annotate exported library sink parameters explicitly.
- **Since:** `v0.58.0` (`#1134`).

### 3. Named tests require `bool` for non-final bare expressions

- **What changed:** Every bare expression inside a named `test "..." { ... }` block is evaluated as an assertion and must have type `bool` (non-final expressions were previously ignored).
- **Diagnostic:** Type error expecting `bool` on a non-final expression in a `test` block.
- **Mechanical fix:** Bind non-boolean setup expressions with `let _ = expr` (or `let val = expr`).
- **Since:** `v0.58.0` (`#604`).

### 4. `ailang serve` requires auth on non-loopback binds (Operators / Internal)

- **What changed:** Dashboard ingest and server endpoints require authentication tokens (`AILANG_OTLP_INGEST_TOKEN`, `AILANG_HUB_TOKEN`) when bound outside loopback.
- **Diagnostic:** HTTP `401` / startup auth error on non-loopback `ailang serve` binds without tokens configured.
- **Mechanical fix:** Configure the required bearer tokens in the server environment or bind to loopback only. (`ailang serve` is **Internal** tier.)
- **Since:** `v0.58.0`.

### 5. Nominal type identity & strict type declarations

- **What changed:** Same-named ADTs and records in different modules are distinct nominal types keyed by canonical module identity; a single-constructor ADT wrapping a record (`type Box = Box({x: int})`) is distinct from the bare record (`#2239`); and types may not shadow built-ins (`PAR_TYPE_DECL_BUILTIN`) or form direct recursive aliases (`TC_TYPE_ALIAS_CYCLE_001`).
- **Diagnostic:** Type mismatch between `modA.T` and `modB.T`, `PAR_TYPE_DECL_BUILTIN`, or `TC_TYPE_ALIAS_CYCLE_001`.
- **Mechanical fix:** Import the canonical type declaration from the defining module rather than redeclaring an identical ADT locally; unwrap single-constructor newtype wrappers explicitly when a structural record is expected.
- **Since:** `v0.60.0`–`v0.60.4` (`#2228`, `#2239`).

### 6. Exhaustive `match` is required

- **What changed:** Non-exhaustive `match` expressions on ADTs, lists, `Option`, `Result`, and primitive types are compile-time errors (guarded arms do not count toward coverage). Ambiguous constructor patterns whose type cannot be inferred fail explicitly instead of guessing.
- **Diagnostic:** `TC_MATCH_002: non-exhaustive match on <Type> — no arm matches: <pattern>`, `TC_MATCH_004` (match too large to check), or `TC_MATCH_007` (untyped constructor pattern).
- **Mechanical fix:** Run `ailang check --fix` to insert `<pattern> => _?` stubs, add the missing arms or a final `_ => ...` arm, or use `unreachable("reason")` for impossible branches (stops with `RT010` if reached).
- **Since:** `v0.60.2`–`v0.60.6` (`#2246`, `#2303`).

```ailang
module docs/upgrade_exhaustive

type Light = Red | Amber | Green

export pure func nextLight(l: Light) -> Light =
  match l {
    Red => Green,
    Amber => Red,
    Green => Amber
  }
```

### 7. Type-checker & module strictness; Go emitter & `extern func` removed

- **What changed:**
  - Type variables are resolved by lexical scope rather than identifier case (`#2211`), signature type variables are rigid (`TC_RIGID_001`, `#2273`), and function-literal parameter/return annotations are enforced (`#2238`).
  - Named `test` bodies and `requires`/`ensures` contracts are type-checked (`#2232`, `#2248`).
  - Typeclass constraints (`Num`, `Eq`, `Ord`, `Show`) cross module boundaries (`TC_CONSTRAINT_001`, `#2290`), and pathological type expansions are bounded (`TC_TYPE_SIZE_001`, `TC_TYPE_WORK_001`).
  - `extern func` declarations are rejected (`EXT004`, `#2390`), and the Go emitter (`ailang compile --emit-go`) was removed (`#2389`) so `1.x` ships two aligned engines: the reference evaluator and `--bytecode` VM.
- **Diagnostic:** `TC_RIGID_001`, `TC_CONSTRAINT_001`, `TC_TYPE_SIZE_001`, `TC_TYPE_WORK_001`, `EXT004`, or `ailang compile` exit 1 pointing to `ailang run` / `ailang run --bytecode`.
- **Mechanical fix:** Fix ill-typed contracts/tests, add missing `deriving (Eq)` for `==` in contracts, introduce nominal `type` wrappers for huge structural types, and migrate `compile --emit-go` callers to `ailang run` / `--bytecode` (or JSON over stdin/stdout via the [Go interop guide](/docs/guides/go-interop)).
- **Since:** `v0.60.4`–`v0.60.6`.

### 8. `show` is total and quotes nested strings

- **What changed:** Built-in `show` no longer truncates long values or omits elements; top-level `show("abc")` returns `abc`, while strings inside lists, tuples, records, maps, and constructors are quoted (`show(["a", "b"])` → `["a", "b"]`).
- **Diagnostic:** No compile error; runtime display output changes if callers relied on unquoted nested strings or automatic truncation.
- **Mechanical fix:** Use `std/string.showPreview(val, maxChars)` for bounded human-readable log previews.
- **Since:** `v0.60.3` (`#2229`).

### 9. Stdlib renames removed, `Json` `JInt`, and `std/ai` `Message.media`

- **What changed:**
  - The 16 legacy `@deprecated` stdlib aliases (deprecated in `v0.60.3` with `TC_DEPRECATED_001`) are **removed in `v0.64.0`**.
  - `std/json` decodes 64-bit integers as `JInt(int)` instead of `JNumber(float)`, `decode` rejects trailing non-whitespace tokens, and `std/ai.Message` renamed `images` → `media` (`v0.61.0`).
- **Diagnostic:** `IMP010: symbol '...' not exported by 'std/...'` naming the replacement; `TC_MATCH_005` warning on `Json` matches with `JNumber` and no `JInt`.
- **Mechanical fix:** Run `sed -E -f tools/migrate-stdlib-renames.sed` and update `glob` (`walkWithSuffix`), `slice` (`sliceLen`), `member(x, xs)` (`contains(xs, x)`), `unsafeGet` (`get`) by hand. For `Json` numbers, match both `JInt` and `JNumber` or use `getInt` / `getNumber` / `asNumber`. Update `Message.images` to `Message.media`.
- **Since:** `v0.60.3` (`JInt`), `v0.61.0` (`Message.media`), `v0.64.0` (16 aliases removed).

### 10. Verification exit codes & bytecode contract refusal

- **What changed:** `ailang verify`, `ailang check --verify`, and `ailang ai-check` exit non-zero when obligations exist and Z3 is missing (`unavailable`), `unproved`, `inconclusive`, `error`, or `counterexample`. `ailang run --bytecode --verify-contracts` is refused before execution instead of silently skipping runtime checks.
- **Diagnostic:** `CTR004: --verify-contracts is not enforced on the bytecode VM yet` (exit 2), or exit 1 from `check --verify` / `verify`.
- **Mechanical fix:** Install Z3 (`AILANG_Z3_PATH`), pass `--allow-inconclusive` when bounded recursion is expected, or drop `--bytecode` when using `--verify-contracts`. See [Contracts & Verification](/docs/guides/contracts#exit-codes).
- **Since:** `v0.59.0`–`v0.61.0`.

### 11. Integer wrap semantics, Z3 overflow caveats, and numeric literal typing

- **What changed:**
  - Runtime `int` arithmetic is signed 64-bit two's-complement wrapping (`9223372036854775807 + 1 == -9223372036854775808`). `ailang verify` models `int` as unbounded SMT `Int` and reports `int_overflow` caveats (use `--int-model=int64` to treat possible wraps as violations) and `float_nan_or_inf` violations.
  - Standalone numeric bindings whose integer literal does not appear in a parameter position of the binding's type (`let k = 3`) are fixed to `int`, and `let ... in` / block bindings follow ML's value restriction (`#2378`).
- **Diagnostic:** Type mismatch using `k * 0.5` on `let k = 3`; `int_overflow` caveat or violation in `ailang verify`.
- **Mechanical fix:** Write `3.0` (or `intToFloat(k)`) for floats, and bound `int` inputs in `requires` when verifying arithmetic with Z3.
- **Since:** `v0.60.6` (`#2378`), `v0.61.0` (`M-VERIFY-NUMERIC-MODEL`).

### 12. Callback effect rows (closed `! {}` vs open rows)

- **What changed:** An unannotated callback type (`T -> U`) has an implicitly open effect row that charges callers (including callbacks nested inside lists, tuples, records, or `Option`). Passing an effectful callback to a pure higher-order function expecting `T -> U ! {}` (like `std/list.map`) is rejected at compile time.
- **Diagnostic:** `Missing effects: IO` (or similar effect mismatch) at the call site.
- **Mechanical fix:** Use effect-polymorphic helpers (`mapE`, `filterE`, `foldlE`, `flatMapE`, `forEachE` from `std/list`) for effectful callbacks, or annotate `! {Effects}` on the caller.
- **Since:** `v0.55.0` (`#1718`, `#1826`) and `v0.61.0`.

### 13. What is NOT covered by the 1.x Stability Promise

The 1.x promise covers the Stable language syntax, the 17 Stable stdlib modules, the 19 Stable CLI commands, the package format (`ailang.toml`, `ailang.lock`, registry resolution), and the 24 Stable hosted MCP tools. It does **not** cover:

- Go packages under `internal/` (there is no Go library API promise).
- **Experimental** stdlib modules (`std/ai`, `std/net`, `std/process`, `std/stream`, etc.), Experimental functions in Stable modules (`@experimental`), or Experimental CLI commands (`fmt`, `lsp`, `verify`, `serve-api`, `messages`).
- **Internal** CLI commands (`eval`, `chains`, `coordinator`, `mission`, `serve`, `iface`, etc.).

---

## Checklist for Package Authors

1. Update `ailang` minimum constraint in `ailang.toml` (e.g. `ailang = ">=0.64.0"` or `">=1.0.0"` once `1.0.0` is tagged).
2. Run `ailang check --package .`, `ailang test --package .`, and `ailang verify --package .` (if using contracts).
3. Run `ailang lock` and `ailang lock --check`.
4. Add a `## <version>` section to your package `CHANGELOG.md` and set `[release].kind` accurately (`PUB025` checks `[release].kind` against measured interface changes).
5. Run `ailang publish --dry-run`, then `ailang publish`. Dependent packages notified through the upgrade cascade can resolve the new version cleanly once published.
