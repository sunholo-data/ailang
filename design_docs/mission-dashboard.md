# Mission Dashboard — the 30-second control context
> **Contract**: ≤40 lines, overwritten by mission-control Gate 4 every iteration (history lives in
> the charter/log). Fresh session = THIS + MEMORY.md. Humans steer via the bookkeeping issue.

**Updated**: 2026-08-11 ~20:15 local (iteration 178)

## Now
- **v0.33.0** · `origin/dev` `5f471b2b7` — **`#619` (P0) PICKED AND PARKED at the quorum gate.**
  2 rounds, both BLOCKED, **both reviewers present** (no N−1 hole). Metered **$0.0955** of $5.
  No code, no sprint, no PR — the deliverable is the reality-check + doc revision.
- 🔴 **The doc named the wrong file.** `cmd/ailang/eval_publish.go` never aggregates raw rows
  (it sums rotation `summary.json`). The real numerator is **`SummarizeRotation`**
  (`rotation_summary.go:246`) — **0** `IsValid()` reads, control `PassRate`=8 — **plus a second**
  at `ModelRollupStats.PassAt1`. A sprint off the doc as written would have shipped a **no-op**.
- ✅ **A third of the scope is already done** (`--skip-existing`, `f3189541a`); `FilterValidResults`
  **cannot** be reused (`go list -deps`: import cycle). **160** invalid rows live in the bank
  (control firing), and the OS board has **no denominator field** — surfacing it is a schema change.
- 💡 **Both blocking objections were TRUE and understated.** R1: the coverage gate compares
  benchmark **counts, never set identity** → new **W9**; its "50%" was quoting our own stale doc
  (code: RATE 0.5 / **ELO 0.9**) → **AC1 now marked NOT SATISFIED**. R2: `null` unmarshals to a
  silent `0.0` in **3** independent `PassRate float64` decls + 14 `jq` consumers.
- 🔧 **Archive repaired**: iter-177 put stamp **174** at the *bottom* of a newest-first file.

## In flight / queued
- **`#618` rollout** (cp plists → `launchctl load` → *then* `unsetenv`) — human-sequenced, `D-8`.
- Batch while `D-9` is open: **`#616`** → `#617`. **#636** `[world-DEMAND]` · **#613** `D-1` ·
  **#604**/`#614` `D-2` · **#649** local-model gap · **#651** quorum zero-signal · **#654**.

## Next
**`D-9` gates `#619`.** Split → W8 gets its own scoped doc, one re-quorum, route (the
reality-check is already banked). Hold → iteration 179 takes **`#616`** (NEW-DOC needed).

## Loop + routing
Controller **opus** · designer/planner/executor/evaluator **all NOT fired** (parked pre-routing;
rotation pointer unchanged, next `codex:gpt-5.6-sol`).

## PARKED ON MARK — #635
- **`D-9`** (NEW, iter-178): quorum reviews a 5-item umbrella while only **W8** is routed, so it
  blocks on content W8 doesn't touch. **(A)** split W8 into its own doc + re-quorum · **(B)** hold.
- **`D-1`** `#613` SSRF · **`D-2`** `#604`/`#614` · **`D-7`** codex **2/2** → (B) de facto · **`D-8`** authorise the `#618` rig rollout.
