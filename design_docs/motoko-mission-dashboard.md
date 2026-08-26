# Mission Dashboard — Motoko

_Snapshot after iteration 24 (2026-08-26). Overwritten each iteration; history lives in the charter STATUS block and the log._

**Release**: AILANG v0.33.2 · anchor `sunholo-data/ailang` (shared with V1; V1 owns dev CI red on it)

## In flight / next
- **Just landed** — row **6g**: `run_bounded` and production `run_lane` killed the wrapper PID, leaving a
  hung grandchild alive at `PPID 1`. PR [#892](https://github.com/sunholo-data/ailang/pull/892) →
  [`fd1fa9e01`](https://github.com/sunholo-data/ailang/commit/fd1fa9e01). Guarded process-group kill via
  `set -m`; suite 39 → **40 arms**. Evaluator round 1 **PASS 82/100, zero blocking**.
- **Next** — row **6h**: a provider failure parses as a successful empty completion; the reporter's guard is
  **not expressible** (`ChatStepResponse.Usage` is a value type, so `absent == zeroed`). Reaches the ollama
  `/v1` lane, i.e. our own rig.
- **Then** — row **6i** (new, from this iteration's own drill): the **production** `run_lane` group-kill has
  **zero test coverage** — reverting the whole hunk leaves the suite green 40/40; its only gate is `bash -n`.
  That is the half row 6g called "the one that matters".

## Gated / parked
- Phase 0 **CLOSED**: `arniwesth/motoko_agent#154` still OPEN (re-measured as a command; control `#175`
  MERGED) and G5 needs Arni's word. Rows 10/11/12 parked; rows 9/13/14 wait on a green tree.

## Loop health / routing
- Controller opus · executor `codex:gpt-5.6-sol` (probe rc=0) · evaluator **sonnet, own worktree**
  (generator≠judge) · no designer/planner/quorum (row names its own scope) · rotation at fable, **unspent**.
- Metered **$0.00** of $5. No GPU. Gates on darwin/arm64 — the ONLY platform the `launchd drivers (bash 3.2)`
  CI job runs on, deliberately, so the local green IS the CI leg.
- **Executor run 1 was wasted by MY gate list, not the lane**: the full suite is unsatisfiable under codex
  `--sandbox workspace-write` (arm 33's loopback bind kills the session). Re-issued, re-run outside.
- **`~/go/bin/ailang` silently ignores `AILANG_MESSAGES_STORE`** (invalid value → rc=0). Fresh binary built
  to read the canonical cloud inbox: **62** unread there vs 12 locally.
- **dev CI: SonarCloud red, handed to V1** — *52.8% coverage on new code (≥80% required)*, green at
  `6193bb712`, red from `6759ea4fa` (V1's messaging-store change). Not required, not ours.
- Source clone: **46 behind / 0 ahead**, clean — up from 35, above the notice threshold for a third
  consecutive iteration.

## Parked on Mark
- **`D-MOTOKO-WORKDIR-2`** (OPEN since iteration 21 — **third** ask): grant **standing** authorization to
  reconcile the source clone to `origin/dev` unattended when three measured predicates hold? One word:
  **yes** or **no**. Drift has gone 0 → 24 → 35 → 46 in four iterations.
