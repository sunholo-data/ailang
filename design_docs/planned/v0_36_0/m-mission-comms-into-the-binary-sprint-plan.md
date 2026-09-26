# Sprint Plan: M-MISSION-COMMS Phase 1 (Go half only)

**Design doc**: [m-mission-comms-into-the-binary.md](m-mission-comms-into-the-binary.md)
**Sprint ID**: M-MISSION-COMMS-P1
**Duration**: 2 days · **Risk**: low · **Est. LOC**: ~810 (390 impl + 420 test)
**Status**: planned

## Why this scope, and not Phase 1 as written

The design doc's Phase 1 ends by swapping the six `gh issue comment` sites in
`tools/launchd/mission-control.sh`. **This sprint deliberately stops short of that**, because the
loops are live:

| Loop | Interval | Runs from |
|---|---|---|
| v1 | **90 min** (one was mid-fire when this plan was written) | `~/.ailang-driver-pin/v1` |
| world | 4 h | its own repo |
| docs | 6 h | `~/.ailang-driver-pin/docs` |
| motoko | 13 h | `~/.ailang-driver-pin/motoko` |

The three pinned copies re-sync from `origin/dev` at **every fire** (`pin-root.sh:114` fetches
before resolving the ref). So a driver change pushed to `dev` reaches three live loops within
90 minutes, unreviewed. That is not a risk worth taking for a refactor.

**HARD CONSTRAINT — this sprint modifies no file under `tools/launchd/` or `scripts/`.**
Everything here is additive Go plus one bounded-exec fix. The shell cutover becomes its own
sprint, gated on this one landing and on a deliberate cutover window.

Everything in scope is also valuable under **any** HD-1 ruling: a bounded client, a mission-identity
façade and a report renderer are needed whether the charter or GitHub is canonical. The
HD-dependent pieces (decision→issue projection, telemetry re-route, the board) are **out of scope**.

## Milestones

### M1 — Bound `GitHubClient`'s exec (closes P-1)

Quorum round 3 blocked partly on this and it is verified (V18): `defaultExecCommand` calls
`exec.Command(...).CombinedOutput()` with **no context**, so a hung `gh` wedges the caller
indefinitely. Today only 1 of the 6 driver call sites is bounded (`_mc_bounded 30`); porting to Go
without this would make things *worse*, not better.

- Add `execCommandCtx` using `exec.CommandContext` with a per-call deadline.
- Deadline configurable via `GitHubConfig`, defaulting to **30s** to match the existing
  `_mc_bounded 30` semantics the driver already relies on.
- A timeout must return a **typed, identifiable error** — not a generic exec failure — so callers
  can distinguish "GitHub is slow" from "gh is broken" (Critical Principle 2).
- `execCommand` is already a struct field, so tests inject a stub; no new seam needed.

**LOC**: ~60 impl + ~90 test
**Acceptance:**
- A stub that sleeps past the deadline returns the typed timeout error, and returns *promptly*.
- The default deadline is 30s and is overridable via config.
- Every existing `internal/messaging` test still passes (no behaviour change on the happy path).

**Mutation arm (must turn the test RED):** revert `execCommandCtx` to the unbounded
`exec.Command` → the timeout test hangs or fails.

### M2 — `internal/mission/comms`: identity façade + report renderer

- `Client`: wraps `messaging.GitHubClient` with mission identity (name, repo, issue) so call sites
  stop passing `MISSION_GH_ISSUE` around as a bare env string.
- `RenderReport`: the ≤400-char iteration report — what landed, goal distance, cost, log link.
  Over-length input must be **truncated deterministically**, never silently dropped, and never
  emitted over cap.
- Table-driven tests across all four mission identities (v1, world, docs, motoko).

**LOC**: ~200 impl + ~180 test
**Acceptance:**
- `RenderReport` output is ≤400 chars for every fixture, including a deliberately over-long one.
- Deterministic: same input → byte-identical output (A1).
- All four mission identities resolve to the right repo/issue.

**Mutation arms:** (a) raise the cap to 4000 → cap assertion fails; (b) make truncation
length-dependent on map iteration → determinism assertion fails.

### M3 — `ailang mission report` subcommand

- New `cmd/ailang/mission_comms.go`, registered in `main.go` beside `messages`/`chains`/`design-quorum`.
- `ailang mission report --mission <name> --body-file <path> [--dry-run]`.
- **`--dry-run` prints exactly what would be posted and exits 0 without touching the network** —
  this is what makes the later shell cutover safe to rehearse against the live thread.
- Failure is loud: a post that cannot be delivered exits non-zero with the typed error from M1.

**LOC**: ~130 impl + ~150 test
**Acceptance:**
- `--dry-run` performs zero network calls (asserted with a stub that fails if invoked).
- Unknown mission name is a clear error, not a silent default.
- `ailang mission --help` lists the subcommand.

**Mutation arm:** make `--dry-run` fall through to the real post → the zero-network assertion fails.

## Out of scope (explicit)

- Any edit to `tools/launchd/*.sh` or `scripts/mission_*.sh`. Not one line.
- Decision→issue projection (blocked on **HD-1**).
- Telemetry re-route (blocked on **HD-2**; also needs the V17 hardening).
- Projects board (**HD-4**).
- The six-call-site cutover — its own sprint, its own window.

## Execution constraints

1. **Work in a git worktree**, not the main checkout. A v1 iteration runs every 90 minutes with
   `--permission-mode bypassPermissions` over the shared tree; uncommitted work there can be
   swept into a loop's commit or confuse its Gate 1 status check.
2. **Land via PR**, not a direct push to `dev`.
3. `make test` and `make lint` before the PR. `make test-launchd-drivers` must also stay green —
   it is the suite that guards the driver this sprint is deliberately not touching.

## Success metrics

- [ ] 3 milestones complete, all acceptance criteria met.
- [ ] 5 mutation arms each demonstrated RED before being recorded green.
- [ ] `git diff --name-only` on the PR touches **no** file under `tools/launchd/` or `scripts/`.
- [ ] P-1 closed in the design doc's open-objections table.
- [ ] `make test`, `make lint`, `make test-launchd-drivers` green.
- [ ] CHANGELOG.md updated.
