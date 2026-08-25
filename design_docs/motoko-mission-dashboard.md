# Mission Dashboard — Motoko

_Snapshot after iteration 22 (2026-08-25). Overwritten each iteration; history lives in the charter STATUS block and the log._

**Release**: AILANG v0.33.2 · anchor `sunholo-data/ailang` (shared with V1; V1 owns dev CI red on it)

## In flight / next
- **Just landed** — row **6e**: the probe self-test could hang for 15 minutes and be cancelled with zero
  diagnostics. Every arm now has a hard cap; `descendant_pids` is bounded by node count as well as by the
  clock. PR #871 → `086b72184`. Evaluator 54 → **91/100**. Suite 34 → 39 arms.
- **Next** — row **6f**: triage-lite the 8 open issues no mission doc mentions (2 are ours: #842, #839).
- **Then** — row **6g** (new): `run_bounded` *and* production `run_lane` kill the wrapper PID, not the
  process group, so a hung grandchild survives. Filed from the judge's non-blocking finding.
- **Then** — row 7 (profile restoration design), row 8 (repin stale OpenRouter models).

## Gated / parked
- Phase 0 remains **CLOSED**: `arniwesth/motoko_agent#154` still OPEN (control `#175` MERGED), and G5 needs
  Arni's ABI-settled word. Rows 10/11/12 stay parked. Rows 9/13/14 wait on a green tree.

## Loop health / routing
- Controller opus · executor `codex:gpt-5.6-sol` (**capped at 30 min this iteration, FLAGGED**) ·
  evaluator sonnet in its own worktree · designer rotation untouched at fable, **unspent**.
- Metered **$0.00** of $5. No GPU, no `rig.lock`.
- Source clone `~/dev/sunholo-data/ailang-motoko`: **24 behind / 0 ahead**, clean — just under the notice
  threshold of 25, drifting again exactly as `D-MOTOKO-WORKDIR-2` predicted.

## Parked on Mark
- **`D-MOTOKO-WORKDIR-2`** (OPEN, asked at iteration 21, unanswered): grant **standing** authorization to
  reconcile the source clone to `origin/dev` unattended when three measured predicates hold? One word:
  **yes** or **no**. Without it this returns as an ask roughly four times per nine days.
