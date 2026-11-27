# Iteration 146: gates for #2481 (rebased, hardened, merged)

Base: origin/dev `aba6f282085c5e81d6bfb52225721a94429f21c8` (Gate 1, 2026-11-27T15:48:47Z). The required checks (`test`, `lint`, `build`, `docs-gate`, `checks`, read from branch protection) were all green there. `test-windows` (not required) was red.

## Rebase

`ccee1caca` → rebased onto `aba6f2820`: 2/2 commits patch-identical (`git patch-id --stable`: `18b1508a…` and `37e2309c…` on both sides).

## Surviving mutants, reproduced by the controller before routing

Each mutant is a copy in /tmp. rc 0 means the plain test passed, so the mutant survived.

| Mutant | Change | rc on `0cfaa88f4` | rc on `76518ae5c` |
|---|---|---|---|
| nowhole | `def whole: type=="number" and . == floor;` → `def whole: true;` | 0 (survived) | 1: `FAIL nightly fractional issue number was accepted` |
| idonly | `"nightly comment:\($issue):\(.id)"` → `"nightly comment:\($issue)"` | 0 (survived) | 1: `FAIL nightly replaced report did not move hash` |
| m6 (judge's) | `startswith("The nightly ")` → `test("nightly")` | n/a | 0 (survives; non-blocking) |

## Controller gates outside the sandbox

| Gate | Base `0cfaa88f4` (rebased) | After executor `76518ae5c` |
|---|---|---|
| `test_mission_idle_fingerprint.sh` | rc 0 | rc 0 |
| `test_mission_idle_fingerprint.sh --mutations` | rc 0, 27 KILLED | rc 0, 29 KILLED |
| `shellcheck` helper + test | rc 0 | rc 0 |
| helper byte-identical to the base | n/a | yes (`cmp`); the change is test-only |
| `make test-launchd-drivers` | n/a | rc 0, `run-suites: all 62 suites passed` (at `8c69675fa`) |

## Done-gate (mission-loop-change pre-flight)

- Surface: `tools/launchd/lib/idle-fingerprint.sh`, the one implementation behind the driver's idle pre-check, `--snapshot` and `--record`.
- Reach: merged to origin/dev as `7b67c35b4`. Every mission re-execs from a worktree pinned to origin/dev.
- Merge tree `da4a3cd0e778…` == judged head `8c69675fa`'s tree (byte-identical; merge parent `aba6f2820`).
- Healthy dry-run (bare read mode, read-only, against the registry's source clones, which carry `origin/HEAD`):
  - world (`sunholo-data/ailang-world`): old and new helper both `c2b69ff9b9e0…`, byte-identical (no open nightly-rig issue);
  - stapledon (`sunholo-data/stapledons-godot`): old `7cceadd6302b…` → new `b96a2f3ea350…`; summary line unchanged (`head=beb17c4 ledger=169 open=10 directives=0 inbox=0 red=0`);
  - `~/.ailang/state/mission-{world,stapledon}-idle-fingerprint*` listings unchanged before and after.
  - First attempt against the pin worktrees: both helpers rc 1, `charter read failed … does not exist in 'origin/dev'`, because pin worktrees have no `origin/HEAD` and the helper falls back to `dev`. Instrument setup, not a verdict.
- Degraded dry-run: a `gh` stub failing only the `labels=nightly-rig` read makes the new helper exit rc 1 with no fingerprint and `idle-fingerprint: nightly-rig issues read failed: HTTP 502 (degraded drill)`. The driver's pre-check then logs `cannot read idle fingerprint … proceeding with a full fire` (`tools/launchd/mission-control.sh`, MISSION IDLE PRE-CHECK), so a failed read fails loud and never skips a fire.

## CI

See [ci.txt](ci.txt). PR head `8c69675fa`: 24/24 checks completed, 0 not-green. Dev merge `7b67c35b4`: 18/18 completed. Every required check is green, plus `launchd drivers (bash 3.2)`. `test-windows` failed on `TestRunSmokeInTempDir_Timeout`. That red is inherited: the merge touches 0 `.go` files, `CI` concluded `failure` on each of the 7 dev commits before it (back to `e5b18e2d7`), and the parent `aba6f2820`'s `test-windows` failed on a different set (`TestCheckPackageReportsDeprecations`, `TestReferenceSolutions_JS`, `TestStdlibRenamesEvaluatorTranscript`, …). V1 owns dev reds.
