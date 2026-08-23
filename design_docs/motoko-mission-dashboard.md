# Mission Dashboard — Motoko

*Snapshot, overwritten each iteration. History lives in the charter STATUS block and the mission log.*

**Last iteration**: 20 · 2026-08-23 · LANDED · evaluator PASS 90/100, zero blocking
**Release**: AILANG v0.33.1

## In flight / next

| | |
|---|---|
| Just landed | row **6d** — PR [#838](https://github.com/sunholo-data/ailang/pull/838) → [`50460040c`](https://github.com/sunholo-data/ailang/commit/50460040c): a HELD driver pin now reports a drifting source clone, once per doubling |
| Next pick | row **6e** — `test_motoko_connection_probe.sh` arm 33 hung a CI job 15m18s (re-run on a byte-identical tree: 88s success, so environment not code) |
| Then | row **7** — profile restoration design |
| Blocked on the rig | row **6** M2 — `AC-D1-live`, needs `rig.lock` and a GPU slot; V38 still not isolated |
| Phase-0 gated | rows 10–12 — re-measured 2026-08-23: G1 `#154` OPEN (control `#175` MERGED), G2 rc=128 w/ control rc=0, G3 registry `latest=2.2.0`. **CLOSED.** G4's *substance* passes (V30–V32) |

## Loop health

- Executing root: `~/.ailang-driver-pin/motoko` (pin worktree @ `origin/dev`, re-exec'd every fire).
- Source clone `~/dev/sunholo-data/ailang-motoko`: **170 behind**, 7 uncommitted files (all superseded — 125 of 129 added lines byte-present on `origin/dev`). **Do not start a session there.**
- Running skill == `origin/dev` (`cmp` rc=0, three ways: pin / `~/.claude` symlink target / origin).
- dev CI green at pick time; one cancelled job on our own PR, diagnosed environmental and re-run green.

## Routing

controller `claude:claude-opus-5` · designer rotation unspent · planner `opus` (fail-closed) ·
executor `codex:gpt-5.6-sol` · evaluator `sonnet`. **metered $0.00** this iteration (budget $5).

## Parked on Mark

- **`D-MOTOKO-WORKDIR-1`** — reconcile the source clone with `origin/dev`, discarding its 7
  uncommitted files? One word: **yes** / **no**. Gate 1's reconcile obligation 2 fails by
  construction and `pin-root.sh`'s own header says the first reconcile is human. Nothing is blocked
  meanwhile: the pin holds every fire and the drift is now visible on the bookkeeping issue.
