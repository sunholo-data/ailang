## M-VERIFY-STDLIB-STALE-PATH: `make verify-stdlib` points at a directory that no longer exists

**Status**: PLANNED (bug — dead tooling)
**Target**: v0.27.x
**Priority**: P3 (Low — not in CI; but it's a silently-dead stability gate)
**Estimated**: ~1 hour
**Dependencies**: None.

**Found during**: M-SNAKE-FEEDBACK (adding exports to `std/list.ail` / `std/io.ail`). Verified on v0.26.2.

---

## Problem

`tools/verify-stdlib.sh` (run by `make verify-stdlib`) reads `STDLIB_DIR="stdlib/std"`:

```
$ ailang iface stdlib/std/io.ail
Error: cannot read file 'stdlib/std/io.ail': open stdlib/std/io.ail: no such file or directory
```

The stdlib moved to `std/` (repo root) long ago — "as of v0.3.20, stdlib files are in std/"
([internal/pipeline/stdlib_canary_test.go:22](../../../internal/pipeline/stdlib_canary_test.go#L22)).
`stdlib/std/` does not exist. So `make verify-stdlib` fails for **everyone**, not because an
interface changed but because it can't find any module. The golden files in `.stdlib-golden/`
(`io`, `list`, `option`, `result`, `string`) were frozen against the old layout and are now
unreachable by the verifier.

It is **not** wired into `make ci` or `make ci-strict`, so this has been dead-but-silent: a
stdlib interface-stability gate that hasn't actually verified anything since the v0.3.20 move.
Adding the M-SNAKE-FEEDBACK exports (`flush`/`printErr`/`eprintln` to `std/io`,
`nth_or`/`head_or`/`last_or` to `std/list`) is exactly the kind of change this gate exists to
flag — and it couldn't.

## Fix

1. Repoint `tools/verify-stdlib.sh` and `tools/freeze-stdlib.sh` (if it shares the path) at
   `std/` instead of `stdlib/std/`.
2. `make freeze-stdlib` to regenerate `.stdlib-golden/*.{json,sha256}` from the current `std/`
   modules (which now include the M-SNAKE-FEEDBACK additions).
3. Decide whether to add `verify-stdlib` to `ci-strict` so interface drift is actually gated
   (it claims to be a stability gate; today it gates nothing). If added, ensure it's tolerant
   of *additive* changes or document that any export change requires a deliberate re-freeze.

## Acceptance Criteria

- [ ] `make verify-stdlib` reads `std/` and passes against the current stdlib.
- [ ] `.stdlib-golden/` re-frozen to include the v0.27.0 stdlib additions.
- [ ] A decision recorded on whether `verify-stdlib` joins `ci-strict` (and if so, it's green).
