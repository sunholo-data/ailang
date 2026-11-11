---
reviewed: 2026-11-11
reviewBy: 2027-02-11
---

# TC_NOMINAL_001: Two different types that share a name

**Code**: `TC_NOMINAL_001`
**Category**: Type checker
**Since**: the release after v0.60.0 (M-V1-NOMINAL-TYPE-IDENTITY)

## What this error means

A nominal type — an ADT such as `type T = A(int)` — is identified by **the module that
declares it and its name**. Two modules that each declare a `T` declare two different types,
however similar they look. This error is reported where a value of one is used where the
other is expected.

Up to v0.60.0 types were identified by name alone, so `check` accepted such programs and they
then failed at run time (`_str_upper: arg 0 - expected string, got int`) or silently took the
wrong `match` arm. The checker now rejects them at the line where the two meet.

## Example

```ailang file=shapes/a.ail
module shapes/a

export type T = A(int)

export pure func mk() -> T = A(42)
```

```ailang file=shapes/b.ail
module shapes/b

export type T = A(string)
```

```ailang error=TC_NOMINAL_001
module shapes/main

import shapes/a (mk)
import shapes/b (T, A)

pure func use(t: T) -> string = match t { A(s) => s }

export func main() -> () ! {IO} = println(use(mk()))
```

```text
Error: type error in shapes/main (decl 1): ... TC_NOMINAL_001: cannot unify T (defined in shapes/a)
with T (defined in shapes/b): they are different types that share a name.
  Suggestion: use the T from one module on both sides (import it by name from the module whose
  T you mean, e.g. import shapes/a (T)), or convert between them explicitly.
```

`mk()` returns `shapes/a`'s `T`; `use` takes `shapes/b`'s `T`, because that is the `T` this
module imported by name.

## How to fix

- **Use one type on both sides.** If `use` should accept what `mk` returns, import `T` (and
  its constructors) from the module that declares the type `mk` returns:
  `import shapes/a (T, A)`.
- **Convert explicitly** when the two types really are different, e.g. a function
  `toB(t: a.T) -> b.T` that pattern-matches one and builds the other.
- A **local** type does not stand in for an imported one either: declaring `type T = A(string)`
  in `main` and passing it `mk()` is the same error, naming `main` as one of the modules.

## What is not an error

- Using two same-named types **separately** — `Ma.desc(Ma.mk())` and `Mb.desc(Mb.mk())` with
  `import shapes/a as Ma (...)` and `import shapes/b as Mb (...)` — is fine.
- **Record aliases are structural**: `type R = {x: int}` declared in two modules is one type,
  as before.
- Interface hashes (`interface_hash`) do not change: a type still prints as its bare name.
