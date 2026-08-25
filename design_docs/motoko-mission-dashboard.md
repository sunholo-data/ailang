# Mission Dashboard — Motoko

_Snapshot after iteration 23 (2026-08-25). Overwritten each iteration; history lives in the charter STATUS block and the log._

**Release**: AILANG v0.33.2 · anchor `sunholo-data/ailang` (shared with V1; V1 owns dev CI red on it)

## In flight / next
- **Just landed** — row **6f**: triage-lite of the two motoko-owned sweep issues. `#839` (`std/net` ignores
  proxy env) **CLOSED** — a version skew: the reporter's `v0.33.0` binary predates the fix (`e5ee6c5e5`,
  PR #613) by 16 days; closed against CI-run regression tests, not bookkeeping. `#842` **CONFIRMED REAL**
  and re-filed as row 6h.
- **Next** — row **6g**: `run_bounded` *and* production `run_lane` kill the wrapper PID, not the process
  group, so a hung grandchild survives at `PPID 1`.
- **Then** — row **6h** (new): a provider failure parses as a successful empty completion, and the guard the
  reporter suggests is **not expressible** — `ChatStepResponse.Usage` is a value type, so
  `absent.Usage == zeroed.Usage`. Reaches the **ollama `/v1` lane**, i.e. our own eval rig.
- **Then** — row 7 (profile restoration design), row 8 (repin stale OpenRouter models).

## Gated / parked
- Phase 0 remains **CLOSED**: `arniwesth/motoko_agent#154` still OPEN (control `#175` MERGED), and G5 needs
  Arni's ABI-settled word. Rows 10/11/12 stay parked. Rows 9/13/14 wait on a green tree.

## Loop health / routing
- Controller opus · **no designer/planner/executor/evaluator spawned** (triage-lite row names its own
  procedure) · designer rotation untouched at fable, **unspent**.
- Metered **$0.00** of $5. No GPU, no `rig.lock`. Gates run on darwin/arm64 only.
- **dev CI RED and handed to V1**: `test` job on `02bf43668` failed at *Download all Go modules*
  (`proxy.golang.org` stream error) on V1's docs-only record commit; re-run fired, not our pick.
- Source clone `~/dev/sunholo-data/ailang-motoko`: **35 behind / 0 ahead**, clean — crossed the notice
  threshold (25) exactly as `D-MOTOKO-WORKDIR-2` predicted, up from 24 one iteration ago.

## Parked on Mark
- **`D-MOTOKO-WORKDIR-2`** (OPEN since iteration 21, unanswered — this is the second ask): grant **standing**
  authorization to reconcile the source clone to `origin/dev` unattended when three measured predicates hold?
  One word: **yes** or **no**. Without it this returns roughly four times per nine days, each time resolving
  to the same word for the same mechanical operation.
