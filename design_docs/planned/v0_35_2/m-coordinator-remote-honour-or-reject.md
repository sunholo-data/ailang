# M-COORDINATOR-REMOTE: honour-or-reject `--remote` on all coordinator subcommands

**Status:** Planned · **Measured:** 2026-09-07 (prod) · **Scope:** CLI wiring only, `cmd/ailang/`

## Defect

`coordinator list|logs|diff|pending|status` accept `--remote gcp` and silently
act on the **local SQLite store**. Measured: `list --remote gcp` presented a
May-27 task that exists only on the operator's laptop as if it were prod;
`logs --remote gcp` returned `sql: no rows in result set` — it never consulted
the cloud. `approvals`, `approve`, `reject` already honour the flag. An
accepted-and-ignored flag is worse than a rejected one: operators make
decisions on fiction.

## Root cause

The five commands use hand-rolled `switch` flag parsers with **no `default`
case** (`coordinator_list.go:25`, `coordinator_lifecycle.go:298`,
`coordinator_inspect.go:44,150`), so `--remote gcp` is swallowed, then each
opens `coordinator.DefaultConfig()` + `NewSQLiteStore` (or a local daemon)
hardcoded. The working pattern lives in `coordinator_remote_store.go`
(`openCoordinatorStore`, `remoteCoordinatorSelected`), adopted at two call
sites: `coordinator_approvals_remote.go:34,174` and the pre-parse routing in
`coordinator_actions.go:26,140`.

## Recommendation: Option A — extend `openCoordinatorStore` to the remaining commands

Chosen over "loud error" because `openCoordinatorStore` already *is* the loud
option: it errors on unknown modes and missing project, and its contract
requires printing `store: <mode>` so the plane is always visible. A
reject-only fix would leave `list/logs/diff/pending` with no remote path at
all — a regression in capability, not just in honesty.

**Principle: honour or reject — never ignore.** Every coordinator subcommand
either routes `--remote` through `openCoordinatorStore` or exits non-zero
naming the flag.

## Design

1. **list, pending, logs, diff**: parse `--remote` (mirror the `flag.FlagSet`
   style of `coordinator_approvals_remote.go`), call
   `openCoordinatorStore(ctx, remote, stateDir)`, use `bundle.Store`, print
   `store: %s` before output. `logs`/`diff` resolve the task from
   `bundle.Store`, not a local path.
2. **status**: it queries the local *daemon* (PID file), not the store — there
   is no remote daemon to query. `--remote <non-local>` here exits non-zero:
   `status: --remote not supported (daemon is local); use list/pending for
   remote state`. Rejecting is honouring the principle.
3. **Strict parsing guard**: add a `default:` case to the five hand-rolled
   parsers returning `unknown flag: %s`. This is what turns today's silent
   swallow into a rejection for any *future* flag too. Small, same files.
4. **Out of scope:** `approve`/`reject`/`approvals` (already correct — do not
   touch); no CLI redesign; no daemon changes.

## Acceptance criteria (all FAIL today)

1. `coordinator list --remote gcp` prints a `store: gcp (project …)` header
   and its rows contain no task IDs absent from prod Firestore (today: shows
   laptop-only May-27 task, no header).
2. `coordinator logs --remote gcp <id>` queries Firestore; the string `sql:
   no rows in result set` cannot appear (today: it does).
3. `diff` and `pending` with `--remote gcp` print the `store:` header and read
   from Firestore.
4. `coordinator status --remote gcp` exits non-zero naming `--remote` (today:
   prints local daemon status, exit 0).
5. `coordinator list --bogus` exits non-zero naming `--bogus` (today: silently
   ignored).
6. Regression: `approvals --remote gcp`, `approve`, `reject` outputs
   byte-identical to pre-change on a fixture store.

## References

- Pattern: `cmd/ailang/coordinator_remote_store.go`, `coordinator_approvals_remote.go`
- Call sites that adopted it: `cmd/ailang/coordinator_actions.go:26,140`
- Defective parsers: `coordinator_list.go`, `coordinator_lifecycle.go:293`, `coordinator_inspect.go:17,121`
- Related inbox item: "unroutable messages accepted silently" — same defect class, messaging plane.
