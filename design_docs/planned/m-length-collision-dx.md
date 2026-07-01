## M-LENGTH-COLLISION-DX: confusing type error when `length` (std/list) is called on a string

**Status**: ✅ IMPLEMENTED (2026-07-01, v0.28.0). When an application fails to unify and the
callee is a bare name exported by >1 stdlib module (e.g. `length` in std/list, std/string,
std/array, std/bytes), the error now carries a name-collision note + alias hint:
`note: 'length' is also exported by std/array, std/bytes, std/string — … import it with an
alias, e.g. 'import std/array (length as lengthAlt)'`. The note lives in the application
constraint's `Path`, so it surfaces ONLY on failure (invisible on correct use — verified
`length([1,2,3])` is unaffected). Impl: `collisionHint` in `internal/types/import_hint.go` +
`coreCalleeName` wiring in `internal/types/typechecker_functions.go`; test in
`import_hint_test.go`. Kept as a *hint* (not a rename), per the reporter's preferred option 2.

--- original ---
**Status (orig)**: PLANNED (DX bug)
**Target**: v0.28.0
**Priority**: P2 (Medium — ~5 min cold-start friction per the reporter; hits anyone importing both std/list and std/string)
**Estimated**: 0.5–1 day
**Dependencies**: None.

**Reported from**: Felix / Snake Showdown build team via `submit_feedback` (ticket `fb_942b7f3dff3d8e52`), AILANG v0.27.0. Contact: kevin.faurholt@gmail.com.

---

## Problem

Both `std/list` and `std/string` export a function named `length` with different signatures
(`[a] -> int` vs `string -> int`). Importing `length` from `std/list` and calling it on a
string produces a raw unification error that names neither the collision nor the fix:

```ailang
import std/list (length, filter, take)
import std/string (repeat)

let valLen = length("hello")   -- length resolves to std/list.length
```
```
type unification failed: cannot unify list[α292] with string
```

The error points at the call site but gives no hint that (a) `length` is bound to the
`std/list` version, or (b) `std/string` also exports `length` and `import std/string (length as strLen)`
is the fix. The reporter lost ~5 min to this.

## Why the generic error can't help today

The unifier ([internal/types/types.go:547](../../internal/types/types.go#L547),
`UnificationError`) only has two types (`list[α]` vs `string`). It doesn't know the callee is
named `length`, nor that another stdlib module exports a same-named function that *would* unify.
So the hint must be added a layer up — at **application/call-site inference**, where the callee
identifier and the imported/stdlib environment are both in scope.

## Proposed fix (non-breaking — the reporter's option 2)

At the call-site type-check, when a function application fails to unify the argument against the
callee's parameter type, AND the callee is a bare identifier (e.g. `length`) that is *also*
exported by another stdlib module with a signature that WOULD unify, attach a hint:

```
type mismatch: `length` (imported from std/list) expects [a], but the argument is `string`.
  std/string also exports `length` (string -> int).
  Fix: import std/string (length as strLen)  and call strLen("hello")
```

Implementation notes:
- The elaborator/inference already resolves `length` to its module (`std/list`) — thread that
  origin into the application node so the error path can name it.
- On unify-failure at an application, consult the stdlib interface registry (`internal/iface`)
  for other modules exporting the same name whose parameter type unifies with the actual arg.
  Emit at most one suggestion (prefer std/string for string args, std/array for arrays, etc.).
- Keep it a *hint* on the existing error, not a new hard error — no behavior change.

**Rejected:** renaming `std/string.length` → `strLen` (reporter's option 1). Breaking, and
`length` is the natural name for both; the collision is inherent to a good stdlib, so the fix is
better diagnostics, not a rename.

## Acceptance Criteria

- [ ] `length("hello")` (with `length` from std/list) yields an error that names the std/list
      origin and suggests `import std/string (length as strLen)`.
- [ ] No suggestion is emitted when the argument genuinely has no same-named alternative.
- [ ] The generic unifier is unchanged; the hint is added only at application inference.
- [ ] A test covering the std/list-vs-std/string `length` case.
