---
reviewed: 2026-11-26
reviewBy: 2027-02-26
---

# AILANG Testing Guide

For the full published testing guide (inline `tests [...]`, named `test "..." { ... }` blocks, `property` QuickCheck tests, shrinking, deterministic seeds, and package suites), see [Testing Guide](docs/guides/testing.md).

## Quick Reference

```ailang
module hello_test

export pure func square(x: int) -> int
  tests [(0, 0), (3, 9), (-4, 16)]
{
  x * x
}

test "addition works" { 1 + 1 == 2 }

property "addition is commutative" {
  forall(x: int, y: int) => x + y == y + x
}
```

## CLI Commands

```bash
ailang test                             # Run all tests in current directory
ailang test hello_test.ail              # Run a single file
ailang test --package .                 # Run package *_test.ail suite
ailang test --bytecode .                # Run named tests on bytecode VM
ailang test --json .                    # Emit JSON report to stdout
ailang test --seed 0 hello_test.ail     # Replay with deterministic seed
ailang test --random-seed .             # Draw and print a fresh master seed
```

## Key Rules

- **Pure test bodies:** Named `test` and `property` blocks are checked pure; test effectful entrypoints via `ailang run --caps ...`.
- **Multi-arg inline tests:** Wrap multiple parameters in a nested tuple (`tests [((1, 2), 3)]`), and use `()` for zero-arg functions (`tests [((), 42)]`).
- **Preconditions in properties:** Use `if cond then true else predicate` (there is no `==>` or `where` clause).
- **Deterministic seeds:** Property runs generate 100 cases from master seed `0` (`ailang-property-seed-v1`) and shrink failing inputs to minimal counterexamples.

## Resources

- **Published guide:** [docs/docs/guides/testing.md](docs/guides/testing.md)
- **Differential testing example:** `examples/runnable/property_differential_testing.ail`
- **Source:** `internal/testing/`
